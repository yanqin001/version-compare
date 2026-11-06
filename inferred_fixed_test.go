package versioncompare

import (
	"errors"
	"testing"
)

// TestNormalizeInferredFixedBranch 验证只校正非法后缀，合法数字、字母补丁和标签包装分支均保持原文。
func TestNormalizeInferredFixedBranch(t *testing.T) {
	t.Parallel()
	for _, kind := range []VersionKind{NormalVersion, GenericOSVersion} {
		for _, test := range []struct{ branch, want string }{
			{"1.2", "1.2"}, {"1.10", "1.10"}, {"2.10s", "2.10s"},
			{" 01.010 ", " 01.010 "}, {"v1.10", "v1.10"}, {"r32", "r32"},
			{"1.10b1", "1.10"}, {"1.10rc1", "1.10"}, {"1.10beta2", "1.10"},
			{"1.10b1+build2", "1.10"},
		} {
			got, err := NormalizeInferredFixedBranch(test.branch, kind)
			if err != nil || got != test.want {
				t.Fatalf("kind=%d branch=%q: got %q,%v; want %q", kind, test.branch, got, err, test.want)
			}
		}
	}
	for _, test := range []struct {
		branch string
		kind   VersionKind
		want   error
	}{
		{"", NormalVersion, ErrEmptyVersion},
		{"main", NormalVersion, ErrInvalidExpression},
		{"???", NormalVersion, ErrInvalidVersion},
		{"1.10", VersionKind(255), ErrUnsupportedVersionType},
	} {
		if _, err := NormalizeInferredFixedBranch(test.branch, test.kind); !errors.Is(err, test.want) {
			t.Fatalf("branch=%q kind=%d: got %v; want %v", test.branch, test.kind, err, test.want)
		}
	}
}

// TestInferredFixedBranchOnlyTightensItsBranch 验证完整修复版本仍作为排他边界，其他维护分支不受影响。
func TestInferredFixedBranchOnlyTightensItsBranch(t *testing.T) {
	t.Parallel()
	branch, err := NormalizeInferredFixedBranch("1.10b1", NormalVersion)
	if err != nil {
		t.Fatal(err)
	}
	for _, expression := range []string{"[0.5,1.10b1)", "[0.5,2.0)"} {
		matcher, err := Compile(expression, CompileOptions{
			StrictBoundaries: true,
			FixedVersions:    []FixedVersion{{Branch: branch, Version: "1.10b1"}},
		})
		if err != nil {
			t.Fatal(err)
		}
		for version, want := range map[string]bool{"1.9": true, "1.10b0": true, "1.10b1": false, "1.10b2": false} {
			if got, err := matcher.Match(version); err != nil || got != want {
				t.Fatalf("expression=%q version=%q: %v,%v; want %v", expression, version, got, err, want)
			}
		}
		if expression == "[0.5,2.0)" {
			if got, err := matcher.Match("1.11"); err != nil || !got {
				t.Fatalf("another branch was tightened: %v,%v", got, err)
			}
		}
	}
}

// TestExplicitFixedBranchRemainsStrict 验证 Compile 不自动改写显式分支或放宽错误校验。
func TestExplicitFixedBranchRemainsStrict(t *testing.T) {
	t.Parallel()
	for _, fixed := range []FixedVersion{
		{Branch: "1.10b1", Version: "1.10b1"},
		{Branch: "1.9", Version: "1.10b1"},
	} {
		if _, err := Compile("[0.5,2.0)", CompileOptions{FixedVersions: []FixedVersion{fixed}}); !errors.Is(err, ErrInvalidExpression) {
			t.Fatalf("explicit mapping %#v: got %v; want ErrInvalidExpression", fixed, err)
		}
	}
}
