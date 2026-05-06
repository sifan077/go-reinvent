package collutil

// GroupBy 按指定函数将切片分组
func GroupBy[T any, K comparable](slice []T, fn func(T) K) map[K][]T {
	result := make(map[K][]T)
	for _, v := range slice {
		key := fn(v)
		result[key] = append(result[key], v)
	}
	return result
}

// Chunk 将切片按指定大小分块
// 最后一块可能不足 size 个元素
func Chunk[T any](slice []T, size int) [][]T {
	if size <= 0 {
		return nil
	}
	var result [][]T
	for i := 0; i < len(slice); i += size {
		end := i + size
		if end > len(slice) {
			end = len(slice)
		}
		result = append(result, slice[i:end])
	}
	return result
}

// Flatten 将二维切片展平为一维
func Flatten[T any](slices [][]T) []T {
	total := 0
	for _, s := range slices {
		total += len(s)
	}
	result := make([]T, 0, total)
	for _, s := range slices {
		result = append(result, s...)
	}
	return result
}

// ToMap 将切片转换为 map
// fn 返回 (key, value)，如果 key 重复则后者覆盖前者
func ToMap[T any, K comparable](slice []T, fn func(T) (K, T)) map[K]T {
	result := make(map[K]T, len(slice))
	for _, v := range slice {
		key, val := fn(v)
		result[key] = val
	}
	return result
}
