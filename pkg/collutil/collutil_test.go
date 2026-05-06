package collutil

import (
	"reflect"
	"strings"
	"testing"
)

func TestFilter(t *testing.T) {
	tests := []struct {
		name string
		s    []int
		fn   func(int) bool
		want []int
	}{
		{"过滤偶数", []int{1, 2, 3, 4, 5, 6}, func(n int) bool { return n%2 == 0 }, []int{2, 4, 6}},
		{"全部满足", []int{2, 4, 6}, func(n int) bool { return n%2 == 0 }, []int{2, 4, 6}},
		{"都不满足", []int{1, 3, 5}, func(n int) bool { return n%2 == 0 }, nil},
		{"空切片", nil, func(n int) bool { return true }, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Filter(tt.s, tt.fn); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Filter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReject(t *testing.T) {
	got := Reject([]int{1, 2, 3, 4, 5}, func(n int) bool { return n%2 == 0 })
	want := []int{1, 3, 5}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Reject() = %v, want %v", got, want)
	}
}

func TestFind(t *testing.T) {
	got, ok := Find([]string{"a", "b", "c"}, func(s string) bool { return s == "b" })
	if !ok || got != "b" {
		t.Errorf("Find() = %v, %v, want b, true", got, ok)
	}

	_, ok = Find([]string{"a", "b", "c"}, func(s string) bool { return s == "d" })
	if ok {
		t.Error("Find() should return false for missing element")
	}
}

func TestEvery(t *testing.T) {
	if !Every([]int{2, 4, 6}, func(n int) bool { return n%2 == 0 }) {
		t.Error("Every() should return true for all even numbers")
	}
	if Every([]int{2, 3, 6}, func(n int) bool { return n%2 == 0 }) {
		t.Error("Every() should return false when not all elements satisfy condition")
	}
}

func TestSome(t *testing.T) {
	if !Some([]int{1, 3, 4}, func(n int) bool { return n%2 == 0 }) {
		t.Error("Some() should return true when at least one element satisfies condition")
	}
	if Some([]int{1, 3, 5}, func(n int) bool { return n%2 == 0 }) {
		t.Error("Some() should return false when no elements satisfy condition")
	}
}

func TestMap(t *testing.T) {
	got := Map([]int{1, 2, 3}, func(n int) string { return strings.Repeat("x", n) })
	want := []string{"x", "xx", "xxx"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Map() = %v, want %v", got, want)
	}
}

func TestReduce(t *testing.T) {
	sum := Reduce([]int{1, 2, 3, 4}, 0, func(acc, n int) int { return acc + n })
	if sum != 10 {
		t.Errorf("Reduce() = %d, want 10", sum)
	}

	concat := Reduce([]string{"a", "b", "c"}, "", func(acc, s string) string { return acc + s })
	if concat != "abc" {
		t.Errorf("Reduce() = %q, want %q", concat, "abc")
	}
}

func TestForEach(t *testing.T) {
	var result []int
	ForEach([]int{1, 2, 3}, func(n int) { result = append(result, n*2) })
	want := []int{2, 4, 6}
	if !reflect.DeepEqual(result, want) {
		t.Errorf("ForEach() result = %v, want %v", result, want)
	}
}

func TestUnique(t *testing.T) {
	tests := []struct {
		name string
		s    []int
		want []int
	}{
		{"有重复", []int{1, 2, 2, 3, 3, 3}, []int{1, 2, 3}},
		{"无重复", []int{1, 2, 3}, []int{1, 2, 3}},
		{"空切片", nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Unique(tt.s); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Unique() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContains(t *testing.T) {
	if !Contains([]int{1, 2, 3}, 2) {
		t.Error("Contains() should return true for existing element")
	}
	if Contains([]int{1, 2, 3}, 4) {
		t.Error("Contains() should return false for non-existing element")
	}
}

func TestIntersect(t *testing.T) {
	tests := []struct {
		name string
		a, b []int
		want []int
	}{
		{"有交集", []int{1, 2, 3, 4}, []int{3, 4, 5, 6}, []int{3, 4}},
		{"无交集", []int{1, 2}, []int{3, 4}, nil},
		{"完全相同", []int{1, 2, 3}, []int{1, 2, 3}, []int{1, 2, 3}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Intersect(tt.a, tt.b); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Intersect() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestUnion(t *testing.T) {
	got := Union([]int{1, 2, 3}, []int{3, 4, 5})
	want := []int{1, 2, 3, 4, 5}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Union() = %v, want %v", got, want)
	}
}

func TestDiff(t *testing.T) {
	tests := []struct {
		name string
		a, b []int
		want []int
	}{
		{"有差集", []int{1, 2, 3, 4}, []int{3, 4, 5, 6}, []int{1, 2}},
		{"无差集", []int{1, 2, 3}, []int{1, 2, 3}, nil},
		{"完全不重叠", []int{1, 2}, []int{3, 4}, []int{1, 2}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Diff(tt.a, tt.b); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Diff() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGroupBy(t *testing.T) {
	// 按奇偶分组
	got := GroupBy([]int{1, 2, 3, 4, 5}, func(n int) string {
		if n%2 == 0 {
			return "even"
		}
		return "odd"
	})
	if len(got) != 2 {
		t.Errorf("GroupBy() should have 2 groups, got %d", len(got))
	}
	if !reflect.DeepEqual(got["odd"], []int{1, 3, 5}) {
		t.Errorf("GroupBy() odd = %v, want [1 3 5]", got["odd"])
	}
	if !reflect.DeepEqual(got["even"], []int{2, 4}) {
		t.Errorf("GroupBy() even = %v, want [2 4]", got["even"])
	}
}

func TestChunk(t *testing.T) {
	tests := []struct {
		name string
		s    []int
		size int
		want [][]int
	}{
		{"正常分块", []int{1, 2, 3, 4, 5}, 2, [][]int{{1, 2}, {3, 4}, {5}}},
		{"整除", []int{1, 2, 3, 4}, 2, [][]int{{1, 2}, {3, 4}}},
		{"size大于长度", []int{1, 2}, 5, [][]int{{1, 2}}},
		{"size为0", []int{1, 2}, 0, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Chunk(tt.s, tt.size); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Chunk() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFlatten(t *testing.T) {
	got := Flatten([][]int{{1, 2}, {3}, {4, 5, 6}})
	want := []int{1, 2, 3, 4, 5, 6}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Flatten() = %v, want %v", got, want)
	}
}

func TestToMap(t *testing.T) {
	type User struct {
		ID   int
		Name string
	}
	users := []User{{1, "Alice"}, {2, "Bob"}, {3, "Charlie"}}
	got := ToMap(users, func(u User) (int, User) { return u.ID, u })
	if got[1].Name != "Alice" || got[2].Name != "Bob" || got[3].Name != "Charlie" {
		t.Errorf("ToMap() = %v, want correct mapping", got)
	}
}
