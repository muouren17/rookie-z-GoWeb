package main

// @title GoWeb项目
// @version 1.0
// @description 简单的Go练手项目

// @contact.name zed
// @contact.email zed@163.com

// @host localhost:8081
// @BasePath /
// @schemes http
//
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization
// @description 使用访问令牌，值为 Bearer 加一个空格再加 access token；登录响应字段为 data.accessTokne。

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
	"webapp/controllers"
	"webapp/dao/mysql"
	"webapp/dao/redis"
	"webapp/jobs"
	"webapp/logger"
	"webapp/pkg/jwt"
	"webapp/pkg/snowflake"
	"webapp/routes"
	"webapp/settings"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func main() {
	if err := settings.Init(); err != nil {
		fmt.Printf("init settings failed , err:%v\n", err)
		return
	}

	if err := jwt.Init(viper.GetString("secret_key")); err != nil {
		fmt.Printf("init jwt failed: %v\n", err)
		return
	}

	if err := logger.Init(settings.Conf.LogConfig, settings.Conf.Mode); err != nil {
		fmt.Printf("init logger failed , err:%v\n", err)
	}
	zap.L().Debug("logger init success....")
	defer zap.L().Sync()
	if err := controllers.InitTrans("zh"); err != nil {
		fmt.Printf("init trans failed , err:%v\n", err)
		return
	}
	if err := mysql.Init(settings.Conf.MySQLConfig); err != nil {
		fmt.Printf("init mysql failed , err:%v\n", err)
		return
	}
	defer mysql.Close()

	if err := mysql.CreateTables(); err != nil {
		fmt.Printf("create tables failed, err:%v\n", err)
		return
	}

	cancel, err := redis.Init(settings.Conf.RedisConfig)
	if err != nil {
		fmt.Printf("init redis failed , err:%v\n", err)
		return
	}
	defer cancel()
	defer redis.Close()

	jobsctx, stopjobs := context.WithCancel(context.Background())

	var jobWG sync.WaitGroup
	jobWG.Add(1)

	go func() {
		defer jobWG.Done()
		jobs.Vote_archive(jobsctx)
	}()

	defer func() {
		stopjobs()
		jobWG.Wait()
	}()

	if err := snowflake.Init(settings.Conf.StartTime, settings.Conf.MachineID); err != nil {
		fmt.Printf("init snowid failed , err:%v\n", err)
		return
	}

	r := routes.Setup(settings.Conf.Mode)
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", settings.Conf.Port),
		Handler: r,
	}

	go func() {
		// 开启一个goroutine启动服务
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zap.L().Fatal("listen: %s\n", zap.Error(err))
		}
	}()

	// 等待中断信号来优雅地关闭服务器，为关闭服务器操作设置一个5秒的超时
	quit := make(chan os.Signal, 1) // 创建一个接收信号的通道
	// kill 默认会发送 syscall.SIGTERM 信号
	// kill -2 发送 syscall.SIGINT 信号，我们常用的Ctrl+C就是触发系统SIGINT信号
	// kill -9 发送 syscall.SIGKILL 信号，但是不能被捕获，所以不需要添加它
	// signal.Notify把收到的 syscall.SIGINT或syscall.SIGTERM 信号转发给quit
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM) // 此处不会阻塞
	<-quit                                               // 阻塞在此，当接收到上述两种信号时才会往下执行
	zap.L().Info("Shutdown Server ...")
	// 创建一个5秒超时的context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// 5秒内优雅关闭服务（将未处理完的请求处理完再关闭服务），超过5秒就超时退出
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server Shutdown: ", zap.Error(err))
	}

	zap.L().Info("Server exiting")

}
