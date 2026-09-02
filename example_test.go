package versioncompare_test

import (
	"fmt"

	versioncompare "github.com/yanqin001/version-compare"
	"github.com/yanqin001/version-compare/osversion"
)

// ExampleVersionSvc 演示兼容接口的比较结果和范围匹配。
func ExampleVersionSvc() {
	service := versioncompare.NewVersionSvc()

	comparison, _ := service.Compare("6.7-rc1", "6.7")
	affected, _ := service.InExp(versioncompare.VersionExp{
		Version:    "1.1.1a",
		Expression: "[1.1.1,1.1.1t)",
	})

	fmt.Println(comparison == int(versioncompare.CompareLess))
	fmt.Println(affected)
	// Output:
	// true
	// true
}

// ExampleCompile 演示编译并复用普通组件版本范围。
func ExampleCompile() {
	matcher, _ := versioncompare.Compile(">=1.1.1, <1.1.1t", versioncompare.CompileOptions{})

	affected, _ := matcher.Match("1.1.1a")
	fixed, _ := matcher.Match("1.1.1t")

	fmt.Println(affected)
	fmt.Println(fixed)
	// Output:
	// true
	// false
}

// Example_osPackageMatcher 演示 Anolis 软件包版本、OS 版本和架构的联合匹配。
func Example_osPackageMatcher() {
	matcher, _ := osversion.Compile(osversion.Constraint{
		Distribution:        osversion.DistributionAnolis,
		VersionExpression:   "(,255-10)",
		OSVersionExpression: "[23,24)",
		Architectures:       []string{"aarch64"},
	})

	affected, _ := matcher.Match(osversion.Package{
		Distribution: osversion.DistributionAnolis,
		Version:      "252.4-6.an23/aarch64",
	})

	fmt.Println(affected)
	// Output:
	// true
}
