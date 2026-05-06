package timeutil

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Schedule 解析后的 cron 调度计划
// 内部存储每个字段的合法值集合，用于快速匹配时间点
type Schedule struct {
	minutes     []int // 0-59
	hours       []int // 0-23
	daysOfMonth []int // 1-31
	months      []int // 1-12
	daysOfWeek  []int // 0-6 (0=Sunday)
}

// ParseCron 解析标准 5 位 cron 表达式
// 格式: 分 时 日 月 周
//
// 支持的语法:
//
//   - 所有值，如 * * * * * 表示每分钟
//     */n     步长，如 */5 表示每 5 个单位
//     m/n     步长，如 3/5 表示从3开始每 5 个单位
//     n-m     范围，如 1-5 表示 1 到 5
//     n,m,o   列表，如 0,15,30,45 表示这四个值
//     n-m/s   范围+步长，如 8-17/2 表示 8 到 17 每隔 2
//
// 示例:
//
//	"*/5 * * * *"       每 5 分钟
//	"0 9 * * *"         每天 9:00
//	"0 9 * * 1-5"       工作日 9:00
//	"0,15,30,45 * * * *" 每小时的 0/15/30/45 分
func ParseCron(expr string) (*Schedule, error) {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return nil, fmt.Errorf("cron expression must have 5 fields, got %d", len(fields))
	}

	s := &Schedule{}
	var err error

	// 依次解析 5 个字段：分(0-59) 时(0-23) 日(1-31) 月(1-12) 周(0-6)
	if s.minutes, err = parseField(fields[0], 0, 59); err != nil {
		return nil, fmt.Errorf("invalid minute field: %w", err)
	}
	if s.hours, err = parseField(fields[1], 0, 23); err != nil {
		return nil, fmt.Errorf("invalid hour field: %w", err)
	}
	if s.daysOfMonth, err = parseField(fields[2], 1, 31); err != nil {
		return nil, fmt.Errorf("invalid day-of-month field: %w", err)
	}
	if s.months, err = parseField(fields[3], 1, 12); err != nil {
		return nil, fmt.Errorf("invalid month field: %w", err)
	}
	if s.daysOfWeek, err = parseField(fields[4], 0, 6); err != nil {
		return nil, fmt.Errorf("invalid day-of-week field: %w", err)
	}

	return s, nil
}

// parseField 解析单个 cron 字段，返回该字段所有合法值的有序切片
// 解析优先级：步长(/) > 范围(-) > 通配符(*) > 单值
func parseField(field string, min, max int) ([]int, error) {
	// 用 map 去重，避免列表中有重复值
	set := make(map[int]bool)

	// 逗号分隔的多个子表达式，如 "0,15,30"
	parts := strings.Split(field, ",")
	for _, part := range parts {
		// 1) 步长表达式: */5 或 1-30/5
		if strings.Contains(part, "/") {
			if err := parseStep(part, min, max, set); err != nil {
				return nil, err
			}
			continue
		}

		// 2) 范围表达式: 1-5
		if strings.Contains(part, "-") {
			if err := parseRange(part, set); err != nil {
				return nil, err
			}
			continue
		}

		// 3) 通配符: *
		if part == "*" {
			for i := min; i <= max; i++ {
				set[i] = true
			}
			continue
		}

		// 4) 单个数值: 30
		n, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid value: %s", part)
		}
		set[n] = true
	}

	// 将 map 转为有序切片，过滤掉超出范围的值
	result := make([]int, 0, len(set))
	for v := range set {
		if v >= min && v <= max {
			result = append(result, v)
		}
	}
	sort.Ints(result)
	return result, nil
}

// parseStep 解析步长表达式，如 */5、8-17/2
func parseStep(part string, min, max int, set map[int]bool) error {
	stepParts := strings.SplitN(part, "/", 2)
	step, err := strconv.Atoi(stepParts[1])
	if err != nil || step <= 0 {
		return fmt.Errorf("invalid step: %s", stepParts[1])
	}

	// 确定范围：* 表示完整范围，否则解析 m-n
	rangeMin, rangeMax := min, max
	if stepParts[0] != "*" {
		rangeParts := strings.SplitN(stepParts[0], "-", 2)
		rangeMin, err = strconv.Atoi(rangeParts[0])
		if err != nil {
			return fmt.Errorf("invalid range start: %s", rangeParts[0])
		}
		if len(rangeParts) == 2 {
			rangeMax, err = strconv.Atoi(rangeParts[1])
			if err != nil {
				return fmt.Errorf("invalid range end: %s", rangeParts[1])
			}
		}
	}

	// 按步长填充
	for i := rangeMin; i <= rangeMax; i += step {
		set[i] = true
	}
	return nil
}

// parseRange 解析范围表达式，如 1-5
func parseRange(part string, set map[int]bool) error {
	rangeParts := strings.SplitN(part, "-", 2)
	rangeMin, err := strconv.Atoi(rangeParts[0])
	if err != nil {
		return fmt.Errorf("invalid range start: %s", rangeParts[0])
	}
	rangeMax, err := strconv.Atoi(rangeParts[1])
	if err != nil {
		return fmt.Errorf("invalid range end: %s", rangeParts[1])
	}
	for i := rangeMin; i <= rangeMax; i++ {
		set[i] = true
	}
	return nil
}

// Match 判断给定时间是否匹配 cron 表达式
// 5 个字段全部匹配才返回 true
func (s *Schedule) Match(t time.Time) bool {
	return contains(s.minutes, t.Minute()) &&
		contains(s.hours, t.Hour()) &&
		contains(s.daysOfMonth, t.Day()) &&
		contains(s.months, int(t.Month())) &&
		contains(s.daysOfWeek, int(t.Weekday()))
}

// Next 计算给定时间之后的下一次触发时间
// 从 t 的下一分钟开始逐分钟搜索，最多搜索 3 年（避免死循环）
func (s *Schedule) Next(t time.Time) time.Time {
	// 先跳到下一分钟的整点，避免重复触发当前分钟
	t = t.Add(time.Minute).Truncate(time.Minute)

	limit := t.AddDate(3, 0, 0) // 搜索上限：3 年
	for t.Before(limit) {
		if s.Match(t) {
			return t
		}
		t = t.Add(time.Minute)
	}
	return time.Time{} // 未找到匹配时间
}

// contains 判断整数切片中是否包含指定值
func contains(slice []int, val int) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}

// CronTicker 基于 cron 表达式的定时器
// 内部启动 goroutine 自动计算下次触发时间并执行回调
type CronTicker struct {
	schedule *Schedule     // 解析后的调度计划
	fn       func()        // 触发时执行的回调函数
	stop     chan struct{} // 关闭信号通道
}

// NewTicker 创建基于 cron 表达式的定时器
// 启动后会自动在后台 goroutine 中循环调度，直到调用 Stop()
func NewTicker(expr string, fn func()) (*CronTicker, error) {
	schedule, err := ParseCron(expr)
	if err != nil {
		return nil, err
	}

	ct := &CronTicker{
		schedule: schedule,
		fn:       fn,
		stop:     make(chan struct{}),
	}

	go ct.run()
	return ct, nil
}

// run 定时器主循环：计算下次触发时间 → 等待 → 执行回调 → 重复
func (ct *CronTicker) run() {
	for {
		next := ct.schedule.Next(time.Now())
		if next.IsZero() {
			return // 无法计算下次触发时间，退出
		}
		// 用 Timer 等待到下次触发时间
		timer := time.NewTimer(next.Sub(time.Now()))
		select {
		case <-timer.C:
			ct.fn() // 时间到，执行回调
		case <-ct.stop:
			timer.Stop()
			return // 收到停止信号，退出
		}
	}
}

// Stop 停止定时器，关闭后台 goroutine
func (ct *CronTicker) Stop() {
	close(ct.stop)
}
