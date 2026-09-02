package versioncompare

import (
	"errors"
	"testing"
)

func TestParseLegacyFixed(t *testing.T) {
	t.Parallel()

	got, err := parseLegacyFixed([]string{"4.14: 4.14.272", "4.19.235"}, NormalVersion)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Branch != "4.14" || got[0].Version != "4.14.272" || got[1].Branch != "4.19" {
		t.Fatalf("parseLegacyFixed() = %#v", got)
	}
}

func TestCompileFixedVersionRejectsDifferentBranch(t *testing.T) {
	t.Parallel()

	_, err := Compile("[4.0,5.0)", CompileOptions{FixedVersions: []FixedVersion{
		{Branch: "4.14", Version: "4.15.1"},
	}})
	if !errors.Is(err, ErrInvalidExpression) {
		t.Fatalf("Compile() error = %v, want ErrInvalidExpression", err)
	}
}
