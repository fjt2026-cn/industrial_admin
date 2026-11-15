package utils

import (
	"errors"
	"fmt"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"

	"industrial_admin/config"
)

// 令牌解析的几类失败原因，便于上层区分提示文案
var (
	ErrTokenEmpty   = errors.New("令牌为空")
	ErrTokenInvalid = errors.New("令牌无效")
	ErrTokenExpired = errors.New("令牌已过期")
)

// Claims 是自定义载荷。
// 只放身份标识（用户 ID、账号），不放手机号、角色明细等可被解码读取的信息；
// RegisteredClaims 提供标准字段（签发方、签发时间、过期时间）。
type Claims struct {
	UserID   uint64 `json:"uid"`
	Username string `json:"uname"`
	jwt.RegisteredClaims
}

// GenerateToken 为登录成功的用户签发令牌。
// 返回：令牌串、过期时间（毫秒，供前端倒计时或自动刷新使用）。
// 私钥来自环境变量 JWT_SECRET，泄露即等于所有账号可被伪造，因此禁止写入代码库。
func GenerateToken(cfg *config.Config, userID uint64, username string) (string, int64, error) {
	now := time.Now()
	exp := now.Add(cfg.JWT.Expire) // 有效期由 JWT_EXPIRE_HOURS 控制，默认 2 小时

	claims := Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    cfg.JWT.Issuer,          // 签发方，解析时可校验来源
			IssuedAt:  jwt.NewNumericDate(now), // 签发时间
			ExpiresAt: jwt.NewNumericDate(exp), // 过期时间，由库自动校验
		},
	}

	// 使用 HS256（对称加密），密钥必须在服务端保管
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(cfg.JWT.Secret))
	if err != nil {
		return "", 0, err
	}
	return signed, exp.UnixMilli(), nil
}

// ParseToken 校验令牌并取出载荷。
// 校验项：签名是否被篡改、是否已过期、签名算法是否为预期的 HMAC。
// 只要返回错误，调用方一律按未登录处理（HTTP 401），不要把具体原因回传给前端。
func ParseToken(cfg *config.Config, tokenStr string) (*Claims, error) {
	if tokenStr == "" {
		return nil, ErrTokenEmpty
	}

	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		// 关键防护：只接受 HMAC 系列算法，防止攻击者把 alg 改成 none 绕过签名
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("非预期的签名算法: %v", t.Header["alg"])
		}
		return []byte(cfg.JWT.Secret), nil
	})
	if err != nil {
		// 过期属于可预期分支，单独归类，方便前端提示“登录已失效，请重新登录”
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}
	return claims, nil
}
