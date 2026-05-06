# 字符串工具库 · 设计规划

## 功能清单

- [x] 安全截取：支持中文等多字节字符，支持负数索引
- [x] 填充：左填充 / 右填充
- [x] 翻转：支持多字节字符
- [x] 命名转换：驼峰 ↔ 下划线
- [x] 首字母大写
- [x] 掩码：手机号 / 邮箱 / 身份证 / 通用掩码
- [x] 类型转换：string ↔ int / float64 / bool，支持 Must 系列

## 目录结构

```
pkg/strutil/
├── transform.go    # 字符串变换：Substr/Pad/Reverse/CamelToSnake/SnakeToCamel/Capitalize
├── mask.go         # 掩码：Mask/MaskPhone/MaskEmail/MaskIDCard
├── convert.go      # 类型转换：ToInt/ToFloat64/ToBool/ToString + Must系列
└── strutil_test.go # 单元测试
cmd/strutil/
└── main.go         # 演示入口
```

## 核心设计

### 1. rune 操作（transform.go）

Go 的 `string` 是 UTF-8 编码，直接按字节索引会截断中文。所有涉及长度/位置的操作都先转 `[]rune`：

```go
func Substr(s string, start, length int) string {
    runes := []rune(s)
    // ... 基于 rune 切片操作
    return string(runes[start : start+length])
}
```

关键函数：

| 函数 | 说明 |
|------|------|
| `Substr(s, start, length)` | 安全截取，支持负数 start |
| `PadLeft(s, length, padChar)` | 左侧填充到指定长度 |
| `PadRight(s, length, padChar)` | 右侧填充到指定长度 |
| `Reverse(s)` | 翻转字符串 |
| `CamelToSnake(s)` | `helloWorld` → `hello_world` |
| `SnakeToCamel(s)` | `hello_world` → `helloWorld` |
| `Capitalize(s)` | 首字母大写 |

### 2. 掩码设计（mask.go）

通用掩码函数 + 特化封装：

```go
// 通用掩码：指定区间替换为掩码字符
func Mask(s string, start, end int, maskChar rune) string

// 特化：保留前3后4
func MaskPhone(phone string) string    // 138****5678
func MaskIDCard(id string) string      // 110***********1234

// 特化：保留首字符和域名
func MaskEmail(email string) string    // t***@example.com
```

### 3. 类型转换（convert.go）

遵循 Go 惯例：返回 `(value, error)` + Must 系列（失败返回默认值）：

```go
func ToInt(s string) (int, error)
func MustInt(s string, defaultVal int) int  // 失败返回 defaultVal
```

`ToBool` 支持多种真值：`"1","true","yes","on"` → true；`"0","false","no","off",""` → false

`ToString` 使用 `fmt.Sprintf("%v", v)` 实现任意类型转 string。

## 知识点总结

| 知识点 | 说明 |
|--------|------|
| rune vs byte | Go string 是 UTF-8，中文占 3 字节，必须用 `[]rune` 操作 |
| strings.Builder | 比 `+` 拼接高效，避免多次内存分配 |
| strings.NewReplacer | 批量字符串替换，比多次 `strings.Replace` 高效 |
| strconv 包 | 标准库类型转换：`Atoi`/`ParseFloat`/`ParseBool` |
| fmt.Sprintf | 万能格式化，`%v` 对任意类型有效 |
