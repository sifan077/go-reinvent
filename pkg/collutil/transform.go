package collutil

// Map 将切片中的每个元素通过函数转换为新类型
func Map[T, R any](slice []T, fn func(T) R) []R {
	result := make([]R, len(slice))
	for i, v := range slice {
		result[i] = fn(v)
	}
	return result
}

// Reduce 将切片归约为单个值
func Reduce[T, R any](slice []T, init R, fn func(R, T) R) R {
	result := init
	for _, v := range slice {
		result = fn(result, v)
	}
	return result
}

// ForEach 遍历切片，对每个元素执行函数
func ForEach[T any](slice []T, fn func(T)) {
	for _, v := range slice {
		fn(v)
	}
}
