package main

import (
	"fmt"
	"go-reinvent/pkg/strutil"
)

func main() {
	fmt.Println("=== 字符串工具演示 ===")

	// 字符串截取
	fmt.Println("\n--- 截取 ---")
	fmt.Printf("Substr(\"hello world\", 0, 5) = %q\n", strutil.Substr("hello world", 0, 5))
	fmt.Printf("Substr(\"你好世界\", 0, 2) = %q\n", strutil.Substr("你好世界", 0, 2))

	// 填充
	fmt.Println("\n--- 填充 ---")
	fmt.Printf("PadLeft(\"42\", 5, '0') = %q\n", strutil.PadLeft("42", 5, '0'))
	fmt.Printf("PadRight(\"hi\", 5, '.') = %q\n", strutil.PadRight("hi", 5, '.'))

	// 翻转
	fmt.Println("\n--- 翻转 ---")
	fmt.Printf("Reverse(\"hello\") = %q\n", strutil.Reverse("hello"))
	fmt.Printf("Reverse(\"你好\") = %q\n", strutil.Reverse("你好"))

	// 命名转换
	fmt.Println("\n--- 命名转换 ---")
	fmt.Printf("CamelToSnake(\"helloWorld\") = %q\n", strutil.CamelToSnake("helloWorld"))
	fmt.Printf("CamelToSnake(\"userID\") = %q\n", strutil.CamelToSnake("userID"))
	fmt.Printf("SnakeToCamel(\"hello_world\") = %q\n", strutil.SnakeToCamel("hello_world"))

	// 掩码
	fmt.Println("\n--- 掩码 ---")
	fmt.Printf("MaskPhone(\"13812345678\") = %q\n", strutil.MaskPhone("13812345678"))
	fmt.Printf("MaskEmail(\"test@example.com\") = %q\n", strutil.MaskEmail("test@example.com"))
	fmt.Printf("MaskIDCard(\"110101199001011234\") = %q\n", strutil.MaskIDCard("110101199001011234"))

	// 类型转换
	fmt.Println("\n--- 类型转换 ---")
	fmt.Printf("MustInt(\"42\", 0) = %d\n", strutil.MustInt("42", 0))
	fmt.Printf("MustInt(\"abc\", -1) = %d\n", strutil.MustInt("abc", -1))
	fmt.Printf("MustFloat64(\"3.14\", 0) = %f\n", strutil.MustFloat64("3.14", 0))
	fmt.Printf("MustBool(\"yes\", false) = %v\n", strutil.MustBool("yes", false))
	fmt.Printf("MustBool(\"no\", true) = %v\n", strutil.MustBool("no", true))
	fmt.Printf("ToString(123) = %q\n", strutil.ToString(123))
}
