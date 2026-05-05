package logger

import (
	"bytes"
	"strings"
	"sync"
	"testing"
)

// TestLevelString 测试级别名称转换
func TestLevelString(t *testing.T) {
	tests := []struct {
		level Level
		want  string
	}{
		{DEBUG, "DEBUG"},
		{INFO, "INFO"},
		{WARN, "WARN"},
		{ERROR, "ERROR"},
		{FATAL, "FATAL"},
		{Level(99), "UNKNOWN"}, // 越界值应返回 UNKNOWN
	}
	for _, tt := range tests {
		if got := tt.level.String(); got != tt.want {
			t.Errorf("Level(%d).String() = %q, want %q", tt.level, got, tt.want)
		}
	}
}

// TestDefaultLogger 测试基本输出是否包含级别和消息
func TestDefaultLogger(t *testing.T) {
	var buf bytes.Buffer
	log := New(
		WithOutput(&buf),
		WithColorful(false), // 关闭颜色，方便检查纯文本
	)

	log.Info("hello")

	output := buf.String()
	if !strings.Contains(output, "INFO") {
		t.Errorf("output missing INFO: %s", output)
	}
	if !strings.Contains(output, "hello") {
		t.Errorf("output missing message: %s", output)
	}
	if !strings.Contains(output, "[") {
		t.Errorf("output missing level brackets: %s", output)
	}
}

// TestLevelFiltering 测试低级别日志是否被正确过滤
func TestLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	log := New(
		WithOutput(&buf),
		WithColorful(false),
		WithLevel(WARN), // 只输出 WARN 及以上
	)

	log.Debug("debug message")
	log.Info("info message")
	log.Warn("warn message")

	output := buf.String()
	if strings.Contains(output, "debug message") {
		t.Error("DEBUG should be filtered")
	}
	if strings.Contains(output, "info message") {
		t.Error("INFO should be filtered")
	}
	if !strings.Contains(output, "warn message") {
		t.Error("WARN should pass through")
	}
}

// TestColorfulOutput 测试彩色模式是否输出 ANSI 转义码
func TestColorfulOutput(t *testing.T) {
	var buf bytes.Buffer
	log := New(
		WithOutput(&buf),
		WithColorful(true),
	)

	log.Error("boom")

	output := buf.String()
	if !strings.Contains(output, colorRed) {
		t.Errorf("output missing red color code: %s", output)
	}
	if !strings.Contains(output, colorReset) {
		t.Errorf("output missing reset code: %s", output)
	}
}

// TestNoColorOutput 测试关闭颜色后不应出现 ANSI 码
func TestNoColorOutput(t *testing.T) {
	var buf bytes.Buffer
	log := New(
		WithOutput(&buf),
		WithColorful(false),
	)

	log.Error("boom")

	output := buf.String()
	if strings.Contains(output, "\033") {
		t.Errorf("output should not contain ANSI codes: %s", output)
	}
}

// TestCallerInfo 测试 caller 信息是否包含文件名
func TestCallerInfo(t *testing.T) {
	var buf bytes.Buffer
	log := New(
		WithOutput(&buf),
		WithColorful(false),
		WithCaller(true),
	)

	log.Info("test")

	output := buf.String()
	if !strings.Contains(output, "logger_test.go") {
		t.Errorf("output missing caller file: %s", output)
	}
}

// TestFormattedVariants 测试 Infof 等格式化方法
func TestFormattedVariants(t *testing.T) {
	var buf bytes.Buffer
	log := New(
		WithOutput(&buf),
		WithColorful(false),
	)

	log.Infof("count: %d", 42)

	output := buf.String()
	if !strings.Contains(output, "count: 42") {
		t.Errorf("output missing formatted message: %s", output)
	}
}

// TestConcurrentWrite 测试并发写入安全性
// 100 个协程各写 10 条，最终应有 1000 行，验证 mutex 没有丢数据
func TestConcurrentWrite(t *testing.T) {
	var buf bytes.Buffer
	log := New(
		WithOutput(&buf),
		WithColorful(false),
	)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				log.Info("concurrent message")
			}
		}()
	}
	wg.Wait()

	lineCount := bytes.Count(buf.Bytes(), []byte("\n"))
	if lineCount != 1000 {
		t.Errorf("expected 1000 lines, got %d", lineCount)
	}
}
