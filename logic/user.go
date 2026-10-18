package logic

import (
	"errors"
	"time"
	"webapp/dao/mysql"
	"webapp/dao/redis"
	"webapp/models"
	"webapp/pkg/jwt"
	"webapp/pkg/snowflake"

	jt "github.com/golang-jwt/jwt/v4"
	"go.uber.org/zap"
)

func SignUp(p *models.ParamSigUp) (err error) {
	err = mysql.CheckUserExist(p.Username)
	if err != nil {
		return
	}

	userID := snowflake.GenID()
	u := &models.User{
		UserID:   userID,
		Username: p.Username,
		Password: p.Password,
	}

	if err = mysql.InsertUser(u); err != nil {
		return err
	}
	return
}

func Login(p *models.ParamLogin) (u *models.User, atoken, rtoken string, err error) {
	u = &models.User{
		Username: p.Username,
		Password: p.Password,
	}
	if err := mysql.Login(u); err != nil {
		return nil, "", "", err
	}

	// 生成token
	atoken, rtoken, err = jwt.GenToken(u.UserID, u.Username)
	if err != nil {
		return nil, "", "", err
	}

	l := &models.ParamPair{
		UserID: u.UserID,
		Token:  atoken,
	}

	err = StoreUToken(l, rtoken)
	if err != nil {
		zap.L().Error("logic.StoreUToken failed", zap.Error(err))
		return u, atoken, rtoken, err
	}
	return
}

func StoreUToken(p *models.ParamPair, rtoken string) (err error) {
	rclaims := new(jwt.RClaims)

	_, err = jt.ParseWithClaims(rtoken, rclaims, Keyfunc)
	if err != nil {
		return err
	}

	if rclaims.ExpiresAt == nil ||
		rclaims.Issuer != "zzed" ||
		rclaims.TokenID <= 0 {
		return errors.New("无效的刷新令牌")
	}

	// 到 refresh_token 过期还剩多久
	ttl := time.Until(rclaims.ExpiresAt.Time)
	if ttl <= 0 {
		return errors.New("刷新令牌已过期")
	}

	pair := &models.Pair{
		UserID: p.UserID,
		Token:  p.Token,
	}
	if err = redis.StoreToken(pair, ttl); err != nil {
		return
	}
	return
}

func GetUToken(p *models.ParamPair) (result bool, err error) {
	var v string
	v, err = redis.GetToken(p.UserID)
	if err != nil {
		return
	}
	if v != p.Token {
		return false, nil
	}
	return true, nil
}

func Keyfunc(token *jt.Token) (interface{}, error) {
	if token.Method.Alg() != jt.SigningMethodHS256.Alg() {
		return nil, errors.New("不支持的签名算法")
	}
	return jwt.MySecret, nil
}

func RefreshToken(aToken, rToken string) (newaToken, newrToken string, err error) {
	if _, err = jt.Parse(rToken, Keyfunc); err != nil {
		return
	}

	rclaims := new(jwt.RClaims)

	_, err = jt.ParseWithClaims(rToken, rclaims, Keyfunc)
	if err != nil {
		return
	}

	if rclaims.ExpiresAt == nil || rclaims.TokenID <= 0 || rclaims.Issuer != "zzed" {
		return "", "", errors.New("无效的刷新令牌")
	}

	aclaims := new(jwt.AClaims)
	_, err = jt.ParseWithClaims(aToken, aclaims, Keyfunc)

	if aclaims.RefreshID != rclaims.TokenID {
		return "", "", errors.New("两个令牌不匹配")
	}

	p := &models.ParamPair{
		UserID: aclaims.UserID,
		Token:  aToken,
	}

	ifsame, errs := GetUToken(p)

	if errs != nil {
		return "", "", errors.New("logic.GetUToken failed")
	}

	if !ifsame {
		return "", "", errors.New("用户已登录")
	}

	if err == nil {
		return aToken, rToken, nil
	}

	v, _ := err.(*jt.ValidationError)

	if v.Errors == jt.ValidationErrorExpired {

		atoken, rtoken, err := jwt.GenToken(aclaims.UserID, aclaims.Username)
		if err != nil {
			return atoken, rtoken, err
		}
		p.Token = atoken
		err = StoreUToken(p, rtoken)
		if err != nil {
			zap.L().Error("logic.StoreUToken failed", zap.Error(err))
			return atoken, rtoken, err
		}
		return atoken, rtoken, err
	}
	return

}
