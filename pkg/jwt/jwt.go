// Package jwt 封装 JWT 签发与校验。
package jwt

import (
	"errors"
	"time"

	jwtlib "github.com/golang-jwt/jwt/v5"
)

// Claims 自定义声明。
type Claims struct {
	UserID uint64 `json:"uid"`
	Role   int8   `json:"role"`
	jwtlib.RegisteredClaims
}

// Manager JWT 管理器。
type Manager struct {
	secret []byte
	ttl    time.Duration
}

// New 创建 JWT 管理器。
func New(secret string, ttl time.Duration) *Manager {
	return &Manager{secret: []byte(secret), ttl: ttl}
}

// TTL 令牌有效期。
func (m *Manager) TTL() time.Duration { return m.ttl }

// Generate 为指定用户签发 token。
func (m *Manager) Generate(userID uint64, role int8) (string, error) {
	now := time.Now()
	claims := &Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwtlib.RegisteredClaims{
			Subject:   "novadrive",
			IssuedAt:  jwtlib.NewNumericDate(now),
			NotBefore: jwtlib.NewNumericDate(now.Add(-time.Minute)),
			ExpiresAt: jwtlib.NewNumericDate(now.Add(m.ttl)),
		},
	}
	token := jwtlib.NewWithClaims(jwtlib.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// Parse 校验并解析 token。
func (m *Manager) Parse(token string) (*Claims, error) {
	if token == "" {
		return nil, errors.New("token 为空")
	}
	parsed, err := jwtlib.ParseWithClaims(token, &Claims{}, func(t *jwtlib.Token) (any, error) {
		if _, ok := t.Method.(*jwtlib.SigningMethodHMAC); !ok {
			return nil, errors.New("非法的签名算法")
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return nil, errors.New("非法的 token")
	}
	return claims, nil
}

// Secret 返回内部密钥，供分享访问签名等场景复用。
func (m *Manager) Secret() []byte { return m.secret }
