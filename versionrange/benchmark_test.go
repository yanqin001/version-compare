package versionrange

import "testing"

// BenchmarkComparatorCompare 记录各策略复用比较器时的耗时与分配，供后续优化对照。
func BenchmarkComparatorCompare(b *testing.B) {
	for _, test := range []struct{ scheme, left, right string }{
		{"numeric", "1.2.10", "1.2.9"},
		{"pep440", "3.15.0a1", "3.15.0rc3"},
		{"maven", "1.0.CR3", "1.0.Final"},
		{"rpm", "1.0~rc1-1", "1.0-1"},
	} {
		b.Run(test.scheme, func(b *testing.B) {
			c, err := NewComparator(Classification{RangeType: TypeVersion, VersionScheme: test.scheme})
			if err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := c.Compare(test.left, test.right); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
