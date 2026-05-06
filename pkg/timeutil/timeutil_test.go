package timeutil

import (
	"testing"
	"time"
)

func TestFormat(t *testing.T) {
	tm := time.Date(2024, 3, 15, 14, 30, 45, 0, time.Local)

	tests := []struct {
		name   string
		layout string
		want   string
	}{
		{"自定义格式", "YYYY-MM-DD HH:mm:ss", "2024-03-15 14:30:45"},
		{"日期", "YYYY/MM/DD", "2024/03/15"},
		{"时间", "HH:mm", "14:30"},
		{"Go标准格式", "2006-01-02", "2024-03-15"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Format(tm, tt.layout); got != tt.want {
				t.Errorf("Format() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		s       string
		layout  string
		wantErr bool
	}{
		{"自定义格式", "2024-03-15 14:30:00", "YYYY-MM-DD HH:mm:ss", false},
		{"日期", "2024/03/15", "YYYY/MM/DD", false},
		{"Go标准格式", "2024-03-15", "2006-01-02", false},
		{"非法格式", "not-a-date", "YYYY-MM-DD", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.s, tt.layout)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStartOfDay(t *testing.T) {
	tm := time.Date(2024, 3, 15, 14, 30, 45, 123456789, time.Local)
	got := StartOfDay(tm)
	want := time.Date(2024, 3, 15, 0, 0, 0, 0, time.Local)
	if !got.Equal(want) {
		t.Errorf("StartOfDay() = %v, want %v", got, want)
	}
}

func TestEndOfDay(t *testing.T) {
	tm := time.Date(2024, 3, 15, 14, 30, 45, 0, time.Local)
	got := EndOfDay(tm)
	want := time.Date(2024, 3, 15, 23, 59, 59, 999999999, time.Local)
	if !got.Equal(want) {
		t.Errorf("EndOfDay() = %v, want %v", got, want)
	}
}

func TestStartOfMonth(t *testing.T) {
	tm := time.Date(2024, 3, 15, 14, 30, 45, 0, time.Local)
	got := StartOfMonth(tm)
	want := time.Date(2024, 3, 1, 0, 0, 0, 0, time.Local)
	if !got.Equal(want) {
		t.Errorf("StartOfMonth() = %v, want %v", got, want)
	}
}

func TestEndOfMonth(t *testing.T) {
	tests := []struct {
		name string
		t    time.Time
		want int // 期望的日
	}{
		{"3月", time.Date(2024, 3, 15, 0, 0, 0, 0, time.Local), 31},
		{"2月闰年", time.Date(2024, 2, 1, 0, 0, 0, 0, time.Local), 29},
		{"2月平年", time.Date(2023, 2, 1, 0, 0, 0, 0, time.Local), 28},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EndOfMonth(tt.t)
			if got.Day() != tt.want {
				t.Errorf("EndOfMonth() day = %d, want %d", got.Day(), tt.want)
			}
		})
	}
}

func TestFriendlyTime(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name string
		t    time.Time
		want string
	}{
		{"刚刚", now.Add(-30 * time.Second), "刚刚"},
		{"5分钟前", now.Add(-5 * time.Minute), "5分钟前"},
		{"2小时前", now.Add(-2 * time.Hour), "2小时前"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FriendlyTime(tt.t); got != tt.want {
				t.Errorf("FriendlyTime() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDuration(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
		want string
	}{
		{"1小时30分钟", 90 * time.Minute, "1小时30分钟"},
		{"45秒", 45 * time.Second, "45秒"},
		{"2小时", 2 * time.Hour, "2小时"},
		{"30分钟", 30 * time.Minute, "30分钟"},
		{"0秒", 0, "0秒"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Duration(tt.d); got != tt.want {
				t.Errorf("Duration() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDaysBetween(t *testing.T) {
	a := time.Date(2024, 3, 15, 0, 0, 0, 0, time.Local)
	b := time.Date(2024, 3, 20, 0, 0, 0, 0, time.Local)
	if got := DaysBetween(a, b); got != 5 {
		t.Errorf("DaysBetween() = %d, want 5", got)
	}
	// 反向也一样
	if got := DaysBetween(b, a); got != 5 {
		t.Errorf("DaysBetween() = %d, want 5", got)
	}
}

func TestConvert(t *testing.T) {
	utc := time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC)
	cst, err := Convert(utc, "Asia/Shanghai")
	if err != nil {
		t.Fatalf("Convert() error = %v", err)
	}
	if cst.Hour() != 20 { // UTC 12:00 = CST 20:00
		t.Errorf("Convert() hour = %d, want 20", cst.Hour())
	}
}

func TestToCST(t *testing.T) {
	utc := time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC)
	cst := ToCST(utc)
	if cst.Hour() != 20 {
		t.Errorf("ToCST() hour = %d, want 20", cst.Hour())
	}
}

func TestParseCron(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		wantErr bool
	}{
		{"每5分钟", "*/5 * * * *", false},
		{"每天9点", "0 9 * * *", false},
		{"工作日9点", "0 9 * * 1-5", false},
		{"复杂表达式", "0,15,30,45 8-17 * * 1-5", false},
		{"字段不足", "* * *", true},
		{"非法字段", "abc * * * *", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseCron(tt.expr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseCron() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestScheduleMatch(t *testing.T) {
	schedule, _ := ParseCron("0 9 * * 1-5")

	tests := []struct {
		name string
		t    time.Time
		want bool
	}{
		{"工作日9点匹配", time.Date(2024, 3, 18, 9, 0, 0, 0, time.Local), true},
		{"工作日10点不匹配", time.Date(2024, 3, 18, 10, 0, 0, 0, time.Local), false},
		{"周六9点不匹配", time.Date(2024, 3, 23, 9, 0, 0, 0, time.Local), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := schedule.Match(tt.t); got != tt.want {
				t.Errorf("Match() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestScheduleNext(t *testing.T) {
	schedule, _ := ParseCron("0 9 * * *")

	// 当前是 8:00，下一次应该是今天的 9:00
	current := time.Date(2024, 3, 18, 8, 0, 0, 0, time.Local)
	next := schedule.Next(current)
	want := time.Date(2024, 3, 18, 9, 0, 0, 0, time.Local)
	if !next.Equal(want) {
		t.Errorf("Next() = %v, want %v", next, want)
	}

	// 当前是 9:00，下一次应该是明天的 9:00（因为 Next 从下一分钟开始）
	current = time.Date(2024, 3, 18, 9, 0, 0, 0, time.Local)
	next = schedule.Next(current)
	want = time.Date(2024, 3, 19, 9, 0, 0, 0, time.Local)
	if !next.Equal(want) {
		t.Errorf("Next() = %v, want %v", next, want)
	}
}
