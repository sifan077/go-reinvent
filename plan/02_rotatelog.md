# 日志轮转库 · 设计规划

## 功能清单

- [x] 按大小切割：单文件超过指定大小自动轮转
- [x] 按日期切割：按天/按小时自动创建新文件
- [x] 旧日志自动清理：按备份数量或保留天数清理
- [x] gzip 压缩旧日志
- [x] 并发安全：支持多 goroutine 同时写入
- [x] 实现 `io.Writer`，与现有 logger 无缝集成

## 目录结构

```
pkg/rotatelog/
├── rotatelog.go      # RotateWriter 核心结构与 Write/Rotate 逻辑
├── option.go         # Option 模式配置
├── strategy.go       # 轮转策略接口与实现（size / date / combined）
├── cleanup.go        # 旧日志清理与 gzip 压缩
└── rotatelog_test.go # 单元测试（11 个）

cmd/rotatelog/
└── main.go           # 使用示例
```

## 核心设计

### 1. 轮转策略接口

```go
type Strategy interface {
    ShouldRotate(info os.FileInfo, now time.Time) bool
    NextFileName(baseName string, now time.Time) string
}
```

三种实现：

| 策略 | 文件名格式 | 触发条件 |
|------|-----------|---------|
| `SizeStrategy` | `app.log`, `app.1.log`, `app.2.log` | 文件大小超过 MaxSize |
| `DateStrategy` | `app.2024-01-15.log` | 日期变化（跨天/跨小时） |
| `CombinedStrategy` | `app.2024-01-15.log`, `app.2024-01-15.1.log` | 日期变化 OR 文件过大 |

### 2. RotateWriter 结构体

```go
type RotateWriter struct {
    mu           sync.Mutex
    filename     string        // 基础文件名
    fp           *os.File      // 当前文件
    size         int64         // 当前文件已写字节数
    strategy     Strategy      // 轮转策略
    maxSize      int64         // 单文件最大字节数
    maxBackups   int           // 最多保留旧文件数
    maxAge       int           // 最多保留天数
    compress     bool          // 是否 gzip 压缩
    dateInterval string        // "daily" / "hourly"
    perm         os.FileMode
    lastRotate   time.Time     // 上次轮转时间
}
```

### 3. Option 模式

```go
WithMaxSize(mb int)            // 单文件最大 MB，默认 100
WithMaxBackups(n int)          // 最多保留旧文件数，默认 0（不限）
WithMaxAge(days int)           // 最多保留天数，默认 0（不限）
WithCompress(enable bool)      // 是否 gzip 压缩，默认 false
WithRotateByDate(interval string) // "daily" / "hourly"
WithPerm(perm os.FileMode)     // 文件权限，默认 0644
```

### 4. Write 流程

```
Write(p []byte)
  ├─ 加锁
  ├─ 确保文件已打开（懒初始化）
  ├─ 检查日期轮转（跨天/跨小时）
  ├─ 写入数据，更新 size
  ├─ 检查大小轮转（size >= maxSize）
  │     ├─ Close 当前文件
  │     ├─ Rename -> 归档名
  │     ├─ 可选 gzip 压缩（同步）
  │     ├─ Create 新文件
  │     └─ 异步 cleanup() 清理旧日志
  └─ 解锁
```

### 5. 清理逻辑

轮转后异步执行：
1. 收集目录下匹配 `basename.*` 的文件（排除当前活跃文件）
2. 按修改时间从新到旧排序
3. `maxBackups` 限制：超出数量的删除
4. `maxAge` 限制：超过天数的删除

### 6. 与现有 Logger 集成

```go
writer := rotatelog.New("logs/app.log",
    rotatelog.WithMaxSize(50),
    rotatelog.WithMaxBackups(3),
    rotatelog.WithMaxAge(7),
    rotatelog.WithRotateByDate("daily"),
)

log := logger.New(
    logger.WithOutput(writer),
    logger.WithColorful(false),
    logger.WithLevel(logger.INFO),
)
```

## 测试覆盖

| 测试 | 验证内容 |
|------|---------|
| TestSizeRotation | 按大小轮转后文件存在 |
| TestSizeRotationNaming | 文件编号连续递增 |
| TestMaxBackupsCleanup | 旧文件数量受限制 |
| TestMaxAgeCleanup | 过期文件被删除 |
| TestCompress | gzip 压缩文件可正常读取 |
| TestDateRotation | 跨天后日期文件生成 |
| TestConcurrentWrite | 100 goroutine 并发安全 |
| TestCloseAndReopen | 关闭后自动重新打开 |
| TestWriteReturnsCorrectLength | Write 返回正确字节数 |
| TestOptionDefaults | 默认配置正确 |
| TestManualRotate | 手动轮转正常 |
