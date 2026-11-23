package versioncompare

import (
	"errors"
	"testing"
)

// TestArmReleaseRanges 验证 rMpN 区间按两段数字排序，项目主版本模式也保留 p 段及原有开闭边界。
func TestArmReleaseRanges(t *testing.T) {
	for _, kind := range []VersionKind{NormalVersion, GenericOSVersion} {
		for _, test := range []struct {
			expression      string
			inside, outside []string
		}{
			{"[r4p0,r31p0)", []string{"r4p0", "r26p0-01rel0", "r30p0-01eac0"}, []string{"r3p99", "r31p0", "r31p0-01rel0"}},
			{"[r4p0,r31p0]", []string{"r31p0", "R31P0-01REL0"}, []string{"r31p1", "r32p0"}},
			{"[r8p0,r31p0)", []string{"r9p0", "r10p0"}, []string{"r7p9", "r31p0"}},
			{"[r12p0,r32p0]", []string{"r12p0", "r32p0"}, []string{"r11p9", "r32p1"}},
			{"[r26p0,r32p0)", []string{"r26p0-01rel0", "r31p1"}, []string{"r25p9", "r32p0"}},
			{"[r10p2,r10p10)", []string{"r10p2", "r10p9"}, []string{"r10p1", "r10p10"}},
			{"(,r4p0)", []string{"r3p9"}, []string{"r4p0", "3.9"}},
			{"[r31p0,)", []string{"r31p0-01rel0", "r99p1"}, []string{"r30p9", "99.1"}},
			{"r31p0", []string{"r31p0", "r31p0-01eac0"}, []string{"31.0", "r31-p0", "r31p0-rc1", "r31p0-01rel0/file", "feature/r31p0"}},
			{"[1,40] || [r4p0,r31p0)", []string{"31.0", "r30p0-01eac0"}, []string{"r31p0", "r40p0-unknown", "41.0"}},
			{"[r4p0,r31p0) || [1,40]", []string{"31.0", "r30p0-01eac0"}, []string{"r31p0", "r40p0-unknown", "41.0"}},
		} {
			matcher, err := Compile(test.expression, CompileOptions{Kind: kind, MatchMain: true})
			if err != nil {
				t.Fatalf("kind=%d Compile(%q): %v", kind, test.expression, err)
			}
			for want, candidates := range map[bool][]string{true: test.inside, false: test.outside} {
				for _, candidate := range candidates {
					if got, err := matcher.Match(candidate); err != nil || got != want {
						t.Errorf("kind=%d range=%s candidate=%s: %t, %v; want %t", kind, test.expression, candidate, got, err, want)
					}
				}
			}
		}
	}
}

// TestArmReleaseBoundsRejectAmbiguousVersions 验证混合端点、逆序和带包后缀的边界不会被猜测成合法范围。
func TestArmReleaseBoundsRejectAmbiguousVersions(t *testing.T) {
	for expression, want := range map[string]error{
		"[r4p0,31.0]":                ErrIncomparableVersion,
		"[r31p0,r4p0]":               ErrInvalidExpression,
		"[r4p0,r31p0-01rel0)":        ErrInvalidExpression,
		"[r4p0-unknown,r31p0)":       ErrInvalidExpression,
		"[r4p0-01eac0,r31p0-02rel0]": ErrInvalidExpression,
	} {
		if _, err := Compile(expression, CompileOptions{}); !errors.Is(err, want) {
			t.Errorf("Compile(%q): %v; want %v", expression, err, want)
		}
	}
}

// TestArmReleaseRangesKeepFixedNamespaces 验证普通版本的修复记录只收紧普通版本，不能裁剪同数字的 rMpN 发布。
func TestArmReleaseRangesKeepFixedNamespaces(t *testing.T) {
	matcher, err := Compile("[r26p0,r32p0] || [26,32]", CompileOptions{
		FixedVersions: []FixedVersion{{Branch: "30", Version: "30.0"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	for candidate, want := range map[string]bool{"30.0": false, "r30p0-01eac0": true} {
		if got, err := matcher.Match(candidate); err != nil || got != want {
			t.Errorf("Match(%q): %t, %v; want %t", candidate, got, err, want)
		}
	}
}
