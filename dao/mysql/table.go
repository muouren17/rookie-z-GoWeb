package mysql

import (
	"fmt"
	"os"
)

func CreateTables() error {
	content, err := os.ReadFile("models/create_table.sql")
	if err != nil {
		return fmt.Errorf("读取建表文件失败:%w\n", err)
	}
	_, err = db.Exec(string(content))
	if err != nil {
		return fmt.Errorf("执行建表SQL失败，%w\n", err)
	}
	return nil
}
