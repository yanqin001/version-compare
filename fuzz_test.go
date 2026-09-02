package versioncompare

import (
	"testing"

	"github.com/yanqin001/version-compare/internal/rangeexpr"
)

func FuzzRangeExpressionParseDoesNotPanic(f *testing.F) {
	seeds := []string{
		"*",
		"1.2.3",
		"[1.0,2.0)",
		">=1.0, <2.0",
		"<2.0 >=1.0",
		"[,1.0)[2.0,3.0)",
		"1.0,2.0,v3",
		"",
		"=>1",
		"[1,[2]]",
		"\x00\xff||(",
		"版本-一",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, expression string) {
		_, _ = rangeexpr.Parse(expression)
	})
}

func FuzzCompileAndMatchDoNotPanic(f *testing.F) {
	seeds := []struct {
		expression string
		version    string
		kind       uint8
		matchMain  bool
	}{
		{expression: "*", version: "1.2.3"},
		{expression: "[1.0,2.0)", version: "1.5.0"},
		{expression: "[6.7-rc1,6.7)", version: "6.7-rc2", matchMain: true},
		{expression: "(,255-10)", version: "252.4-6.an23/aarch64", kind: 1},
		{expression: "=>1", version: "anything", kind: 2},
		{expression: "\x00||[", version: "\xff", kind: 255},
	}
	for _, seed := range seeds {
		f.Add(seed.expression, seed.version, seed.kind, seed.matchMain)
	}

	f.Fuzz(func(t *testing.T, expression, version string, kind uint8, matchMain bool) {
		matcher, err := compile(expression, CompileOptions{
			Kind:      VersionKind(kind),
			MatchMain: matchMain,
		})
		if err == nil {
			_, _ = matcher.match(version)
		}
	})
}
