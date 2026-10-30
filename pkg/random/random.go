// Package random 提供 UUID、随机 token 与签名工具。
package random

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
)

// UUID 生成 v4 UUID。
func UUID() string {
	return uuid.NewString()
}

// Token 生成 n 字节密码学随机数的十六进制串（长度 2n）。
func Token(n int) (string, error) {
	if n <= 0 {
		n = 16
	}
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Password 生成 n 位可读的随机提取码（小写字母 + 数字）。
func Password(n int) (string, error) {
	const alphabet = "abcdefghijkmnpqrstuvwxyz23456789"
	if n <= 0 {
		n = 4
	}
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	out := make([]byte, n)
	for i := range out {
		out[i] = alphabet[int(b[i])%len(alphabet)]
	}
	return string(out), nil
}

// Sign 使用 HMAC-SHA256 对 payload 生成十六进制签名。
func Sign(secret, payload string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

// SignBase64 使用 HMAC-SHA256 生成 URL 安全的 Base64 签名。
func SignBase64(secret, payload string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// EqualSign 恒定时间比较签名。
func EqualSign(a, b string) bool {
	return hmac.Equal([]byte(a), []byte(b))
}

// Alias 生成短别名（用于日志追踪等），长度固定 8。
func Alias() string {
	s, err := Token(4)
	if err != nil {
		return fmt.Sprintf("%08x", 0)
	}
	return s
}
