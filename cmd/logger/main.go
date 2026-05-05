package main

import (
	"go-reinvent/pkg/logger"
	"os"
)

func main() {
	// 示例1：默认配置 —— 彩色输出到终端，DEBUG 级别，全量日志
	log := logger.New()
	log.Debug("starting application")
	log.Infof("server listening on port %d", 8080)
	log.Warn("config file not found, using defaults")
	log.Error("failed to connect to database")

	// 示例2：文件输出 —— 关闭颜色，只记录 INFO 及以上，附带调用位置
	f, _ := os.Create("app.log")
	defer f.Close()
	fileLog := logger.New(
		logger.WithOutput(f),         // 输出到文件
		logger.WithColorful(false),   // 文件不需要 ANSI 颜色码
		logger.WithLevel(logger.INFO), // 只记录 INFO 及以上
		logger.WithCaller(true),      // 显示文件名:行号
	)
	fileLog.Info("this goes to file without color codes")
	fileLog.Debugf("this will not appear (below INFO)")
	fileLog.Errorf("connection refused: %s", "127.0.0.1:5432")
}
