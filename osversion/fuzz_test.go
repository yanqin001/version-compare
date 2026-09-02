package osversion

import "testing"

// FuzzNativeComparisonDoesNotPanic 验证各发行版解析器面对任意输入时不会 panic。
func FuzzNativeComparisonDoesNotPanic(f *testing.F) {
	seeds := []struct {
		distribution string
		left         string
		right        string
	}{
		{distribution: "rpm", left: "1:2.0~rc1-1.el9", right: "1:2.0-1.el9"},
		{distribution: "ubuntu", left: "1:2.0~rc1-1", right: "1:2.0-1"},
		{distribution: "alpine", left: "1.0_rc1-r0", right: "1.0-r0"},
		{distribution: "arch", left: "2:1.0rc1-1", right: "2:1.0-1"},
		{distribution: "custom", left: "252.4-6.an23/aarch64", right: "255-10"},
		{distribution: "rpm", left: "\x00\xff", right: ""},
	}
	for _, seed := range seeds {
		f.Add(seed.distribution, seed.left, seed.right)
	}

	f.Fuzz(func(t *testing.T, distribution, left, right string) {
		_, _ = compareNative(distribution, left, right)
	})
}
