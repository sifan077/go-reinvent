package rotatelog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Strategy 轮转策略接口
// 判断是否需要轮转，以及生成归档文件名
type Strategy interface {
	ShouldRotate(info os.FileInfo, now time.Time) bool
	NextFileName(baseName string, now time.Time) string
}

// SizeStrategy 按大小轮转
// 文件名格式：app.log -> app.1.log -> app.2.log
type SizeStrategy struct {
	MaxSize int64 // 字节
}

func (s *SizeStrategy) ShouldRotate(info os.FileInfo, _ time.Time) bool {
	return info.Size() >= s.MaxSize
}

func (s *SizeStrategy) NextFileName(baseName string, _ time.Time) string {
	dir := filepath.Dir(baseName)
	ext := filepath.Ext(baseName)
	name := strings.TrimSuffix(filepath.Base(baseName), ext)

	// 找到当前最大的编号，+1
	maxNum := 0
	pattern := filepath.Join(dir, name+".*"+ext)
	matches, _ := filepath.Glob(pattern)
	for _, m := range matches {
		base := filepath.Base(m)
		// 跳过主文件本身
		if base == filepath.Base(baseName) {
			continue
		}
		var num int
		// 格式：name.NUM.ext
		trimmed := strings.TrimPrefix(base, name+".")
		trimmed = strings.TrimSuffix(trimmed, ext)
		if _, err := fmt.Sscanf(trimmed, "%d", &num); err == nil && num > maxNum {
			maxNum = num
		}
	}
	return filepath.Join(dir, fmt.Sprintf("%s.%d%s", name, maxNum+1, ext))
}

// DateStrategy 按日期轮转
// 文件名格式：app.2024-01-15.log（daily）或 app.2024-01-15-14.log（hourly）
type DateStrategy struct {
	Interval string // "daily" 或 "hourly"
}

func (s *DateStrategy) ShouldRotate(_ os.FileInfo, now time.Time) bool {
	// 需要比较当前时间和文件创建时间是否在同一周期
	// 这里简化处理：由 RotateWriter 持有上一次轮转的时间点来判断
	// ShouldRotate 在此策略中不直接使用文件信息，由外部通过 needsDateRotate 辅助判断
	return false
}

func (s *DateStrategy) NextFileName(baseName string, now time.Time) string {
	dir := filepath.Dir(baseName)
	ext := filepath.Ext(baseName)
	name := strings.TrimSuffix(filepath.Base(baseName), ext)

	var dateStr string
	if s.Interval == "hourly" {
		dateStr = now.Format("2006-01-02-15")
	} else {
		dateStr = now.Format("2006-01-02")
	}
	return filepath.Join(dir, fmt.Sprintf("%s.%s%s", name, dateStr, ext))
}

// CombinedStrategy 按日期 + 大小组合轮转
// 文件名格式：app.2024-01-15.log，超大时 app.2024-01-15.1.log
type CombinedStrategy struct {
	DateStrategy
	MaxSize int64
}

func (s *CombinedStrategy) ShouldRotate(info os.FileInfo, now time.Time) bool {
	return info.Size() >= s.MaxSize
}

func (s *CombinedStrategy) NextFileName(baseName string, now time.Time) string {
	// 先用日期策略生成基础名
	dateName := s.DateStrategy.NextFileName(baseName, now)
	dir := filepath.Dir(dateName)
	ext := filepath.Ext(dateName)
	name := strings.TrimSuffix(filepath.Base(dateName), ext)

	// 检查该日期下已有多少个文件
	pattern := filepath.Join(dir, name+".*"+ext)
	matches, _ := filepath.Glob(pattern)

	if len(matches) == 0 {
		// 第一个文件用日期名，不带编号
		return dateName
	}

	// 找最大编号
	maxNum := 0
	for _, m := range matches {
		base := filepath.Base(m)
		if base == filepath.Base(dateName) {
			// 已有无编号的日期文件，编号从1开始
			if maxNum < 1 {
				maxNum = 1
			}
			continue
		}
		trimmed := strings.TrimPrefix(base, name+".")
		trimmed = strings.TrimSuffix(trimmed, ext)
		var num int
		if _, err := fmt.Sscanf(trimmed, "%d", &num); err == nil && num > maxNum {
			maxNum = num
		}
	}

	if maxNum == 0 {
		maxNum = 1
	}
	return filepath.Join(dir, fmt.Sprintf("%s.%d%s", name, maxNum+1, ext))
}

// NeedsDateRotate 判断是否跨周期（用于 DateStrategy 和 CombinedStrategy）
func NeedsDateRotate(lastRotate time.Time, now time.Time, interval string) bool {
	if interval == "hourly" {
		return lastRotate.Hour() != now.Hour() || lastRotate.YearDay() != now.YearDay()
	}
	// daily
	return lastRotate.YearDay() != now.YearDay() || lastRotate.Year() != now.Year()
}
