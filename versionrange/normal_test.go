package versionrange

import (
	"errors"
	"testing"

	versioncompare "github.com/yanqin001/version-compare"
)

// TestNormalAndHistoricalAliases 验证九个历史策略统一为 normal，原始字母、预发布与构建后缀完整参与排序。
func TestNormalAndHistoricalAliases(t *testing.T) {
	for _, scheme := range []string{SchemeNormal, "numeric", "release", "semver", "calver", "go", "npm", "cargo", "nuget", "composer"} {
		classification := Classification{RangeType: TypeVersion, VersionScheme: scheme, VersionPrefix: "v"}
		if got := classification.Canonical().VersionScheme; got != SchemeNormal {
			t.Fatalf("%s canonical = %s", scheme, got)
		}
		c, err := NewComparator(classification)
		if err != nil {
			t.Fatal(err)
		}
		for _, test := range []struct {
			a, b string
			want int
		}{
			{"v2.10s", "2.10r", 1}, {"v2.10s", "2.11", -1},
			{"v2.13-pre4", "2.13-pre10", -1}, {"2.13-pre4", "v2.13", -1},
			{"v2.27-rc2", "2.27-rc10", -1}, {"2.27.1", "v2.27", 1},
			{"2.27-vendor2", "2.27-vendor10", -1}, {"2.27", "v2.27.0", 0},
			{"1.2.3+build.1", "1.2.3+build.2", -1},
			{"release-2.1.13-stable", "2.1.13", 0}, {"r1rv78", "r1v60", 1},
		} {
			got, err := c.Compare(test.a, test.b)
			if err != nil || got != test.want {
				t.Fatalf("%s Compare(%q,%q) = %d,%v; want %d", scheme, test.a, test.b, got, err, test.want)
			}
		}
		if !c.InBranch("release-2.27.1s", "2.27") || c.InBranch("2.270.1", "2.27") {
			t.Fatal("normal maintenance branch mismatch")
		}
	}
}

// TestNormalRangeAndProjectGate 验证坐标范围比较完整后缀，而根包显式 MatchMain 仍可用于项目门禁。
func TestNormalRangeAndProjectGate(t *testing.T) {
	c, err := NewComparator(Classification{RangeType: TypeVersion, VersionScheme: SchemeNormal})
	if err != nil {
		t.Fatal(err)
	}
	if got, err := c.Compare("2.27-rc1", "2.27"); err != nil || got != -1 {
		t.Fatalf("suffix comparison = %d,%v", got, err)
	}
	for _, main := range []bool{false, true} {
		m, err := versioncompare.Compile("[2.27]", versioncompare.CompileOptions{StrictBoundaries: true, MatchMain: main})
		if err != nil {
			t.Fatal(err)
		}
		if got, err := m.Match("2.27-rc1"); err != nil || got != main {
			t.Fatalf("MatchMain=%v: %v,%v", main, got, err)
		}
	}
	if err := (Classification{RangeType: TypeVersion, VersionScheme: SchemeNormal}).ValidateRange("2.27", "2.27-rc1", false); !errors.Is(err, versioncompare.ErrInvalidExpression) {
		t.Fatalf("inverted range error = %v", err)
	}
}
