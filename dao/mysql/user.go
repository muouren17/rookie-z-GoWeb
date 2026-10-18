package mysql

import (
	"crypto/md5"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"strings"
	"webapp/models"

	"github.com/spf13/viper"
	"golang.org/x/crypto/bcrypt"
)

func CheckUserExist(username string) error {
	sqlstr := "select count(user_id) from user where username=?"
	var count int
	err := db.Get(&count, sqlstr, username)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrorUserExist
	}
	return nil
}

func Login(user *models.User) (err error) {
	oPassword := user.Password
	sqlstr := "select user_id,username,password from user where username=?"
	err = db.Get(user, sqlstr, user.Username)
	if err == sql.ErrNoRows {
		return ErrorUserNotExits
	}
	if err != nil {
		return
	}
	needsUpgrade, err := verifyPassword(user.Password, oPassword)
	if err != nil {
		return err
	}
	if !needsUpgrade {
		return nil
	}

	// 旧密码校验通过后再升级。条件更新避免覆盖并发修改后的密码。
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(oPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	result, err := db.Exec("update user set password=? where user_id=? and password=?", string(hashedPassword), user.UserID, user.Password)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed == 0 {
		// 另一请求可能已完成升级；重新校验当前值，不能继续相信旧值。
		var currentPassword string
		err := db.Get(&currentPassword, "select password from user where user_id=?", user.UserID)
		if err == sql.ErrNoRows {
			return ErrorUserNotExits
		}
		if err != nil {
			return err
		}
		if err := bcrypt.CompareHashAndPassword([]byte(currentPassword), []byte(oPassword)); err != nil {
			return ErrorInvalidPassword
		}
		user.Password = currentPassword
		return nil
	}
	user.Password = string(hashedPassword)
	return nil
}

// verifyPassword 返回是否需要将旧格式升级为 bcrypt。
func verifyPassword(storedPassword, password string) (bool, error) {
	// bcrypt 最多接受 72 字节；校验时同样拒绝超长输入。
	if len(password) > 72 {
		return false, ErrorInvalidPassword
	}
	if strings.HasPrefix(storedPassword, "$2") {
		if err := bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(password)); err != nil {
			return false, ErrorInvalidPassword
		}
		return false, nil
	}

	// 仅兼容原有格式，不用于新注册。旧账号迁移前，mysql.secret 必须保留原值。
	// 所有旧账号迁移完成后，可移除此分支及对应配置。
	legacySecret := viper.GetString("mysql.secret")
	if legacySecret == "" {
		return false, ErrorInvalidPassword
	}
	legacyPassword, err := hex.DecodeString(storedPassword)
	if err != nil {
		return false, ErrorInvalidPassword
	}
	digest := md5.Sum([]byte(legacySecret))
	expected := append([]byte(password), digest[:]...)
	if subtle.ConstantTimeCompare(legacyPassword, expected) != 1 {
		return false, ErrorInvalidPassword
	}
	return true, nil
}

func InsertUser(user *models.User) (err error) {
	// bcrypt 会为每个密码生成随机盐，数据库只保存哈希结果。
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	sqlstr := "insert into user(user_id,username,password) values(?,?,?) "
	_, err = db.Exec(sqlstr, user.UserID, user.Username, string(hashedPassword))
	return
}

func GetUserDetail(id int64) (user *models.User, err error) {
	user = new(models.User)
	sqlstr := "select user_id,username from user where user_id =?"
	err = db.Get(user, sqlstr, id)
	return
}
