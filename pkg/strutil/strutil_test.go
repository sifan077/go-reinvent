package strutil

import "testing"

func TestSubstr(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		start  int
		length int
		want   string
	}{
		{"英文正常截取", "hello world", 0, 5, "hello"},
		{"英文从中间截取", "hello world", 6, 5, "world"},
		{"中文截取", "你好世界", 0, 2, "你好"},
		{"中文从中间截取", "你好世界", 2, 2, "世界"},
		{"负数起始", "hello", -3, 3, "llo"},
		{"超出长度", "hello", 0, 100, "hello"},
		{"空字符串", "", 0, 5, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Substr(tt.s, tt.start, tt.length); got != tt.want {
				t.Errorf("Substr(%q, %d, %d) = %q, want %q", tt.s, tt.start, tt.length, got, tt.want)
			}
		})
	}
}

func TestPadLeft(t *testing.T) {
	tests := []struct {
		name    string
		s       string
		length  int
		padChar rune
		want    string
	}{
		{"正常填充", "hello", 10, '*', "*****hello"},
		{"不需要填充", "hello", 3, '*', "hello"},
		{"中文填充", "你好", 5, '-', "---你好"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PadLeft(tt.s, tt.length, tt.padChar); got != tt.want {
				t.Errorf("PadLeft(%q, %d, %q) = %q, want %q", tt.s, tt.length, tt.padChar, got, tt.want)
			}
		})
	}
}

func TestPadRight(t *testing.T) {
	tests := []struct {
		name    string
		s       string
		length  int
		padChar rune
		want    string
	}{
		{"正常填充", "hello", 10, '*', "hello*****"},
		{"不需要填充", "hello", 3, '*', "hello"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PadRight(tt.s, tt.length, tt.padChar); got != tt.want {
				t.Errorf("PadRight(%q, %d, %q) = %q, want %q", tt.s, tt.length, tt.padChar, got, tt.want)
			}
		})
	}
}

func TestReverse(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want string
	}{
		{"英文", "hello", "olleh"},
		{"中文", "你好世界", "界世好你"},
		{"空字符串", "", ""},
		{"单字符", "a", "a"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Reverse(tt.s); got != tt.want {
				t.Errorf("Reverse(%q) = %q, want %q", tt.s, got, tt.want)
			}
		})
	}
}

func TestCamelToSnake(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want string
	}{
		{"简单驼峰", "helloWorld", "hello_world"},
		{"多段", "getUserNameById", "get_user_name_by_id"},
		{"连续大写缩写", "userID", "user_id"},
		{"全部小写", "hello", "hello"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CamelToSnake(tt.s); got != tt.want {
				t.Errorf("CamelToSnake(%q) = %q, want %q", tt.s, got, tt.want)
			}
		})
	}
}

func TestSnakeToCamel(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want string
	}{
		{"简单下划线", "hello_world", "helloWorld"},
		{"多段", "get_user_name", "getUserName"},
		{"全部小写", "hello", "hello"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := SnakeToCamel(tt.s); got != tt.want {
				t.Errorf("SnakeToCamel(%q) = %q, want %q", tt.s, got, tt.want)
			}
		})
	}
}

func TestCapitalize(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want string
	}{
		{"英文", "hello", "Hello"},
		{"中文", "你好", "你好"},
		{"空字符串", "", ""},
		{"已是大写", "Hello", "Hello"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Capitalize(tt.s); got != tt.want {
				t.Errorf("Capitalize(%q) = %q, want %q", tt.s, got, tt.want)
			}
		})
	}
}

func TestMask(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		start    int
		end      int
		maskChar rune
		want     string
	}{
		{"正常掩码", "1234567890", 3, 7, '*', "123****890"},
		{"全部掩码", "hello", 0, 5, '*', "*****"},
		{"不需要掩码", "hello", 0, 0, '*', "hello"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Mask(tt.s, tt.start, tt.end, tt.maskChar); got != tt.want {
				t.Errorf("Mask(%q, %d, %d, %q) = %q, want %q", tt.s, tt.start, tt.end, tt.maskChar, got, tt.want)
			}
		})
	}
}

func TestMaskPhone(t *testing.T) {
	tests := []struct {
		name  string
		phone string
		want  string
	}{
		{"标准手机号", "13812345678", "138****5678"},
		{"短号", "123", "123"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MaskPhone(tt.phone); got != tt.want {
				t.Errorf("MaskPhone(%q) = %q, want %q", tt.phone, got, tt.want)
			}
		})
	}
}

func TestMaskEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  string
	}{
		{"标准邮箱", "test@example.com", "t***@example.com"},
		{"单字符前缀", "a@b.com", "a***@b.com"},
		{"无@符号", "invalid", "invalid"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MaskEmail(tt.email); got != tt.want {
				t.Errorf("MaskEmail(%q) = %q, want %q", tt.email, got, tt.want)
			}
		})
	}
}

func TestMaskIDCard(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want string
	}{
		{"标准身份证", "110101199001011234", "110***********1234"},
		{"短号", "123", "123"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MaskIDCard(tt.id); got != tt.want {
				t.Errorf("MaskIDCard(%q) = %q, want %q", tt.id, got, tt.want)
			}
		})
	}
}

func TestToInt(t *testing.T) {
	tests := []struct {
		name    string
		s       string
		want    int
		wantErr bool
	}{
		{"正常数字", "123", 123, false},
		{"负数", "-42", -42, false},
		{"带空格", " 100 ", 100, false},
		{"非法字符", "abc", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToInt(tt.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("ToInt(%q) error = %v, wantErr %v", tt.s, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ToInt(%q) = %d, want %d", tt.s, got, tt.want)
			}
		})
	}
}

func TestMustInt(t *testing.T) {
	if got := MustInt("123", 0); got != 123 {
		t.Errorf("MustInt(\"123\", 0) = %d, want 123", got)
	}
	if got := MustInt("abc", 99); got != 99 {
		t.Errorf("MustInt(\"abc\", 99) = %d, want 99", got)
	}
}

func TestToFloat64(t *testing.T) {
	tests := []struct {
		name    string
		s       string
		want    float64
		wantErr bool
	}{
		{"正常小数", "3.14", 3.14, false},
		{"整数", "100", 100, false},
		{"非法字符", "abc", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToFloat64(tt.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("ToFloat64(%q) error = %v, wantErr %v", tt.s, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ToFloat64(%q) = %f, want %f", tt.s, got, tt.want)
			}
		})
	}
}

func TestToBool(t *testing.T) {
	tests := []struct {
		name    string
		s       string
		want    bool
		wantErr bool
	}{
		{"true", "true", true, false},
		{"1", "1", true, false},
		{"yes", "yes", true, false},
		{"on", "on", true, false},
		{"TRUE", "TRUE", true, false},
		{"false", "false", false, false},
		{"0", "0", false, false},
		{"no", "no", false, false},
		{"off", "off", false, false},
		{"空字符串", "", false, false},
		{"非法值", "maybe", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToBool(tt.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("ToBool(%q) error = %v, wantErr %v", tt.s, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ToBool(%q) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}

func TestToString(t *testing.T) {
	tests := []struct {
		name string
		v    any
		want string
	}{
		{"int", 123, "123"},
		{"float", 3.14, "3.14"},
		{"string", "hello", "hello"},
		{"bool", true, "true"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ToString(tt.v); got != tt.want {
				t.Errorf("ToString(%v) = %q, want %q", tt.v, got, tt.want)
			}
		})
	}
}
