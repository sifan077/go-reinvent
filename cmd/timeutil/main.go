package main

import (
	"fmt"
	"go-reinvent/pkg/timeutil"
	"time"
)

func main() {
	fmt.Println("=== 时间工具演示 ===")

	now := time.Now()

	// 格式化
	fmt.Println("\n--- 格式化 ---")
	fmt.Printf("自定义格式: %s\n", timeutil.Format(now, "YYYY-MM-DD HH:mm:ss"))
	fmt.Printf("日期: %s\n", timeutil.Format(now, "YYYY/MM/DD"))
	fmt.Printf("时间: %s\n", timeutil.Format(now, "HH:mm:ss"))

	// 解析
	fmt.Println("\n--- 解析 ---")
	t, _ := timeutil.Parse("2024-03-15 14:30:00", "YYYY-MM-DD HH:mm:ss")
	fmt.Printf("解析结果: %v\n", t)

	// 时间边界
	fmt.Println("\n--- 时间边界 ---")
	fmt.Printf("今天零点: %s\n", timeutil.Format(timeutil.StartOfDay(now), "HH:mm:ss"))
	fmt.Printf("今天最后: %s\n", timeutil.Format(timeutil.EndOfDay(now), "HH:mm:ss"))
	fmt.Printf("本月第一天: %s\n", timeutil.Format(timeutil.StartOfMonth(now), "YYYY-MM-DD"))
	fmt.Printf("本月最后一天: %d\n", timeutil.EndOfMonth(now).Day())

	// 友好时间
	fmt.Println("\n--- 友好时间 ---")
	fmt.Printf("30秒前: %s\n", timeutil.FriendlyTime(now.Add(-30*time.Second)))
	fmt.Printf("5分钟前: %s\n", timeutil.FriendlyTime(now.Add(-5*time.Minute)))
	fmt.Printf("2小时前: %s\n", timeutil.FriendlyTime(now.Add(-2*time.Hour)))
	fmt.Printf("昨天: %s\n", timeutil.FriendlyTime(now.Add(-25*time.Hour)))

	// 友好时长
	fmt.Println("\n--- 友好时长 ---")
	fmt.Printf("90分钟: %s\n", timeutil.Duration(90*time.Minute))
	fmt.Printf("2小时30分: %s\n", timeutil.Duration(150*time.Minute))

	// 日期计算
	fmt.Println("\n--- 日期计算 ---")
	a := time.Date(2024, 3, 15, 0, 0, 0, 0, time.Local)
	b := time.Date(2024, 3, 20, 0, 0, 0, 0, time.Local)
	fmt.Printf("2024-03-15 到 2024-03-20 相隔 %d 天\n", timeutil.DaysBetween(a, b))

	// 时区
	fmt.Println("\n--- 时区 ---")
	utc := time.Now().UTC()
	cst := timeutil.ToCST(utc)
	fmt.Printf("UTC: %s\n", timeutil.Format(utc, "HH:mm:ss"))
	fmt.Printf("北京: %s\n", timeutil.Format(cst, "HH:mm:ss"))

	// Cron
	fmt.Println("\n--- Cron ---")
	schedule, _ := timeutil.ParseCron("*/5 * * * *")
	next := schedule.Next(now)
	fmt.Printf("每5分钟执行，下次: %s\n", timeutil.Format(next, "HH:mm:ss"))

	schedule2, _ := timeutil.ParseCron("0 9 * * 1-5")
	next2 := schedule2.Next(now)
	fmt.Printf("工作日9点执行，下次: %s\n", timeutil.Format(next2, "YYYY-MM-DD HH:mm"))

	// Cron 定时器（演示 2 秒后停止）
	fmt.Println("\n--- Cron 定时器 ---")
	ticker, _ := timeutil.NewTicker("* * * * *", func() {
		fmt.Println("  [定时任务执行]")
	})
	fmt.Println("定时器已启动（每分钟执行），3秒后停止...")
	time.Sleep(3 * time.Second)
	ticker.Stop()
	fmt.Println("定时器已停止")
}
