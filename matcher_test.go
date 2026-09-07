package versioncompare

import (
	"errors"
	"testing"
)

func TestMatcherCompatibility(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		version    string
		expression string
		want       bool
	}{
		{name: "无界上限", version: "0.9.1", expression: "(,0.9.5]", want: true},
		{name: "预发布下界", version: "6.7-rc2", expression: "[6.7-rc1,6.7)", want: true},
		{name: "排除正式版", version: "6.7", expression: "[6.7-rc1,6.7)", want: false},
		{name: "OpenSSL 补丁", version: "1.1.1a", expression: "[1.1.1,1.1.1t)", want: true},
		{name: "OpenSSL 上界", version: "1.1.1t", expression: "[1.1.1,1.1.1t)", want: false},
		{name: "干净端点忽略后缀", version: "21.0.0-pre1", expression: "[21.0.0,21.0.0]", want: true},
		{name: "比较符", version: "4.0.6", expression: ">=4.0.0, <4.0.7", want: true},
		{name: "相邻区间", version: "3.1.2", expression: "[,1.0)[3.0,4.0)", want: true},
		{name: "空版本不匹配通配", version: "", expression: "*", want: false},
		{name: "通配符", version: "1.0", expression: "*", want: true},
		{name: "预发布版本包含", version: "1.1.2-rc2", expression: "[1.1.2-rc1,1.1.2-rc3]", want: true},
		{name: "预发布版本包含", version: "1.1.2-rc4", expression: "[1.1.2-rc1,1.1.2-rc3]", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			matcher, err := Compile(test.expression, CompileOptions{})
			if err != nil {
				t.Fatalf("Compile() error = %v", err)
			}
			got, err := matcher.Match(test.version)
			if err != nil {
				t.Fatalf("Match() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("Match() = %v, want %v", got, test.want)
			}
		})
	}
}

// TestMatcherStrictBoundaries 验证严格边界只关闭自动 MatchMain，且显式 MatchMain 仍优先。
func TestMatcherStrictBoundaries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		expression string
		version    string
		options    CompileOptions
		want       bool
	}{
		{
			name:       "默认精确值保留主版本兼容",
			expression: "[21.0.0]",
			version:    "21.0.0-pre1",
			want:       true,
		},
		{
			name:       "严格精确值比较后缀",
			expression: "[21.0.0]",
			version:    "21.0.0-pre1",
			options:    CompileOptions{StrictBoundaries: true},
			want:       false,
		},
		{
			name:       "严格精确值仍匹配正式版",
			expression: "[21.0.0]",
			version:    "21.0.0",
			options:    CompileOptions{StrictBoundaries: true},
			want:       true,
		},
		{
			name:       "严格区间下界比较后缀",
			expression: "[21.0.0,22.0.0)",
			version:    "21.0.0-pre1",
			options:    CompileOptions{StrictBoundaries: true},
			want:       false,
		},
		{
			name:       "显式主版本匹配优先于严格边界",
			expression: "[21.0.0]",
			version:    "21.0.0-pre1",
			options: CompileOptions{
				MatchMain:        true,
				StrictBoundaries: true,
			},
			want: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			matcher, err := Compile(test.expression, test.options)
			if err != nil {
				t.Fatalf("Compile() error = %v", err)
			}
			got, err := matcher.Match(test.version)
			if err != nil {
				t.Fatalf("Match() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("Match() = %v, want %v", got, test.want)
			}
		})
	}
}

// TestMatcherStrictBoundariesApplyToFixedVersion 验证固定修复版本也完整比较预发布后缀。
func TestMatcherStrictBoundariesApplyToFixedVersion(t *testing.T) {
	t.Parallel()

	options := CompileOptions{
		FixedVersions:    []FixedVersion{{Branch: "21.0", Version: "21.0.1"}},
		StrictBoundaries: true,
	}
	matcher, err := Compile("[20.0.0,22.0.0)", options)
	if err != nil {
		t.Fatal(err)
	}

	// 修复版本是排他上界，其预发布版本仍早于修复版本，应继续视为受影响。
	matched, err := matcher.Match("21.0.1-rc1")
	if err != nil {
		t.Fatal(err)
	}
	if !matched {
		t.Fatal("Match() = false, want true")
	}
}

func TestMatcherFixedVersion(t *testing.T) {
	t.Parallel()

	matcher, err := Compile("[3.13-rc1,4.3-rc1)", CompileOptions{
		FixedVersions: []FixedVersion{
			{Branch: "3.16", Version: "3.16.35"},
			{Branch: "3.18", Version: "3.18.54"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	for version, want := range map[string]bool{
		"3.16.34-rc1": true,
		"3.16.35":     false,
		"3.18.54":     false,
		"4.1.29":      true,
	} {
		got, matchErr := matcher.Match(version)
		if matchErr != nil {
			t.Fatalf("Match(%q) error = %v", version, matchErr)
		}
		if got != want {
			t.Errorf("Match(%q) = %v, want %v", version, got, want)
		}
	}
}

func TestVersionService(t *testing.T) {
	t.Parallel()

	service := NewVersionSvc()
	got, err := service.InExp(VersionExp{
		Version:    "4.14.271",
		Expression: "[4.0,5.0)",
		Fixed:      []string{"4.14:4.14.272"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Fatal("InExp() = false, want true")
	}
}

func TestWildcardStillValidatesCandidate(t *testing.T) {
	t.Parallel()

	matcher, err := Compile("*", CompileOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = matcher.Match("???"); !errors.Is(err, ErrInvalidVersion) {
		t.Fatalf("Match() error = %v, want ErrInvalidVersion", err)
	}
}

func TestExactOpaqueVersion(t *testing.T) {
	t.Parallel()

	matcher, err := Compile("master", CompileOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for version, want := range map[string]bool{"master": true, "main": false} {
		matched, matchErr := matcher.Match(version)
		if matchErr != nil {
			t.Fatalf("Match(%q) error = %v", version, matchErr)
		}
		if matched != want {
			t.Fatalf("Match(%q) = %v, want %v", version, matched, want)
		}
	}
}
