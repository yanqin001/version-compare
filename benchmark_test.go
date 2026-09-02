package versioncompare

import "testing"

var (
	benchmarkMatcherResult bool
	benchmarkCompareResult int
	benchmarkSortResult    []string
	benchmarkCompiled      *Matcher
)

func BenchmarkCompile(b *testing.B) {
	options := CompileOptions{FixedVersions: []FixedVersion{
		{Branch: "3.16", Version: "3.16.35"},
		{Branch: "3.18", Version: "3.18.54"},
		{Branch: "4.1", Version: "4.1.29"},
	}}
	expression := "[3.13-rc1,4.3-rc1)||[5.0.0,5.1.4)||[6.0.0,6.0.2)"

	b.ReportAllocs()
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		matcher, err := Compile(expression, options)
		if err != nil {
			b.Fatal(err)
		}
		benchmarkCompiled = matcher
	}
}

func BenchmarkMatcherMatch(b *testing.B) {
	matcher, err := Compile("[1.0.0,2.0.0)||[3.0.0-rc1,3.1.5)", CompileOptions{})
	if err != nil {
		b.Fatal(err)
	}
	versions := []string{
		"0.9.9",
		"1.0.0",
		"1.8.12-rc1",
		"2.0.0",
		"3.0.0-rc2",
		"3.1.4",
		"@mattermost/types@4.0.0",
	}

	b.ReportAllocs()
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		matched, matchErr := matcher.Match(versions[index%len(versions)])
		if matchErr != nil {
			b.Fatal(matchErr)
		}
		benchmarkMatcherResult = matched
	}
}

func BenchmarkCompare(b *testing.B) {
	svc := NewVersionSvc()
	pairs := [][2]string{
		{"1.0.9", "1.0.10"},
		{"1.1.1a", "1.1.1t"},
		{"6.7-rc10", "6.7-rc2"},
		{"@mattermost/types@9.0.0", "8.1.3"},
		{"21", "21.0.0"},
	}

	b.ReportAllocs()
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		pair := pairs[index%len(pairs)]
		result, err := svc.Compare(pair[0], pair[1])
		if err != nil {
			b.Fatal(err)
		}
		benchmarkCompareResult = result
	}
}

func BenchmarkSort(b *testing.B) {
	svc := NewVersionSvc()
	source := []string{
		"1.0.6", "1.0.1", "1.0.2", "1.0.3", "1.0.4", "1.0.5",
		"1.0.7", "1.0.8", "1.0.9", "1.0.10", "1.0.11", "1.3",
		"1.0.0", "2.0-rc2", "2.0-rc1", "2.0", "1.1.1a", "1.1.1t",
		"3.0.0", "2.9.9", "4.0.1", "4.0.0", "3.1.5", "3.1.4",
	}
	working := make([]string, len(source))

	b.ReportAllocs()
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		copy(working, source)
		sorted, err := svc.Sort(working)
		if err != nil {
			b.Fatal(err)
		}
		benchmarkSortResult = sorted
	}
}
