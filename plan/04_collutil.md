# 泛型集合工具库 · 设计规划

## 功能清单

- [x] 过滤：Filter / Reject / Find / Every / Some
- [x] 转换：Map / Reduce / ForEach
- [x] 集合运算：Unique / Contains / Intersect / Union / Diff
- [x] 分组与变换：GroupBy / Chunk / Flatten / ToMap

## 目录结构

```
pkg/collutil/
├── filter.go       # 过滤：Filter/Reject/Find/Every/Some
├── transform.go    # 转换：Map/Reduce/ForEach
├── set.go          # 集合：Unique/Contains/Intersect/Union/Diff
├── group.go        # 分组：GroupBy/Chunk/Flatten/ToMap
└── collutil_test.go
cmd/collutil/
└── main.go
```

## 核心设计

### 1. 泛型基础语法

Go 1.18+ 泛型使用类型参数约束：

```go
// T any — 任意类型
func Filter[T any](slice []T, fn func(T) bool) []T

// T comparable — 可比较类型（支持 == 和 !=），可用作 map key
func Unique[T comparable](slice []T) []T

// 多类型参数
func Map[T, R any](slice []T, fn func(T) R) []R

// 混合约束
func GroupBy[T any, K comparable](slice []T, fn func(T) K) map[K][]T
```

### 2. 函数式操作（filter.go + transform.go）

```go
// 过滤：保留满足条件的元素
Filter([]int{1,2,3,4,5}, func(n int) bool { return n%2 == 0 })
// → [2, 4]

// 映射：类型转换
Map([]int{1,2,3}, func(n int) string { return strconv.Itoa(n) })
// → ["1", "2", "3"]

// 归约：聚合
Reduce([]int{1,2,3,4}, 0, func(acc, n int) int { return acc + n })
// → 10
```

### 3. 集合运算（set.go）

核心思路：用 `map[T]struct{}` 做 O(1) 查找，比双重循环 O(n²) 高效。

```go
// 去重：保持顺序
func Unique[T comparable](slice []T) []T {
    seen := make(map[T]struct{}, len(slice))
    var result []T
    for _, v := range slice {
        if _, ok := seen[v]; !ok {
            seen[v] = struct{}{}
            result = append(result, v)
        }
    }
    return result
}
```

| 函数 | 说明 | 时间复杂度 |
|------|------|-----------|
| `Unique` | 去重，保持顺序 | O(n) |
| `Contains` | 包含判断 | O(n) |
| `Intersect` | 交集 | O(n+m) |
| `Union` | 并集 | O(n+m) |
| `Diff` | 差集（在 a 不在 b） | O(n+m) |

### 4. 分组与变换（group.go）

```go
// 按条件分组
GroupBy([]int{1,2,3,4}, func(n int) string {
    if n%2 == 0 { return "偶" }
    return "奇"
})
// → map[偶:[2 4] 奇:[1 3]]

// 分块
Chunk([]int{1,2,3,4,5}, 2)
// → [[1 2] [3 4] [5]]

// 切片转 map
ToMap(users, func(u User) (int, User) { return u.ID, u })
// → map[1:{Alice} 2:{Bob}]
```

## 知识点总结

| 知识点 | 说明 |
|--------|------|
| `[T any]` | 最宽松约束，任意类型 |
| `[T comparable]` | 可比较类型，支持 `==`/`!=`，可做 map key |
| 多类型参数 `[T, R any]` | Map 函数输入输出类型不同 |
| `map[T]struct{}` | 集合实现最佳实践，value 为空结构体不占内存 |
| 函数作为参数 | Go 一等公民，配合泛型实现函数式编程 |
| 零值 `var zero T` | 泛型函数中获取类型的零值 |
