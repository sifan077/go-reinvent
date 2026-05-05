package rotatelog

import (
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// logFile 表示一个旧日志文件，用于排序和清理
type logFile struct {
	path    string
	modTime time.Time
}

// cleanup 清理旧日志文件
// 1. 收集目录下匹配的日志文件（排除当前活跃文件）
// 2. 按 maxBackups 限制删除多余文件
// 3. 按 maxAge 限制删除过期文件
// 4. 可选 gzip 压缩
func (w *RotateWriter) cleanup() {
	dir := filepath.Dir(w.filename)
	ext := filepath.Ext(w.filename)
	base := strings.TrimSuffix(filepath.Base(w.filename), ext)

	// 收集所有匹配的旧日志文件（排除当前活跃文件 w.filename）
	matches := w.findOldFiles(dir, base, ext, w.filename)

	// 按修改时间从新到旧排序
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].modTime.After(matches[j].modTime)
	})

	now := time.Now()
	var toDelete []logFile

	for i, f := range matches {
		// maxBackups 限制
		if w.maxBackups > 0 && i >= w.maxBackups {
			toDelete = append(toDelete, f)
			continue
		}

		// maxAge 限制
		if w.maxAge > 0 && now.Sub(f.modTime).Hours() > float64(w.maxAge*24) {
			toDelete = append(toDelete, f)
			continue
		}
	}

	// 删除需要清理的文件
	for _, f := range toDelete {
		os.Remove(f.path)
		// 压缩模式下也删除可能存在的 .gz 版本
		if w.compress {
			os.Remove(f.path + ".gz")
		}
	}
}

// findOldFiles 查找目录下匹配的旧日志文件
func (w *RotateWriter) findOldFiles(dir, base, ext, activeFile string) []logFile {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}

	var files []logFile
	prefix := base + "."

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		// 匹配 base.NUM.ext 或 base.DATE.ext 或 base.DATE.NUM.ext 或 .gz 后缀
		if !strings.HasPrefix(name, prefix) {
			continue
		}

		fullPath := filepath.Join(dir, name)

		// 排除当前活跃文件
		if fullPath == activeFile {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		files = append(files, logFile{
			path:    fullPath,
			modTime: info.ModTime(),
		})
	}

	return files
}

// gzipFile 将文件压缩为 .gz，原文件会被替换
func (w *RotateWriter) gzipFile(path string) error {
	src, err := os.Open(path)
	if err != nil {
		return err
	}
	defer src.Close()

	dstPath := path + ".gz"
	dst, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, w.perm)
	if err != nil {
		return err
	}
	defer dst.Close()

	gz := gzip.NewWriter(dst)
	defer gz.Close()

	if _, err := io.Copy(gz, src); err != nil {
		return err
	}

	// 关闭 gzip writer 以写入尾部数据
	if err := gz.Close(); err != nil {
		return err
	}
	dst.Close()
	src.Close()

	// 压缩成功后删除原文件
	return os.Remove(path)
}
