package main

import (
	"fmt"
	"go-reinvent/pkg/logger"
	"go-reinvent/pkg/rotatelog"
	"time"
)

func main() {
	// 示例 1：按大小轮转
	// 单文件超过 1MB 自动轮转，保留 3 个备份，保留 7 天
	sizeWriter := rotatelog.New("logs/size.log",
		rotatelog.WithMaxSize(1), // 1MB，demo 中写入足够数据触发轮转
		rotatelog.WithMaxBackups(3),
		rotatelog.WithMaxAge(7),
	)
	defer sizeWriter.Close()

	log := logger.New(
		logger.WithOutput(sizeWriter),
		logger.WithColorful(false),
		logger.WithLevel(logger.INFO),
	)

	log.Info("=== 示例 1：按大小轮转（1MB 阈值）===")
	for i := 0; i < 20000; i++ {
		log.Infof("第 %d 条日志消息，填充数据以触发大小轮转", i)
	}
	log.Info("按大小轮转示例结束，请查看 logs/size.log 和 logs/size.1.log 等文件")

	// 示例 2：按日期轮转 + 手动 Rotate
	// 日期轮转需要跨天才自动触发，这里用手动 Rotate() 演示效果
	dateWriter := rotatelog.New("logs/date.log",
		rotatelog.WithRotateByDate("daily"),
		rotatelog.WithMaxBackups(10),
	)
	defer dateWriter.Close()

	dateLog := logger.New(
		logger.WithOutput(dateWriter),
		logger.WithColorful(false),
		logger.WithLevel(logger.DEBUG),
	)

	dateLog.Info("=== 示例 2：按日期轮转 ===")
	dateLog.Info("日期轮转需要跨天才自动触发，下面用手动 Rotate() 演示")
	dateLog.Debug("这是一条调试日志")
	dateLog.Warn("这是一条警告日志")

	// 手动触发轮转，模拟日期变化
	if err := dateWriter.Rotate(); err != nil {
		fmt.Printf("手动轮转失败: %v\n", err)
	}
	dateLog.Info("轮转后的新日志，旧日志已归档为 logs/date.YYYY-MM-DD.log 格式")

	// 示例 3：按日期 + 大小组合轮转 + gzip 压缩
	combinedWriter := rotatelog.New("logs/combined.log",
		rotatelog.WithRotateByDate("daily"),
		rotatelog.WithMaxSize(1),
		rotatelog.WithMaxBackups(5),
		rotatelog.WithCompress(true),
	)
	defer combinedWriter.Close()

	combinedLog := logger.New(
		logger.WithOutput(combinedWriter),
		logger.WithColorful(false),
		logger.WithLevel(logger.INFO),
	)

	combinedLog.Info("=== 示例 3：日期+大小组合轮转 + gzip 压缩（1MB 阈值）===")
	for i := 0; i < 20000; i++ {
		combinedLog.Infof("组合模式日志 %d - %s，填充数据以触发轮转", i, time.Now().Format(time.RFC3339))
	}
	combinedLog.Info("组合轮转示例结束，请查看 logs/ 目录下的 .gz 文件")

	fmt.Println("\n所有示例执行完成，请查看 logs/ 目录")
}
