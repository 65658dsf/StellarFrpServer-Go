package geetest

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// GeetestClient 极验验证客户端
type GeetestClient struct {
	CaptchaID  string
	CaptchaKey string
	APIServer  string
}

// NewGeetestClient 创建极验验证客户端
func NewGeetestClient(captchaID, captchaKey, apiServer string) *GeetestClient {
	return &GeetestClient{
		CaptchaID:  captchaID,
		CaptchaKey: captchaKey,
		APIServer:  apiServer,
	}
}

// VerifyResponse 验证响应
type VerifyResponse struct {
	Status      string                 `json:"status"`
	Code        string                 `json:"code"`
	Msg         string                 `json:"msg"`
	Result      string                 `json:"result"`
	Reason      string                 `json:"reason"`
	CaptchaID   string                 `json:"captcha_id"`
	LotNumber   string                 `json:"lot_number"`
	CaptchaArgs map[string]interface{} `json:"captcha_args"`
}

// VerifyParams 验证参数
type VerifyParams struct {
	LotNumber     string `json:"lot_number"`
	CaptchaOutput string `json:"captcha_output"`
	PassToken     string `json:"pass_token"`
	GenTime       string `json:"gen_time"`
}

// Verify 验证极验验证码
func (c *GeetestClient) Verify(params VerifyParams) (bool, error) {
	// 构建签名
	signToken := c.generateSignToken(params.LotNumber)

	// 构建请求URL
	apiURL := fmt.Sprintf("%s/validate?captcha_id=%s", c.APIServer, c.CaptchaID)

	// 构建请求体
	data := url.Values{}
	data.Set("lot_number", params.LotNumber)
	data.Set("captcha_output", params.CaptchaOutput)
	data.Set("pass_token", params.PassToken)
	data.Set("gen_time", params.GenTime)
	data.Set("sign_token", signToken)

	// 发送请求
	req, err := http.NewRequest("POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return false, err
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	// 解析响应
	var verifyResp VerifyResponse
	if err := json.Unmarshal(body, &verifyResp); err != nil {
		return false, err
	}

	// 验证结果
	// 处理请求异常情况
	if verifyResp.Status == "error" {
		return false, errors.New(verifyResp.Msg)
	}

	// 验证结果
	if verifyResp.Status == "success" && verifyResp.Result == "success" {
		return true, nil
	}

	// 验证失败但请求成功
	return false, errors.New(verifyResp.Reason)
}

// generateSignToken 生成签名令牌
func (c *GeetestClient) generateSignToken(lotNumber string) string {
	// 使用HMAC-SHA256算法计算签名
	// 使用用户当前完成验证的流水号lot_number作为原始消息message，使用客户验证私钥作为key
	h := hmac.New(sha256.New, []byte(c.CaptchaKey))
	h.Write([]byte(lotNumber))

	// 返回十六进制签名
	return hex.EncodeToString(h.Sum(nil))
}
