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
	"time"
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
	Status    string `json:"status"`
	Code      string `json:"code"`
	Msg       string `json:"msg"`
	Result    string `json:"result"`
	CaptchaID string `json:"captcha_id"`
	LotNumber string `json:"lot_number"`
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
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	signToken := c.generateSignToken(params.LotNumber, timestamp)

	// 构建请求URL
	apiURL := fmt.Sprintf("%s/validate", c.APIServer)

	// 构建请求体
	data := url.Values{}
	data.Set("lot_number", params.LotNumber)
	data.Set("captcha_output", params.CaptchaOutput)
	data.Set("pass_token", params.PassToken)
	data.Set("gen_time", params.GenTime)

	// 发送请求
	req, err := http.NewRequest("POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return false, err
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Captcha-ID", c.CaptchaID)
	req.Header.Set("Timestamp", timestamp)
	req.Header.Set("Signature", signToken)

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
	if verifyResp.Status == "success" && verifyResp.Result == "success" {
		return true, nil
	}

	return false, errors.New(verifyResp.Msg)
}

// generateSignToken 生成签名令牌
func (c *GeetestClient) generateSignToken(lotNumber, timestamp string) string {
	// 构建签名原文
	signStr := fmt.Sprintf("%s%s%s", lotNumber, c.CaptchaID, timestamp)

	// 使用HMAC-SHA256算法计算签名
	h := hmac.New(sha256.New, []byte(c.CaptchaKey))
	h.Write([]byte(signStr))

	// 返回十六进制签名
	return hex.EncodeToString(h.Sum(nil))
}
