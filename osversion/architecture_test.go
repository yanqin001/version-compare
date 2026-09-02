package osversion

import (
	"errors"
	"sync"
	"testing"
)

func TestCompileArchitecturesUnrestricted(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		allowed []string
	}{
		{name: "nil list", allowed: nil},
		{name: "empty list", allowed: []string{}},
		{name: "asterisk", allowed: []string{"*"}},
		{name: "any", allowed: []string{" ANY "}},
		{name: "multiple wildcards", allowed: []string{"*", "any"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			matcher, err := CompileArchitectures(test.allowed)
			if err != nil {
				t.Fatalf("CompileArchitectures() error = %v", err)
			}
			for _, actual := range []string{"", "amd64", "noarch", "src", "unknown"} {
				if !matcher.Match(actual) {
					t.Errorf("Match(%q) = false, want true", actual)
				}
			}
		})
	}
}

func TestCompileArchitecturesRejectsInvalidConstraints(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		allowed []string
	}{
		{name: "empty item", allowed: []string{""}},
		{name: "blank item", allowed: []string{"   "}},
		{name: "slash only", allowed: []string{"///"}},
		{name: "asterisk with concrete", allowed: []string{"*", "amd64"}},
		{name: "any with concrete", allowed: []string{"arm64", "any"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			matcher, err := CompileArchitectures(test.allowed)
			if matcher != nil {
				t.Errorf("CompileArchitectures() matcher = %#v, want nil", matcher)
			}
			if !errors.Is(err, ErrInvalidArchitecture) {
				t.Errorf("CompileArchitectures() error = %v, want ErrInvalidArchitecture", err)
			}
		})
	}
}

func TestArchitectureMatcherAliasesAndNormalization(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		allowed []string
		actual  string
		want    bool
	}{
		{name: "x86 alias", allowed: []string{"x86_64"}, actual: "AMD64", want: true},
		{name: "x86 leading slash", allowed: []string{"amd64"}, actual: " /x86_64 ", want: true},
		{name: "arm alias", allowed: []string{"aarch64"}, actual: "arm64", want: true},
		{name: "power alias", allowed: []string{"ppc64el"}, actual: "/PPC64LE", want: true},
		{name: "exact other architecture", allowed: []string{"s390x"}, actual: " S390X ", want: true},
		{name: "different architecture", allowed: []string{"amd64"}, actual: "arm64", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			matcher, err := CompileArchitectures(test.allowed)
			if err != nil {
				t.Fatalf("CompileArchitectures() error = %v", err)
			}
			if got := matcher.Match(test.actual); got != test.want {
				t.Errorf("Match(%q) = %v, want %v", test.actual, got, test.want)
			}
		})
	}
}

func TestArchitectureMatcherNoarchIsNotWildcard(t *testing.T) {
	t.Parallel()

	matcher, err := CompileArchitectures([]string{"all"})
	if err != nil {
		t.Fatalf("CompileArchitectures() error = %v", err)
	}
	if !matcher.Match("noarch") {
		t.Error("Match(noarch) = false, want true")
	}
	if matcher.Match("amd64") {
		t.Error("Match(amd64) = true, want false")
	}

	matcher, err = CompileArchitectures([]string{"amd64"})
	if err != nil {
		t.Fatalf("CompileArchitectures() error = %v", err)
	}
	if matcher.Match("all") {
		t.Error("Match(all) = true, want false")
	}
}

func TestArchitectureMatcherSourceCompatibility(t *testing.T) {
	t.Parallel()

	matcher, err := CompileArchitectures([]string{"source"})
	if err != nil {
		t.Fatalf("CompileArchitectures() error = %v", err)
	}
	for _, actual := range []string{"src", "source", "", "   ", "/"} {
		if !matcher.Match(actual) {
			t.Errorf("Match(%q) = false, want true", actual)
		}
	}
	if matcher.Match("amd64") {
		t.Error("Match(amd64) = true, want false")
	}

	matcher, err = CompileArchitectures([]string{"amd64"})
	if err != nil {
		t.Fatalf("CompileArchitectures() error = %v", err)
	}
	if matcher.Match("") {
		t.Error("Match(empty) = true, want false")
	}
}

func TestCompileArchitecturesCopiesInput(t *testing.T) {
	t.Parallel()

	allowed := []string{"amd64"}
	matcher, err := CompileArchitectures(allowed)
	if err != nil {
		t.Fatalf("CompileArchitectures() error = %v", err)
	}
	allowed[0] = "arm64"

	if !matcher.Match("amd64") {
		t.Error("Match(amd64) = false after input mutation, want true")
	}
	if matcher.Match("arm64") {
		t.Error("Match(arm64) = true after input mutation, want false")
	}
}

func TestArchitectureMatcherConcurrentUse(t *testing.T) {
	matcher, err := CompileArchitectures([]string{"amd64", "arm64", "src"})
	if err != nil {
		t.Fatalf("CompileArchitectures() error = %v", err)
	}

	const goroutines = 32
	var waitGroup sync.WaitGroup
	waitGroup.Add(goroutines)
	for range goroutines {
		go func() {
			defer waitGroup.Done()
			for range 100 {
				if !matcher.Match("x86_64") || !matcher.Match("aarch64") || !matcher.Match("") {
					t.Error("concurrent Match() returned an unexpected result")
					return
				}
			}
		}()
	}
	waitGroup.Wait()
}

func TestArchitectureMatcherNilReceiver(t *testing.T) {
	t.Parallel()

	var matcher *ArchitectureMatcher
	if matcher.Match("amd64") {
		t.Error("nil matcher Match() = true, want false")
	}
}

func TestMatchArchitecture(t *testing.T) {
	t.Parallel()

	matched, err := MatchArchitecture("/AARCH64", []string{"arm64"})
	if err != nil {
		t.Fatalf("MatchArchitecture() error = %v", err)
	}
	if !matched {
		t.Error("MatchArchitecture() = false, want true")
	}

	matched, err = MatchArchitecture("amd64", []string{"*", "arm64"})
	if matched {
		t.Error("MatchArchitecture() = true for invalid constraints, want false")
	}
	if !errors.Is(err, ErrInvalidArchitecture) {
		t.Errorf("MatchArchitecture() error = %v, want ErrInvalidArchitecture", err)
	}
}
