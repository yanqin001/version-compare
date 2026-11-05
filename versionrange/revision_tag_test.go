package versionrange

import (
	"errors"
	"testing"

	versioncompare "github.com/yanqin001/version-compare"
)

// TestNormalRevisionTags 验证结构化 normal 范围复用相同解析，不把 r 标签留在旧的不可排序路径。
func TestNormalRevisionTags(t *testing.T) {
	classification := Classification{RangeType: TypeVersion, VersionScheme: SchemeNormal}
	comparator, err := NewComparator(classification)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		a, b string
		want int
	}{
		{"r36-p1", "36-p1", 0}, {"r36-p2", "r36-p10", -1}, {"r36-p1", "r36", -1}, {"r36", "r37", -1},
	} {
		if got, err := comparator.Compare(test.a, test.b); err != nil || got != test.want {
			t.Fatalf("Compare(%q,%q)=%d,%v", test.a, test.b, got, err)
		}
	}
	if err := classification.ValidateRange("r36-p1", "r36-p10", false); err != nil {
		t.Fatal(err)
	}
	if !comparator.InBranch("r36-p1", "r36") {
		t.Fatal("r36 branch not recognized")
	}
	if _, err := comparator.Compare("r36feature", "r36-p1"); !errors.Is(err, versioncompare.ErrInvalidVersion) {
		t.Fatalf("named branch unexpectedly sortable: %v", err)
	}
	if _, err := comparator.Compare("e59301bc55c9a4552c634d78a096c4b64c0dd9fc", "r36-p1"); !errors.Is(err, ErrUnordered) {
		t.Fatalf("commit unexpectedly sortable: %v", err)
	}
}
