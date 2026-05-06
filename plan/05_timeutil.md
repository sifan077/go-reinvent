# 时间工具库 · 设计规划

## 功能清单

- [x] 格式化：支持 YYYY-MM-DD 自定义 layout + Go 标准 layout
- [x] 解析：ParseInLocation 使用本地时区
- [x] 时间边界：StartOfDay / EndOfDay / StartOfMonth / EndOfMonth
- [x] 友好显示："刚刚"、"5分钟前"、"昨天 14:30"
- [x] 友好时长："1小时30分钟"
- [x] 工作日：IsWorkday / DaysBetween / AddWorkdays
- [x] 时区：Convert / ToCST / LocalNow
- [x] Cron：解析 5 位表达式 + 定时器封装

## 目录结构

```
pkg/timeutil/
├── format.go       # 格式化：Format/Parse/StartOfDay/EndOfDay/StartOfMonth/EndOfMonth
├── friendly.go     # 友好显示：FriendlyTime/Duration
├── calc.go         # 计算：IsWorkday/DaysBetween/AddWorkdays
├── timezone.go     # 时区：Convert/ToCST/LocalNow
├── cron.go         # Cron：ParseCron/Schedule/Match/Next/NewTicker
└── timeutil_test.go
cmd/timeutil/
└── main.go
```

## 核心设计

### 1. Layout 转换（format.go）

Go 的时间 layout 独树一帜——用一个**具体时间** `Mon Jan 2 15:04:05 MST 2006` 作为模板。
为降低记忆成本，提供 `YYYY-MM-DD HH:mm:ss` 风格的自定义 layout：

```go
var layoutReplacer = strings.NewReplacer(
    "YYYY", "2006",
    "MM",   "01",
    "DD",   "02",
    "HH",   "15",
    "mm",   "04",
    "ss",   "05",
)

func convertLayout(layout string) string {
    if strings.Contains(layout, "2006") {
        return layout  // 已是 Go 标准格式
    }
    return layoutReplacer.Replace(layout)
}
```

### 2. 时间不可变性

`time.Time` 是值类型，所有操作返回新值，不修改原值：

```go
// 错误理解：t 被修改了
t.Add(time.Hour)

// 正确：接收返回值
t = t.Add(time.Hour)
```

边界函数利用这一点：
```go
func EndOfMonth(t time.Time) time.Time {
    return StartOfMonth(t).AddDate(0, 1, 0).Add(-time.Nanosecond)
}
```

### 3. 友好时间（friendly.go）

基于 `time.Now().Sub(t)` 的时间差做分支：

```
< 1min   → "刚刚"
< 1hour  → "X分钟前"
< 24hour → "X小时前"
< 48hour → "昨天 HH:mm"
< 72hour → "前天 HH:mm"
其他     → "YYYY-MM-DD HH:mm"
```

### 4. 工作日计算（calc.go）

`AddWorkdays` 逐日步进，跳过周末：

```go
func AddWorkdays(t time.Time, n int) time.Time {
    step := 1
    if n < 0 { step = -1; n = -n }
    result := t
    for n > 0 {
        result = result.AddDate(0, 0, step)
        if IsWorkday(result) { n-- }
    }
    return result
}
```

### 5. Cron 解析（cron.go）

标准 5 位 cron：`分 时 日 月 周`

支持的语法：
| 语法 | 示例 | 说明 |
|------|------|------|
| `*` | `*` | 所有值 |
| `*/n` | `*/5` | 步长，每 n 个单位 |
| `n-m` | `1-5` | 范围 |
| `n,m,o` | `0,15,30,45` | 列表 |
| 组合 | `8-17/2` | 范围+步长 |

解析流程：
```
表达式 → 按空格拆 5 个字段 → 每个字段按逗号拆分 → 逐段解析（通配符/步长/范围/单值）→ 合并为有序切片
```

匹配逻辑：时间的分/时/日/月/周分别在对应切片中即匹配。

Next 计算：从 t+1min 开始逐分钟遍历，最多搜 3 年。

### 6. CronTicker 定时器

```go
type CronTicker struct {
    schedule *Schedule
    fn       func()
    stop     chan struct{}
}

// 启动后台 goroutine，计算下次触发时间 → timer → 执行 → 循环
// 通过 close(stop) 通知退出
```

## 知识点总结

| 知识点 | 说明 |
|--------|------|
| Go 时间 layout | 用 `2006-01-02 15:04:05` 这个具体时间做模板，不是 `yyyy-MM-dd` |
| time.Time 值类型 | 不可变，所有操作返回新值 |
| time.Date 构造 | 构造指定时间，时区用 `time.Local` / `time.UTC` / `LoadLocation` |
| time.Truncate | 截断到指定精度，`Truncate(time.Minute)` 去掉秒和纳秒 |
| chan + goroutine | 定时器模式：goroutine 等待 timer，select 监听 stop 信号 |
| strings.NewReplacer | 批量替换，比多次 Replace 高效 |
