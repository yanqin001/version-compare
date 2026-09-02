package versioncore

import (
	"errors"
	"testing"
)

func TestParseAndCompare(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		left      string
		right     string
		matchMain bool
		want      int
	}{
		{name: "数字段", left: "1.0.10", right: "1.0.9", want: 1},
		{name: "尾零等价", left: "21", right: "21.0.0", want: 0},
		{name: "预发布早于正式版", left: "6.7-rc1", right: "6.7", want: -1},
		{name: "忽略预发布", left: "6.7-rc1", right: "6.7", matchMain: true, want: 0},
		{name: "OpenSSL 补丁字母", left: "1.1.1a", right: "1.1.1", want: 1},
		{name: "OpenSSL 多字母补丁", left: "1.0.2zj", right: "1.0.2z", want: 1},
		{name: "预发布数字自然排序", left: "1.0-rc10", right: "1.0-rc2", want: 1},
		{name: "npm 前缀", left: "@mattermost/types@9.0.0", right: "8.1.3", want: 1},
		{name: "Conan 后缀", left: "1.5.2@jgsogo/stable#0", right: "1.5.1", want: 1},
		{name: "连字符数字标签", left: "VER-2-2-0", right: "2.1.9", want: 1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			left, err := Parse(test.left, Normal)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", test.left, err)
			}
			right, err := Parse(test.right, Normal)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", test.right, err)
			}
			got, err := Compare(left, right, test.matchMain)
			if err != nil {
				t.Fatalf("Compare() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("Compare() = %d, want %d", got, test.want)
			}
		})
	}
}

func TestOpaqueVersions(t *testing.T) {
	t.Parallel()

	left, err := Parse("master", Normal)
	if err != nil {
		t.Fatal(err)
	}
	right, err := Parse("main", Normal)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = Compare(left, right, false); !errors.Is(err, ErrIncomparable) {
		t.Fatalf("Compare() error = %v, want ErrIncomparable", err)
	}
}

func TestHasMainPrefix(t *testing.T) {
	t.Parallel()

	version, _ := Parse("4.14.271-rc1", Normal)
	branch, _ := Parse("4.14", Normal)
	if !HasMainPrefix(version, branch) {
		t.Fatal("HasMainPrefix() = false, want true")
	}
}
