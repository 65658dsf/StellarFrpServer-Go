package handler

import (
	"context"
	"crypto/hmac"
	"crypto/rc4"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"stellarfrp/config"
	"stellarfrp/internal/repository"
	"stellarfrp/pkg/logger"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// RealNameAuthHandler 实名认证处理器
type RealNameAuthHandler struct {
	cfg      *config.Config
	userRepo repository.UserRepository
	logger   *logger.Logger
}

// NewRealNameAuthHandler 创建实名认证处理器实例
func NewRealNameAuthHandler(cfg *config.Config, userRepo repository.UserRepository, logger *logger.Logger) *RealNameAuthHandler {
	return &RealNameAuthHandler{
		cfg:      cfg,
		userRepo: userRepo,
		logger:   logger,
	}
}

// RealNameAuthRequest 实名认证请求结构体
type RealNameAuthRequest struct {
	LotNumber     string `json:"lot_number" binding:"required"`
	CaptchaOutput string `json:"captcha_output" binding:"required"`
	PassToken     string `json:"pass_token" binding:"required"`
	GenTime       string `json:"gen_time" binding:"required"`
	IDNo          string `json:"idNo" binding:"required"`
	Name          string `json:"name" binding:"required"`
}

// verifyGeetestCaptcha 验证极验验证码
func (h *RealNameAuthHandler) verifyGeetestCaptcha(lotNumber, captchaOutput, passToken, genTime string) (bool, string) {
	signToken := hmac.New(sha256.New, []byte(h.cfg.Geetest.CaptchaKey))
	signToken.Write([]byte(lotNumber))
	calculatedSignToken := hex.EncodeToString(signToken.Sum(nil))

	params := url.Values{}
	params.Set("lot_number", lotNumber)
	params.Set("captcha_output", captchaOutput)
	params.Set("pass_token", passToken)
	params.Set("gen_time", genTime)
	params.Set("sign_token", calculatedSignToken)

	validateURL := fmt.Sprintf("%s/validate?captcha_id=%s", h.cfg.Geetest.APIServer, h.cfg.Geetest.CaptchaID)

	h.logger.Debug("Geetest validation request", "url", validateURL, "params", params.Encode())

	resp, err := http.PostForm(validateURL, params)
	if err != nil {
		h.logger.Error("Geetest request failed", "error", err)
		return false, "验证码服务器请求失败"
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
		h.logger.Error("Geetest server response error", "status_code", resp.StatusCode, "body", string(bodyBytes))
		return false, fmt.Sprintf("验证码服务器响应异常: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		h.logger.Error("Failed to decode geetest response", "error", err)
		return false, "解析验证码响应失败"
	}

	h.logger.Debug("Geetest validation response", "result", result)

	if status, ok := result["status"].(string); ok && status == "success" {
		if r, ok := result["result"].(string); ok && r == "success" {
			return true, ""
		}
	}
	reason, _ := result["reason"].(string)
	if reason == "" {
		reason = "验证失败"
	}
	return false, reason
}

// rc4Encrypt RC4加密
func rc4Encrypt(data string, key string) (string, error) {
	c, err := rc4.NewCipher([]byte(key))
	if err != nil {
		return "", err
	}
	dst := make([]byte, len(data))
	c.XORKeyStream(dst, []byte(data))
	return hex.EncodeToString(dst), nil
}

// RealNameAuth 实名认证接口
func (h *RealNameAuthHandler) RealNameAuth(c *gin.Context) {
	var req RealNameAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "参数错误: " + err.Error()})
		return
	}

	// 1. 从Gin Context获取用户ID (由UserAuth中间件设置)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusOK, gin.H{"code": 401, "msg": "无法获取用户信息，请检查Token是否有效或认证中间件配置"})
		return
	}
	userIDInt64, ok := userID.(int64)
	if !ok {
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "用户ID格式错误"})
		return
	}

	// 根据用户ID获取用户信息
	user, err := h.userRepo.GetByID(context.Background(), userIDInt64)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 401, "msg": "无效的用户或Token"}) // 或者更具体的错误，如 "用户不存在"
		return
	}

	// 2. 验证极验验证码
	isValidCaptcha, captchaMsg := h.verifyGeetestCaptcha(req.LotNumber, req.CaptchaOutput, req.PassToken, req.GenTime)
	if !isValidCaptcha {
		c.JSON(http.StatusOK, gin.H{"code": 401, "msg": fmt.Sprintf("验证码验证失败: %s", captchaMsg)})
		return
	}

	// 3. 检查是否已经实名认证
	if user.IsVerified == 1 {
		c.JSON(http.StatusOK, gin.H{"code": 409, "msg": "您已完成实名认证"})
		return
	}

	// 4. 验证身份证号格式和年龄 (与Python逻辑一致)
	if len(req.IDNo) != 18 {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "身份证号格式错误"})
		return
	}
	birthYear, err := strconv.Atoi(req.IDNo[6:10])
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "身份证号格式错误"})
		return
	}
	currentYear := time.Now().Year()
	age := currentYear - birthYear
	if age < 6 || age > 55 {
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": "年龄必须在6-55岁之间"})
		return
	}

	// 5. 检查用户是否有可用的认证次数
	// 注意：Python代码中的 auth_count 对应这里的 VerifyCount
	// Python中是 > 0 可用，这里假设 verify_count >= 0, 如果为0则没有次数
	// 根据您的 users 表结构, verify_count unsigned NOT NULL DEFAULT '0'，所以应该是 > 0
	if user.VerifyCount <= 0 {
		c.JSON(http.StatusOK, gin.H{"code": 403, "msg": "没有可用的认证次数，请先购买认证次数"})
		return
	}

	// 6. 调用阿里云实名认证API
	aliCloudHost := h.cfg.AliCloud.Host
	aliCloudPath := h.cfg.AliCloud.Path
	aliCloudAppCode := h.cfg.AliCloud.AppCode

	apiURL := aliCloudHost + aliCloudPath
	payload := url.Values{}
	payload.Set("idNo", req.IDNo)
	payload.Set("name", req.Name)

	httpClient := &http.Client{}
	httpReq, err := http.NewRequest("POST", apiURL, strings.NewReader(payload.Encode()))
	if err != nil {
		h.logger.Error("创建阿里云请求失败", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "认证服务请求失败"})
		return
	}
	httpReq.Header.Add("Authorization", "APPCODE "+aliCloudAppCode)
	httpReq.Header.Add("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")

	h.logger.Debug("AliCloud Auth Request", "url", apiURL, "payload", payload.Encode())

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		h.logger.Error("请求阿里云API失败", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "认证服务请求失败"})
		return
	}
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		h.logger.Error("读取阿里云API响应失败", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "读取认证服务响应失败"})
		return
	}

	h.logger.Debug("AliCloud Auth Response", "status_code", resp.StatusCode, "body", string(bodyBytes))

	var aliResult map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &aliResult); err != nil {
		h.logger.Error("解析阿里云API响应失败", "error", err, "body", string(bodyBytes))
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "解析认证服务响应失败"})
		return
	}

	respCode, _ := aliResult["respCode"].(string)

	// 开始数据库事务
	tx, err := h.userRepo.(repository.TransactionalUserRepository).BeginTx(context.Background())
	if err != nil {
		h.logger.Error("开始数据库事务失败", "error", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "服务器内部错误"})
		return
	}
	// 使用带有事务的 userRepo
	userRepoTx := h.userRepo.(repository.TransactionalUserRepository).WithTx(tx)

	if respCode == "0000" { // 认证成功
		verifyDate := time.Now().Format("2006-01-02 15:04:05")
		idInfo := fmt.Sprintf("%s|%s|%s", req.Name, req.IDNo, verifyDate)
		encryptedInfo, err := rc4Encrypt(idInfo, h.cfg.AliCloud.IdentityKey)
		if err != nil {
			h.logger.Error("RC4加密失败", "error", err)
			tx.Rollback()
			c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "信息加密失败"})
			return
		}
		user.VerifyInfo = sql.NullString{String: encryptedInfo, Valid: true}
		user.IsVerified = 1
		user.VerifyCount = user.VerifyCount - 1 // 认证成功，次数减1

		// 如果用户组ID是1，则修改为2
		if user.GroupID == 1 {
			user.GroupID = 2
		}

		if err := userRepoTx.Update(context.Background(), user); err != nil {
			h.logger.Error("更新用户信息失败 (认证成功)", "error", err, "username", user.Username)
			tx.Rollback()
			c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "更新用户信息失败"})
			return
		}
		if err := tx.Commit(); err != nil {
			h.logger.Error("提交数据库事务失败 (认证成功)", "error", err)
			c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "服务器内部错误"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "实名认证成功"})
	} else { // 认证失败
		user.VerifyCount = user.VerifyCount - 1 // 认证失败，次数也减1
		if user.VerifyCount < 0 {               // 防止变为负数，虽然理论上前面有判断
			user.VerifyCount = 0
		}
		if err := userRepoTx.Update(context.Background(), user); err != nil {
			h.logger.Error("更新用户信息失败 (认证失败)", "error", err, "username", user.Username)
			tx.Rollback()
			c.JSON(http.StatusOK, gin.H{"code": 500, "msg": "更新用户信息失败"})
			return
		}
		if err := tx.Commit(); err != nil {
			h.logger.Error("提交数据库事务失败 (认证失败)", "error", err)
			// 即使提交失败，也返回认证接口的错误信息
		}

		errorMessages := map[string]string{
			"0001": "开户名不能为空",
			"0002": "开户名不能包含特殊字符",
			"0003": "身份证号不能为空",
			"0004": "身份证号格式错误",
			"0007": "该身份证号码不存在",
			"0008": "身份证信息不匹配",
			"0010": "系统维护，请稍后再试",
		}
		msg, found := errorMessages[respCode]
		if !found {
			msg = "实名认证失败，未知错误码: " + respCode
			// 记录一下未知的错误码
			if errMsg, ok := aliResult["respMsg"].(string); ok {
				h.logger.Warn("AliCloud Auth Failed with unknown respCode", "respCode", respCode, "respMsg", errMsg)
			} else {
				h.logger.Warn("AliCloud Auth Failed with unknown respCode and no respMsg", "respCode", respCode)
			}
		}
		c.JSON(http.StatusOK, gin.H{"code": 400, "msg": msg})
	}
}

// Ping 用于测试路由是否可达
func (h *RealNameAuthHandler) Ping(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "pong from realname auth"})
}
