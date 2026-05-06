package collutil

// Unique 去重，保持原始顺序
// 利用 map 记录已出现的元素，跳过重复项
func Unique[T comparable](slice []T) []T {
	seen := make(map[T]struct{}, len(slice))
	var result []T
	for _, item := range slice {
		if _, exists := seen[item]; exists {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

// Contains 判断切片是否包含指定元素
func Contains[T comparable](slice []T, target T) bool {
	for _, item := range slice {
		if item == target {
			return true
		}
	}
	return false
}

// Intersect 计算两个切片的交集（去重，保持 a 中的顺序）
// 思路：先将 b 转为 set，再遍历 a 找出共同元素
func Intersect[T comparable](a, b []T) []T {
	// 将 b 的元素存入 set，用于 O(1) 查找
	bSet := toSet(b)

	// 遍历 a，保留在 b 中也存在的元素（用 seen 去重）
	var result []T
	added := make(map[T]struct{})
	for _, item := range a {
		if _, inB := bSet[item]; inB {
			if _, alreadyAdded := added[item]; !alreadyAdded {
				added[item] = struct{}{}
				result = append(result, item)
			}
		}
	}
	return result
}

// Union 计算两个切片的并集（去重，保持顺序）
// 先输出 a 的所有元素，再追加 b 中不在 a 里的元素
func Union[T comparable](a, b []T) []T {
	seen := make(map[T]struct{}, len(a)+len(b))

	var result []T
	// 先加入 a 的全部元素
	for _, item := range a {
		if _, exists := seen[item]; !exists {
			seen[item] = struct{}{}
			result = append(result, item)
		}
	}
	// 再加入 b 中未出现过的元素
	for _, item := range b {
		if _, exists := seen[item]; !exists {
			seen[item] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}

// Diff 计算差集：返回在 a 中但不在 b 中的元素
// 思路：先将 b 转为 set，再从 a 中排除存在于 set 的元素
func Diff[T comparable](a, b []T) []T {
	exclude := toSet(b)

	var result []T
	for _, item := range a {
		if _, shouldExclude := exclude[item]; !shouldExclude {
			result = append(result, item)
		}
	}
	return result
}

// toSet 将切片转为 map 集合，用于 O(1) 查找
func toSet[T comparable](slice []T) map[T]struct{} {
	set := make(map[T]struct{}, len(slice))
	for _, item := range slice {
		set[item] = struct{}{}
	}
	return set
}
