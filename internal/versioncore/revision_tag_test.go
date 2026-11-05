package versioncore

import (
	"errors"
	"testing"
)

// TestRevisionTagParsing 验证两种通用规则共享 r 标签解析，后缀及原始输入不丢失。
func TestRevisionTagParsing(t *testing.T) {
	for _, kind := range []Kind{Normal, GenericOS} {
		for _, pair := range [][2]string{
			{"r36", "36"}, {"r36-p1", "36-p1"}, {" R36-p10 ", "36-p10"},
			{"r36.1-p2", "36.1-p2"}, {"r36_1-p2", "36.1-p2"},
			{"r36+vendor2", "36+vendor2"}, {"r36-rc1", "36-rc1"},
		} {
			left, err := Parse(pair[0], kind)
			if err != nil {
				t.Fatal(err)
			}
			right, err := Parse(pair[1], kind)
			if err != nil {
				t.Fatal(err)
			}
			if got, err := Compare(left, right, false); err != nil || got != 0 {
				t.Fatalf("kind=%d Compare(%q,%q)=%d,%v", kind, pair[0], pair[1], got, err)
			}
			if left.Original() != pair[0] {
				t.Fatalf("original input lost: %q", left.Original())
			}
		}
		for _, pair := range [][2]string{{"r36-p1", "r36-p2"}, {"r36-p2", "r36-p10"}, {"r36-p10", "r36"}, {"r36", "r37"}} {
			left, _ := Parse(pair[0], kind)
			right, _ := Parse(pair[1], kind)
			if got, err := Compare(left, right, false); err != nil || got != -1 {
				t.Fatalf("suffix order %v=%d,%v", pair, got, err)
			}
		}
	}
}

// TestRevisionTagPrefixDoesNotBroadenIdentifiers 验证其他标识不使用新转换，commit 和命名分支仍不透明。
func TestRevisionTagPrefixDoesNotBroadenIdentifiers(t *testing.T) {
	for _, value := range []string{"feature-r36-p1", "r36-p1/main", "r36--p1", "r36-p1?", "r1v60", "r1rv78", "r36feature", "e59301bc55c9a4552c634d78a096c4b64c0dd9fc"} {
		if got := stripRevisionTagPrefix(value); got != value {
			t.Fatalf("unexpected prefix conversion %q -> %q", value, got)
		}
	}
	for _, value := range []string{"feature-r36-p1", "r36-p1/main", "r36-p1?"} {
		if _, err := Parse(value, Normal); !errors.Is(err, ErrInvalidVersion) {
			t.Fatalf("Parse(%q)=%v", value, err)
		}
	}
	for _, value := range []string{"r36feature", "e59301bc55c9a4552c634d78a096c4b64c0dd9fc"} {
		parsed, err := Parse(value, Normal)
		if err != nil || !parsed.IsOpaque() {
			t.Fatalf("identifier lost opacity: %q: %v", value, err)
		}
	}
}
