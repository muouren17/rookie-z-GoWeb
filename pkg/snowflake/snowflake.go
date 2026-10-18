package snowflake

import (
	"fmt"
	"time"

	sf "github.com/bwmarrin/snowflake"
)

// 全局节点
var node *sf.Node

// Init 初始化雪花算法节点
// startTime: 起始时间，格式 "2006-01-02" (如 "2023-01-01")
// machineID: 机器ID，集群中每个节点的ID必须唯一 (0-1023)
func Init(startTime string, machineID int64) (err error) {
	var st time.Time
	// 解析起始时间
	st, err = time.Parse("2006-01-02", startTime)
	if err != nil {
		return fmt.Errorf("解析起始时间失败: %w", err)
	}

	// 设置雪花算法的 Epoch (毫秒级)
	sf.Epoch = st.UnixNano() / 1000000

	// 创建节点
	node, err = sf.NewNode(machineID)
	if err != nil {
		return fmt.Errorf("创建雪花节点失败: %w", err)
	}
	return nil
}

// GenID 生成并返回一个唯一的 int64 ID
func GenID() int64 {
	if node == nil {
		panic("雪花算法节点未初始化，请先调用 Init")
	}
	return node.Generate().Int64()
}
