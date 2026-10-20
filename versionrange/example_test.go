package versionrange_test

import (
	"fmt"

	"github.com/yanqin001/version-compare/versionrange"
)

// ExampleNewComparator 展示按 PEP 440 比较预发布版本，结果使用标准 -1/0/1。
func ExampleNewComparator() {
	comparator, err := versionrange.NewComparator(versionrange.Classification{
		RangeType: versionrange.TypeVersion, VersionScheme: "pep440",
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	order, err := comparator.Compare("3.15.0a1", "3.15.0rc3")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(order)
	// Output: -1
}

// ExampleClassification_ValidateRange 展示特殊标签可作为结构化边界，不受表达式括号语法限制。
func ExampleClassification_ValidateRange() {
	classification := versionrange.Classification{
		RangeType: versionrange.TypeVersion, VersionScheme: "semver", VersionSuffix: "(jdk8/11)",
	}
	fmt.Println(classification.ValidateRange("v2.6.0(jdk8/11)", "v2.7.0(jdk8/11)", false))
	// Output: <nil>
}
