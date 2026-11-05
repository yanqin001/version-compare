package versioncompare

import "testing"

// TestRevisionTagRanges 验证精确值、区间、固定修复边界与项目门禁均支持 r 标签。
func TestRevisionTagRanges(t *testing.T) {
	for _, kind := range []VersionKind{NormalVersion, GenericOSVersion} {
		for _, test := range []struct {
			expression      string
			options         CompileOptions
			inside, outside []string
		}{
			{"r36-p1", CompileOptions{StrictBoundaries: true}, []string{"r36-p1", "R36-p1", "36-p1"}, []string{"r36", "r36-p2"}},
			{"[r36-p1,r36-p10)", CompileOptions{StrictBoundaries: true}, []string{"r36-p1", "r36-p2", "r36-p9"}, []string{"r35", "r36-p0", "r36-p10", "r37"}},
			{"[r36-p1]", CompileOptions{MatchMain: true}, []string{"r36", "r36-p2", "r36-p10"}, []string{"r37-p1"}},
			{"[r36-p1,r37)", CompileOptions{StrictBoundaries: true, FixedVersions: []FixedVersion{{Branch: "r36", Version: "r36-p2"}}}, []string{"r36-p1"}, []string{"r36-p2", "r36-p10", "r36", "r37"}},
		} {
			test.options.Kind = kind
			matcher, err := Compile(test.expression, test.options)
			if err != nil {
				t.Fatal(err)
			}
			for _, version := range test.inside {
				if got, err := matcher.Match(version); err != nil || !got {
					t.Fatalf("kind=%d expression=%q candidate=%q: %v,%v", kind, test.expression, version, got, err)
				}
			}
			for _, version := range test.outside {
				if got, err := matcher.Match(version); err != nil || got {
					t.Fatalf("kind=%d expression=%q candidate=%q: %v,%v", kind, test.expression, version, got, err)
				}
			}
		}
	}
}
