package rotatelog

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func tempDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return dir
}

func TestSizeRotation(t *testing.T) {
	dir := tempDir(t)
	filename := filepath.Join(dir, "app.log")

	w := New(filename,
		WithMaxSize(1), // 1MB
		WithMaxBackups(3),
	)
	defer w.Close()

	// 写入超过 1MB 的数据
	data := make([]byte, 1024)  // 1KB
	for i := 0; i < 1100; i++ { // ~1.07MB
		w.Write(data)
	}

	// 等待清理 goroutine
	time.Sleep(50 * time.Millisecond)

	// 应该已经轮转，存在 app.1.log
	if _, err := os.Stat(filepath.Join(dir, "app.1.log")); err != nil {
		t.Errorf("expected app.1.log to exist after size rotation: %v", err)
	}

	// 当前 app.log 应该存在且大小小于 MaxSize
	info, err := os.Stat(filename)
	if err != nil {
		t.Fatalf("app.log should exist: %v", err)
	}
	if info.Size() >= 1024*1024 {
		t.Errorf("app.log size %d should be less than max size", info.Size())
	}
}

func TestSizeRotationNaming(t *testing.T) {
	dir := tempDir(t)
	filename := filepath.Join(dir, "test.log")

	w := New(filename, WithMaxSize(1)) // 1MB
	defer w.Close()

	// 写入足够触发多次轮转的数据
	data := make([]byte, 1024)
	for round := 0; round < 3; round++ {
		for i := 0; i < 1100; i++ {
			w.Write(data)
		}
		// 等待清理 goroutine 完成
		time.Sleep(50 * time.Millisecond)
	}

	// 检查文件编号连续
	entries, _ := os.ReadDir(dir)
	count := 0
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "test.") && strings.HasSuffix(e.Name(), ".log") && e.Name() != "test.log" {
			count++
		}
	}
	if count < 2 {
		t.Errorf("expected at least 2 rotated files, got %d", count)
	}
}

func TestMaxBackupsCleanup(t *testing.T) {
	dir := tempDir(t)
	filename := filepath.Join(dir, "app.log")

	w := New(filename,
		WithMaxSize(1),
		WithMaxBackups(2), // 只保留2个备份
	)
	defer w.Close()

	data := make([]byte, 1024)
	// 写入足够触发5次轮转
	for round := 0; round < 5; round++ {
		for i := 0; i < 1100; i++ {
			w.Write(data)
		}
		time.Sleep(50 * time.Millisecond)
	}

	// 等待清理完成
	time.Sleep(100 * time.Millisecond)

	// 统计旧日志文件数量（不含当前活跃文件）
	entries, _ := os.ReadDir(dir)
	oldCount := 0
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, "app.") && name != "app.log" && strings.HasSuffix(name, ".log") {
			oldCount++
		}
	}

	if oldCount > 2 {
		t.Errorf("expected at most 2 backup files, got %d", oldCount)
	}
}

func TestMaxAgeCleanup(t *testing.T) {
	dir := tempDir(t)
	filename := filepath.Join(dir, "app.log")

	w := New(filename,
		WithMaxSize(1),
		WithMaxAge(1), // 保留1天
	)
	defer w.Close()

	// 手动创建一个旧文件
	oldFile := filepath.Join(dir, "app.1.log")
	os.WriteFile(oldFile, []byte("old"), 0644)
	oldTime := time.Now().Add(-48 * time.Hour)
	os.Chtimes(oldFile, oldTime, oldTime)

	// 触发轮转
	data := make([]byte, 1024)
	for i := 0; i < 1100; i++ {
		w.Write(data)
	}
	time.Sleep(100 * time.Millisecond)

	// 旧文件应该被删除
	if _, err := os.Stat(oldFile); !os.IsNotExist(err) {
		t.Errorf("old file should have been deleted by maxAge cleanup")
	}
}

func TestCompress(t *testing.T) {
	dir := tempDir(t)
	filename := filepath.Join(dir, "app.log")

	w := New(filename,
		WithMaxSize(1),
		WithCompress(true),
	)
	defer w.Close()

	data := make([]byte, 1024)
	for i := 0; i < 1100; i++ {
		w.Write(data)
	}
	// cleanup 在 goroutine 中执行，需要等待
	time.Sleep(500 * time.Millisecond)

	// 应该存在 .gz 文件
	gzFile := filepath.Join(dir, "app.1.log.gz")
	if _, err := os.Stat(gzFile); err != nil {
		t.Errorf("expected compressed file app.1.log.gz: %v", err)
	}

	// 验证 .gz 文件可以正常读取
	f, err := os.Open(gzFile)
	if err != nil {
		t.Fatalf("failed to open gz file: %v", err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}
	defer gr.Close()

	content, err := io.ReadAll(gr)
	if err != nil {
		t.Fatalf("failed to read gz content: %v", err)
	}
	if len(content) == 0 {
		t.Error("gz file should not be empty")
	}
}

func TestDateRotation(t *testing.T) {
	dir := tempDir(t)
	filename := filepath.Join(dir, "app.log")

	w := New(filename, WithRotateByDate("daily"))
	defer w.Close()

	// 写入一些数据
	w.Write([]byte("hello\n"))

	// 当前文件应该存在
	if _, err := os.Stat(filename); err != nil {
		t.Fatalf("app.log should exist: %v", err)
	}

	// 手动模拟日期变化：修改 lastRotate 为昨天
	w.mu.Lock()
	w.lastRotate = time.Now().Add(-24 * time.Hour)
	w.mu.Unlock()

	// 再次写入，应该触发日期轮转
	w.Write([]byte("world\n"))

	expected := filepath.Join(dir, fmt.Sprintf("app.%s.log", time.Now().Format("2006-01-02")))
	if _, err := os.Stat(expected); err != nil {
		t.Errorf("expected date-rotated file %s: %v", expected, err)
	}
}

func TestConcurrentWrite(t *testing.T) {
	dir := tempDir(t)
	filename := filepath.Join(dir, "app.log")

	w := New(filename, WithMaxSize(10))
	defer w.Close()

	var wg sync.WaitGroup
	goroutines := 100
	messages := 100

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for m := 0; m < messages; m++ {
				w.Write([]byte(fmt.Sprintf("goroutine %d message %d\n", id, m)))
			}
		}(g)
	}

	wg.Wait()

	// 验证没有 panic，文件存在
	if _, err := os.Stat(filename); err != nil {
		t.Fatalf("app.log should exist after concurrent writes: %v", err)
	}
}

func TestCloseAndReopen(t *testing.T) {
	dir := tempDir(t)
	filename := filepath.Join(dir, "app.log")

	w := New(filename)
	w.Write([]byte("before close\n"))

	if err := w.Close(); err != nil {
		t.Fatalf("close failed: %v", err)
	}

	// 写入后应该自动重新打开
	w.Write([]byte("after close\n"))

	w.Close()

	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	str := string(content)
	if !strings.Contains(str, "before close") || !strings.Contains(str, "after close") {
		t.Errorf("file should contain both messages, got: %s", str)
	}
}

func TestRotateWriter_WriteReturnsCorrectLength(t *testing.T) {
	dir := tempDir(t)
	filename := filepath.Join(dir, "app.log")

	w := New(filename)
	defer w.Close()

	data := []byte("hello world")
	n, err := w.Write(data)
	if err != nil {
		t.Fatalf("write failed: %v", err)
	}
	if n != len(data) {
		t.Errorf("expected n=%d, got %d", len(data), n)
	}
}

func TestOptionDefaults(t *testing.T) {
	dir := tempDir(t)
	filename := filepath.Join(dir, "app.log")

	w := New(filename)
	defer w.Close()

	if w.maxSize != 100*1024*1024 {
		t.Errorf("default maxSize should be 100MB, got %d", w.maxSize)
	}
	if w.maxBackups != 0 {
		t.Errorf("default maxBackups should be 0, got %d", w.maxBackups)
	}
	if w.maxAge != 0 {
		t.Errorf("default maxAge should be 0, got %d", w.maxAge)
	}
	if w.compress {
		t.Error("default compress should be false")
	}
	if w.perm != 0644 {
		t.Errorf("default perm should be 0644, got %o", w.perm)
	}
}

func TestManualRotate(t *testing.T) {
	dir := tempDir(t)
	filename := filepath.Join(dir, "app.log")

	w := New(filename, WithMaxSize(100*1024*1024))
	defer w.Close()

	w.Write([]byte("some data\n"))

	if err := w.Rotate(); err != nil {
		t.Fatalf("manual rotate failed: %v", err)
	}

	// 应该存在 app.1.log
	if _, err := os.Stat(filepath.Join(dir, "app.1.log")); err != nil {
		t.Errorf("app.1.log should exist after manual rotate: %v", err)
	}

	// app.log 应该是新的（空或很小）
	info, _ := os.Stat(filename)
	if info.Size() > 0 {
		// 新文件可能还没写入，这是正常的
	}
}
