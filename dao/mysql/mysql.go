package mysql

import (
	"fmt"
	"webapp/settings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

var db *sqlx.DB

func Init(cfg *settings.MySQLConfig) (err error) {
	// host := viper.GetString("mysql.host")
	// port := viper.GetInt("mysql.port")
	// user := viper.GetString("mysql.user")
	// password := viper.GetString("mysql.password")
	// dbname := viper.GetString("mysql.dbname")
	// dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", user, password, host, port, dbname)
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?multiStatements=true&parseTime=true&loc=Local", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DbName)
	db, err = sqlx.Connect("mysql", dsn)
	if err != nil {
		zap.L().Error("connect DB failed", zap.Error(err))
		return
	}
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	// db.SetMaxIdleConns(viper.GetInt("mysql.max_open_conns"))
	// db.SetMaxOpenConns(viper.GetInt("mysql.max_idel_conns"))
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	return
}

func Close() {
	_ = db.Close()
}
