package utils

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const (
	// BcryptCost 计算代价：2^10 次迭代，单次哈希约 10~50ms。
	// 值越大越难暴力破解，但会同步拖慢登录接口，课程项目取 10 即可。
	BcryptCost = 10

	// MinPasswordLength 口令最短长度，注册与重置口令共用同一规则。
	MinPasswordLength = 8

	// MaxPasswordBytes bcrypt 输入上限为 72 字节，超出部分会被静默截断，
	// 因此必须在入库前拦截，避免“超长口令只校验了前 72 字节”。
	MaxPasswordBytes = 72
)

// ValidatePassword 校验口令是否符合策略。
// 返回的错误文案可直接展示给用户，由控制器转成 40001 参数错误。
func ValidatePassword(plain string) error {
	if len(plain) < MinPasswordLength {
		return fmt.Errorf("口令长度不得少于 %d 位", MinPasswordLength)
	}
	if len([]byte(plain)) > MaxPasswordBytes {
		return fmt.Errorf("口令过长，不能超过 %d 字节", MaxPasswordBytes)
	}
	return nil
}

// HashPassword 把明文口令转换为 bcrypt 哈希。
// 返回的字符串形如 $2b$10$xxxxx，其中已内嵌算法版本、代价因子与随机盐，
// 因此数据库只需要一个 password_hash 字段，不必单独保存盐；
// 同一明文每次调用得到的哈希都不同，这是加盐的直接体现。
func HashPassword(plain string) (string, error) {
	if err := ValidatePassword(plain); err != nil {
		return "", err
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(plain), BcryptCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

// VerifyPassword 登录时比对明文口令与库中哈希。
// bcrypt 会从哈希串中取出盐重算摘要并做常量时间比较，因此：
//   - 不需要（也无法）手工加盐；
//   - 比较过程不因前缀匹配而提前返回，可抵御时序攻击；
//   - 返回 true 表示口令正确，false 表示口令错误（不区分“哈希损坏”与“口令错误”，
//     避免给攻击者提供额外信息）。
func VerifyPassword(hash, plain string) bool {
	if hash == "" || plain == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
