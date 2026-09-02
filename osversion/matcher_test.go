package osversion

import (
	"errors"
	"testing"
)

func TestMatcherAnolisCompatibility(t *testing.T) {
	t.Parallel()

	matcher, err := Compile(Constraint{
		Distribution:        DistributionAnolis,
		VersionExpression:   "(,255-10)",
		OSVersionExpression: "an23",
		Architectures:       []string{"aarch64"},
	})
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		pkg     Package
		matched bool
	}{
		{
			name: "旧格式自动提取 OS 和架构",
			pkg: Package{
				Distribution: DistributionAnolis,
				Version:      "252.4-6.an23/aarch64",
			},
			matched: true,
		},
		{
			name: "达到修复版本",
			pkg: Package{
				Distribution: DistributionAnolis,
				Version:      "255-10.an23/aarch64",
			},
			matched: false,
		},
		{
			name: "OS 版本不同",
			pkg: Package{
				Distribution: DistributionAnolis,
				Version:      "252.4-6.an8/aarch64",
			},
			matched: false,
		},
		{
			name: "架构不同",
			pkg: Package{
				Distribution: DistributionAnolis,
				Version:      "252.4-6.an23/x86_64",
			},
			matched: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, matchErr := matcher.Match(test.pkg)
			if matchErr != nil {
				t.Fatalf("Match() error = %v", matchErr)
			}
			if got != test.matched {
				t.Fatalf("Match() = %v, want %v", got, test.matched)
			}
		})
	}
}

func TestMatcherUbuntuNativeVersion(t *testing.T) {
	t.Parallel()

	matcher, err := Compile(Constraint{
		Distribution:        DistributionUbuntu,
		VersionExpression:   "[1:2.31.1-0ubuntu9.1,1:2.31.1-0ubuntu9.17)",
		OSVersionExpression: "[22.04,22.04]",
		Architectures:       []string{"amd64"},
	})
	if err != nil {
		t.Fatal(err)
	}
	matched, err := matcher.Match(Package{
		Distribution: DistributionUbuntu,
		Version:      "1:2.31.1-0ubuntu9.16",
		OSVersion:    "Ubuntu 22.04 LTS",
		Architecture: "x86_64",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !matched {
		t.Fatal("Match() = false, want true")
	}
}

func TestMatcherDistributionMismatch(t *testing.T) {
	t.Parallel()

	matcher, err := Compile(Constraint{Distribution: DistributionUbuntu, VersionExpression: "*"})
	if err != nil {
		t.Fatal(err)
	}
	matched, err := matcher.Match(Package{Distribution: DistributionDebian, Version: "1.0"})
	if err != nil {
		t.Fatal(err)
	}
	if matched {
		t.Fatal("Match() = true for a different distribution")
	}
}

func TestMatcherGenericFallback(t *testing.T) {
	t.Parallel()

	matcher, err := Compile(Constraint{
		Distribution:      "custom-os",
		VersionExpression: "(,252-10)",
	})
	if err != nil {
		t.Fatal(err)
	}
	matched, err := matcher.Match(Package{Distribution: "custom-os", Version: "252.4-6/custom"})
	if err != nil {
		t.Fatal(err)
	}
	if matched {
		t.Fatal("Match() = true, want false")
	}
}

func TestCompileRejectsInvalidOneSidedNativeBound(t *testing.T) {
	t.Parallel()

	_, err := Compile(Constraint{
		Distribution:      DistributionUbuntu,
		VersionExpression: "<invalid",
	})
	if !errors.Is(err, ErrInvalidVersion) {
		t.Fatalf("Compile() error = %v, want ErrInvalidVersion", err)
	}
}

func TestNativeWildcardStillValidatesCandidate(t *testing.T) {
	t.Parallel()

	matcher, err := Compile(Constraint{
		Distribution:      DistributionRPM,
		VersionExpression: "*",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = matcher.Match(Package{Version: "1.0-1-extra"}); !errors.Is(err, ErrInvalidVersion) {
		t.Fatalf("Match() error = %v, want ErrInvalidVersion", err)
	}
}

func TestGenericExactOpaqueVersion(t *testing.T) {
	t.Parallel()

	matcher, err := Compile(Constraint{
		Distribution:      DistributionGeneric,
		VersionExpression: "master",
	})
	if err != nil {
		t.Fatal(err)
	}
	for version, want := range map[string]bool{"master": true, "main": false} {
		matched, matchErr := matcher.Match(Package{Version: version})
		if matchErr != nil {
			t.Fatalf("Match(%q) error = %v", version, matchErr)
		}
		if matched != want {
			t.Fatalf("Match(%q) = %v, want %v", version, matched, want)
		}
	}
}
