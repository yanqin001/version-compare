package versionrange

import (
	"errors"
	"testing"
)

// TestEcosystemOrdering 覆盖通用比较曾误判的预发布、生态限定符、伪版本和正式版别名。
func TestEcosystemOrdering(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		scheme, left, right string
		want                int
	}{
		{"numeric", "1.0.10", "1.0.2", 1},
		{"numeric", "r1v60", "r1rv78", -1},
		{"numeric", "r1rv78-stable", "1.78", 0},
		{"numeric", "999999999999999999999999999999", "1000000000000000000000000000000", -1},
		{"semver", "2.1.13", "release-2.1.13-stable", 0},
		{"semver", "2.1.13", "RELEASE-V2.1.13-FINAL", 0},
		{"semver", "2.1.13", "v2.1.13-ga", 0},
		{"semver", "2.1.13", "v2.1.13-custom", 0},
		{"semver", "1.2.3-beta.2", "1.2.3-beta.11", -1},
		{"semver", "1.2.3+build.1", "1.2.3+build.2", 0},
		{"pep440", "3.15.0a1", "3.15.0rc3", -1},
		{"pep440", "1.0.dev1", "1.0a1", -1},
		{"pep440", "1.0", "1.0.post1", -1},
		{"pep440", "1!1.0", "2.0", 1},
		{"maven", "1.0.0.CR3", "1.0.0.Final", -1},
		{"maven", "1.0-SNAPSHOT", "1.0", -1},
		{"go", "0.0.0-20260101000000-aaaaaaaaaaaa", "v0.0.0-20260201000000-bbbbbbbbbbbb", -1},
		{"npm", "1.0.0-rc.1", "1.0.0", -1},
		{"cargo", "1.0.0-alpha.2", "1.0.0-alpha.10", -1},
		{"nuget", "1.0.0.1", "1.0.0.2", -1},
		{"rubygems", "1.0.0.rc1", "1.0.0", -1},
		{"composer", "1.0.0-RC1", "1.0.0", -1},
		{"calver", "2026.09.2", "2026.10.1", -1},
		{"calver", "2026-09-02", "2026-10-01", -1},
		{"dpkg", "1.0~rc1-1", "1.0-1", -1},
		{"rpm", "1.0~rc1-1", "1.0-1", -1},
		{"apk", "1.0-r1", "1.0-r2", -1},
		{"alpm", "1.0-1", "1.0-2", -1},
	} {
		t.Run(test.scheme+"/"+test.left, func(t *testing.T) {
			compare, err := NewComparator(Classification{RangeType: TypeVersion, VersionScheme: test.scheme})
			if err != nil {
				t.Fatal(err)
			}
			order, err := compare.Compare(test.left, test.right)
			if err != nil {
				t.Fatal(err)
			}
			if order != test.want {
				t.Fatalf("Compare() = %d, want %d", order, test.want)
			}
			reverse, err := compare.Compare(test.right, test.left)
			if err != nil {
				t.Fatal(err)
			}
			if reverse != -test.want {
				t.Fatalf("reverse Compare() = %d, want %d", reverse, -test.want)
			}
		})
	}
}

// TestDeclaredTagFormats 验证构建序号、运行时包装和命名发布序列，不对未知标签做词典序推断。
func TestDeclaredTagFormats(t *testing.T) {
	compare, err := NewComparator(Classification{RangeType: TypeVersion, VersionScheme: "numeric", RefKind: "tag", VersionPrefix: "b"})
	if err != nil {
		t.Fatal(err)
	}
	order, err := compare.Compare("b999", "b1000")
	if err != nil {
		t.Fatal(err)
	}
	if order != -1 {
		t.Fatalf("Compare() = %d, want -1", order)
	}
	_, err = compare.Compare("a999", "b1000")
	if !errors.Is(err, ErrUnordered) {
		t.Fatalf("error = %v, want ErrUnordered", err)
	}
	compare, err = NewComparator(Classification{RangeType: TypeVersion, VersionScheme: "semver", VersionSuffix: "(jdk8/11)"})
	if err != nil {
		t.Fatal(err)
	}
	order, err = compare.Compare("v2.6.0(jdk8/11)", "v2.7.0(jdk8/11)")
	if err != nil {
		t.Fatal(err)
	}
	if order != -1 {
		t.Fatalf("Compare() = %d, want -1", order)
	}
	_, err = compare.Compare("v2.6.0(jdk17)", "v2.7.0(jdk8/11)")
	if !errors.Is(err, ErrUnordered) {
		t.Fatalf("error = %v, want ErrUnordered", err)
	}
	compare, err = NewComparator(Classification{RangeType: TypeVersion, VersionScheme: "custom", VersionOrder: []string{"zeta", "alpha", "ulmo"}})
	if err != nil {
		t.Fatal(err)
	}
	order, err = compare.Compare("zeta", "alpha")
	if err != nil {
		t.Fatal(err)
	}
	if order != -1 {
		t.Fatalf("Compare() = %d, want -1", order)
	}
	_, err = compare.Compare("maple", "ulmo")
	if !errors.Is(err, ErrUnordered) {
		t.Fatalf("error = %v, want ErrUnordered", err)
	}
}

// TestClassificationRejectsAmbiguity 验证非法组合不能静默切换旧比较器，Git/未知明确保持不可排序。
func TestClassificationRejectsAmbiguity(t *testing.T) {
	for _, c := range []Classification{
		{RangeType: TypeVersion, VersionScheme: "typo"},
		{VersionScheme: "semver"},
		{RangeType: TypeVersion, VersionScheme: "semver", RefKind: "commit"},
		{RangeType: TypeGit, RefKind: "commit", VersionScheme: "numeric"},
		{RangeType: TypeVersion, VersionScheme: "custom", VersionOrder: []string{"a", "a"}},
	} {
		_, err := NewComparator(c)
		if err == nil {
			t.Fatal("expected error")
		}
		if errors.Is(err, ErrUnordered) {
			t.Fatalf("invalid classification returned ErrUnordered: %v", err)
		}
	}
	for _, c := range []Classification{{RangeType: TypeGit, RefKind: "commit"}, {RangeType: TypeUnknown}, {RangeType: TypeVersion, VersionScheme: "custom"}} {
		_, err := NewComparator(c)
		if !errors.Is(err, ErrUnordered) {
			t.Fatalf("error = %v, want ErrUnordered", err)
		}
	}
	compare, err := NewComparator(Classification{RangeType: TypeVersion, VersionScheme: "semver"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = compare.Compare("1.2.*", "1.2.3")
	if err == nil {
		t.Fatal("expected error")
	}
	_, err = compare.Compare("e59301bc55c9a4552c634d78a096c4b64c0dd9fc", "1.2.3")
	if !errors.Is(err, ErrUnordered) {
		t.Fatalf("error = %v, want ErrUnordered", err)
	}
}
