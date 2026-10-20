package versionrange

import (
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	versioncompare "github.com/yanqin001/version-compare"
)

// TestClassificationValidation 覆盖分类字段组合、旧 JSON 和可检查的公共错误契约。
func TestClassificationValidation(t *testing.T) {
	invalid := []Classification{
		{RangeType: "typo"},
		{RangeType: TypeVersion},
		{VersionScheme: "numeric"},
		{RangeType: TypeVersion, VersionScheme: "numeric", RefKind: "branch"},
		{RangeType: TypeVersion, VersionScheme: "numeric", RefKind: "typo"},
		{RangeType: TypeGit},
		{RangeType: TypeUnknown, VersionPrefix: "b"},
		{RangeType: TypeVersion, VersionScheme: "numeric", VersionOrder: []string{"1", "2"}},
		{RangeType: TypeVersion, VersionScheme: "custom", VersionPrefix: "tag-"},
		{RangeType: TypeVersion, VersionScheme: "custom", VersionSuffix: "-tag"},
		{RangeType: TypeVersion, VersionScheme: "custom", VersionOrder: []string{"a", "a"}},
		{RangeType: TypeVersion, VersionScheme: "custom", VersionOrder: []string{""}},
		{RangeType: TypeVersion, VersionScheme: "custom", VersionOrder: []string{" a"}},
	}
	for _, c := range invalid {
		if err := c.Validate(); !errors.Is(err, ErrInvalidClassification) {
			t.Errorf("Validate(%+v) = %v, want ErrInvalidClassification", c, err)
		}
	}
	var legacy Classification
	if err := json.Unmarshal([]byte(`{"begin":"1.0","end":"2.0"}`), &legacy); err != nil {
		t.Fatal(err)
	}
	if legacy.Present() || legacy.Validate() != nil || legacy.ValidateRange("old", "tags", false) != nil {
		t.Fatal("empty classification must remain distinguishable from an explicit policy")
	}
	if _, err := NewComparator(legacy); !errors.Is(err, ErrInvalidClassification) {
		t.Fatalf("NewComparator(empty) = %v, want ErrInvalidClassification", err)
	}
	var typed Classification
	if err := json.Unmarshal([]byte(`{"range_type":"version","version_scheme":"NUMERIC","ref_kind":"tag","version_prefix":"b"}`), &typed); err != nil {
		t.Fatal(err)
	}
	compare, err := NewComparator(typed)
	if err != nil {
		t.Fatal(err)
	}
	if order, err := compare.Compare("b9", "b10"); err != nil || order != -1 {
		t.Fatalf("canonical JSON comparison = %d, %v", order, err)
	}
}

// TestRangeValidation 检查结构化区间不依赖表达式语法，并保留未知与无效范围的区别。
func TestRangeValidation(t *testing.T) {
	c := Classification{RangeType: TypeVersion, VersionScheme: "numeric"}
	for _, test := range []struct {
		begin, end string
		inclusive  bool
		want       error
	}{
		{"1", "2", false, nil},
		{"1", "1.0", true, nil},
		{"1", "1.0", false, versioncompare.ErrInvalidExpression},
		{"2", "1", true, versioncompare.ErrInvalidExpression},
		{"a", "2", false, versioncompare.ErrInvalidVersion},
		{"", "2", false, ErrUnordered},
	} {
		if err := c.ValidateRange(test.begin, test.end, test.inclusive); !errors.Is(err, test.want) {
			t.Errorf("ValidateRange(%q, %q, %t) = %v, want %v", test.begin, test.end, test.inclusive, err, test.want)
		}
	}
	for _, c := range []Classification{{RangeType: TypeUnknown}, {RangeType: TypeGit, RefKind: "commit"}} {
		if err := c.ValidateRange("a", "b", false); err != nil {
			t.Fatal(err)
		}
		if _, err := NewComparator(c); !errors.Is(err, versioncompare.ErrIncomparableVersion) {
			t.Fatalf("unknown ordering must retain legacy error identity: %v", err)
		}
	}
	c = Classification{RangeType: TypeVersion, VersionScheme: "semver", VersionSuffix: "(jdk8/11)"}
	if err := c.ValidateRange("v2.6.0(jdk8/11)", "v2.7.0(jdk8/11)", false); err != nil {
		t.Fatal(err)
	}
}

// TestComparatorErrors 验证脏版本可用 errors.Is 识别，且未初始化比较器不会 panic。
func TestComparatorErrors(t *testing.T) {
	for _, c := range []*Comparator{nil, {}} {
		if _, err := c.Compare("1", "2"); !errors.Is(err, ErrInvalidClassification) {
			t.Fatalf("zero comparator error = %v", err)
		}
		if c.InBranch("1.2.3", "1.2") {
			t.Fatal("uninitialized comparator matched a branch")
		}
	}
	for _, scheme := range []string{"numeric", "calver", "semver", "pep440", "dpkg"} {
		c, err := NewComparator(Classification{RangeType: TypeVersion, VersionScheme: scheme})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := c.Compare("1.*", "1.2"); !errors.Is(err, versioncompare.ErrInvalidVersion) {
			t.Errorf("%s invalid version error = %v", scheme, err)
		}
		for _, hash := range []string{strings.Repeat("a", 40), strings.Repeat("B", 64)} {
			if _, err := c.Compare(hash, "1.2"); !errors.Is(err, ErrUnordered) {
				t.Errorf("%s commit comparison error = %v", scheme, err)
			}
		}
	}
}

// TestComparatorOwnershipAndConcurrency 验证自定义顺序不持有输入切片，且可安全并发复用。
func TestComparatorOwnershipAndConcurrency(t *testing.T) {
	input := Classification{RangeType: TypeVersion, VersionScheme: "custom", VersionOrder: []string{"zeta", "alpha", "ulmo"}}
	c, err := NewComparator(input)
	if err != nil {
		t.Fatal(err)
	}
	input.VersionOrder[0] = "changed"
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				if order, err := c.Compare("zeta", "alpha"); err != nil || order != -1 {
					t.Errorf("concurrent comparison = %d, %v", order, err)
				}
			}
		}()
	}
	wg.Wait()
}

// TestNumericBranchBoundaries 防止维护分支 1.2 错误覆盖 1.20，并覆盖正式版标签归一化。
func TestNumericBranchBoundaries(t *testing.T) {
	c, err := NewComparator(Classification{RangeType: TypeVersion, VersionScheme: "semver"})
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		version, branch string
		want            bool
	}{
		{"release-1.2.3-stable", "1.2", true},
		{"1.2.0-rc1", "1.2", true},
		{"1.20.0", "1.2", false},
		{"1.2.3", "main", false},
	} {
		if got := c.InBranch(test.version, test.branch); got != test.want {
			t.Errorf("InBranch(%q, %q) = %t", test.version, test.branch, got)
		}
	}
}
