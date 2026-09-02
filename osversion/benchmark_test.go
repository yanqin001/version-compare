package osversion

import "testing"

var (
	benchmarkOSMatcherResult bool
	benchmarkOSMatcher       *Matcher
)

// BenchmarkOSCompile 测量联合 OS 约束的编译开销。
func BenchmarkOSCompile(b *testing.B) {
	constraint := Constraint{
		Distribution:        DistributionUbuntu,
		VersionExpression:   "[1:2.31.1-0ubuntu9.1,1:2.31.1-0ubuntu9.17)||[1:2.35-0ubuntu3,1:2.35-0ubuntu3.10)",
		OSVersionExpression: "[20.04,24.04]",
		Architectures:       []string{"amd64", "arm64"},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		matcher, err := Compile(constraint)
		if err != nil {
			b.Fatal(err)
		}
		benchmarkOSMatcher = matcher
	}
}

// BenchmarkOSMatcherMatch 测量复用联合 OS Matcher 的单次匹配开销。
func BenchmarkOSMatcherMatch(b *testing.B) {
	matcher, err := Compile(Constraint{
		Distribution:        DistributionUbuntu,
		VersionExpression:   "[1:2.31.1-0ubuntu9.1,1:2.31.1-0ubuntu9.17)||[1:2.35-0ubuntu3,1:2.35-0ubuntu3.10)",
		OSVersionExpression: "[20.04,24.04]",
		Architectures:       []string{"amd64", "arm64"},
	})
	if err != nil {
		b.Fatal(err)
	}
	packages := []Package{
		{Distribution: DistributionUbuntu, Version: "1:2.31.1-0ubuntu9.16", OSVersion: "22.04", Architecture: "amd64"},
		{Distribution: DistributionUbuntu, Version: "1:2.31.1-0ubuntu9.17", OSVersion: "22.04", Architecture: "arm64"},
		{Distribution: DistributionUbuntu, Version: "1:2.35-0ubuntu3.9", OSVersion: "24.04", Architecture: "x86_64"},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		matched, matchErr := matcher.Match(packages[index%len(packages)])
		if matchErr != nil {
			b.Fatal(matchErr)
		}
		benchmarkOSMatcherResult = matched
	}
}
