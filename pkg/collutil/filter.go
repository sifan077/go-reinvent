package collutil

// Filter 过滤切片，保留满足条件的元素
func Filter[T any](slice []T, fn func(T) bool) []T {
	var result []T
	for _, v := range slice {
		if fn(v) {
			result = append(result, v)
		}
	}
	return result
}

// Reject 反向过滤，移除满足条件的元素
func Reject[T any](slice []T, fn func(T) bool) []T {
	return Filter(slice, func(t T) bool { return !fn(t) })
}

// Find 查找首个满足条件的元素
func Find[T any](slice []T, fn func(T) bool) (T, bool) {
	for _, v := range slice {
		if fn(v) {
			return v, true
		}
	}
	var zero T
	return zero, false
}

// Every 判断是否所有元素都满足条件
func Every[T any](slice []T, fn func(T) bool) bool {
	for _, v := range slice {
		if !fn(v) {
			return false
		}
	}
	return true
}

// Some 判断是否存在任一元素满足条件
func Some[T any](slice []T, fn func(T) bool) bool {
	for _, v := range slice {
		if fn(v) {
			return true
		}
	}
	return false
}
