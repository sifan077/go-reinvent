# 彩色分级日志库 · 设计规划

## 功能清单

- [ ] 日志级别：DEBUG / INFO / WARN / ERROR / FATAL
- [ ] 时间格式化：`2006-01-02 15:04:05.000`
- [ ] 控制台彩色输出：不同级别不同颜色（ANSI 转义码）
- [ ] 自定义日志格式：支持配置输出模板
- [ ] 文件输出：可选写入文件（无颜色码）
- [ ] Caller 信息：可选打印调用位置（文件名:行号）

## 目录结构

```
pkg/logger/
├── logger.go       # Logger 结构体、核心输出逻辑
├── level.go        # 日志级别定义与解析
├── color.go        # ANSI 颜色封装
├── formatter.go    # 格式化器接口与默认实现
├── option.go       # Option 模式配置
└── logger_test.go  # 单元测试
```

## 核心设计

### 1. 日志级别

```go
type Level int

const (
    LevelDebug Level = iota
    LevelInfo
    LevelWarn
    LevelError
    LevelFatal
)
```

- 每个级别有对应的颜色和名称字符串
- Logger 持有一个最低级别，低于该级别的日志不输出

### 2. ANSI 颜色

```go
const (
    ColorRed    = "\033[31m"
    ColorYellow = "\033[33m"
    ColorBlue   = "\033[34m"
    ColorGray   = "\033[37m"
    ColorReset  = "\033[0m"
)
```

| 级别   | 颜色    |
|--------|---------|
| DEBUG  | Gray    |
| INFO   | Blue    |
| WARN   | Yellow  |
| ERROR  | Red     |
| FATAL  | Red+粗体 |

### 3. Logger 结构体

```go
type Logger struct {
    level     Level
    out       io.Writer    // 输出目标（os.Stdout / 文件）
    colorful  bool         // 是否启用颜色
    showCaller bool        // 是否显示调用位置
    formatter  Formatter   // 格式化器
}
```

### 4. Option 模式

```go
type Option func(*Logger)

func WithLevel(level Level) Option
func WithColorful(enable bool) Option
func WithOutput(w io.Writer) Option
func WithCaller(enable bool) Option
func WithFormatter(f Formatter) Option
```

使用方式：
```go
log := logger.New(
    logger.WithLevel(logger.LevelInfo),
    logger.WithColorful(true),
    logger.WithCaller(true),
)
```

### 5. Formatter 接口

```go
type Formatter interface {
    Format(level Level, msg string, caller string, ts time.Time) string
}
```

默认实现 `TextFormatter`，输出格式：
```
2024-01-15 14:30:05.123 [INFO] main.go:42 - something happened
```

### 6. 对外 API

```go
func (l *Logger) Debug(msg string)
func (l *Logger) Info(msg string)
func (l *Logger) Warn(msg string)
func (l *Logger) Error(msg string)
func (l *Logger) Fatal(msg string)

// 支持格式化
func (l *Logger) Debugf(format string, args ...any)
func (l *Logger) Infof(format string, args ...any)
// ...
```

## 测试计划

- [ ] 各级别输出是否正确
- [ ] 低于最低级别的日志是否被过滤
- [ ] 彩色开关是否生效
- [ ] Caller 信息是否正确
- [ ] 文件输出是否无颜色码
- [ ] 格式化输出是否符合预期

## 参考

- 标准库 `log` 包的 API 设计
- ANSI 转义码：`\033[{n}m` 为颜色，`\033[0m` 为重置
