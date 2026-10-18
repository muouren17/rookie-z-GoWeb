package jwt

import (
	"errors"
	"math/rand"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/spf13/viper"
)

var MySecret []byte

type AClaims struct {
	UserID    int64  `json:"user_id"`
	Username  string `json:"username"`
	RefreshID int64  `json:"refresh_id"`
	jwt.RegisteredClaims
}

type RClaims struct {
	TokenID int64 `json:"token_id"`
	jwt.RegisteredClaims
}

func Init(secret string) error {
	if len(secret) < 32 {
		return errors.New("secret_key 至少需要32字节")
	}
	MySecret = []byte(secret)
	return nil
}

// GenToken 生成JWT
func GenToken(userid int64, username string) (aToken, rToken string, err error) {
	// 创建一个我们自己的声明
	aclaims := &AClaims{
		UserID:   userid,
		Username: username, // 自定义字段
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(viper.GetInt("auth.jwt_expire")) * time.Hour)),
			Issuer:    "zed",
		},
	}
	rclaims := &RClaims{
		TokenID: rand.Int63(),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 20)),
			Issuer:    "zzed",
		},
	}
	aclaims.RefreshID = rclaims.TokenID
	// 使用指定的签名方法创建签名对象
	aToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, aclaims).SignedString(MySecret)
	rToken, err = jwt.NewWithClaims(jwt.SigningMethodHS256, rclaims).SignedString(MySecret)
	// 使用指定的secret签名并获得完整的编码后的字符串token
	return
}

// ParseToken 解析JWT
func ParseToken(tokenString string) (*AClaims, error) {
	// 解析token
	// 如果是自定义Claim结构体则需要使用 ParseWithClaims 方法
	u := new(AClaims)
	token, err := jwt.ParseWithClaims(tokenString, u, func(token *jwt.Token) (i interface{}, err error) {
		// 直接使用标准的Claim则可以直接使用Parse方法
		//token, err := jwt.Parse(tokenString, func(token *jwt.Token) (i interface{}, err error) {
		return MySecret, nil
	})
	if err != nil {
		return nil, err
	}
	// 对token对象中的Claim进行类型断言
	// if token.Valid { // 校验token
	// 	return u, nil
	// }
	// return nil, errors.New("invalid token")
	if !token.Valid || u.ExpiresAt == nil || u.Issuer != "zed" || u.UserID <= 0 {
		return nil, errors.New("无效的访问令牌")
	}
	return u, nil
}
