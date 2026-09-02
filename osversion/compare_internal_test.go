package osversion

import "testing"

func TestResolveAlgorithm(t *testing.T) {
	t.Parallel()

	tests := []struct {
		distribution string
		want         algorithm
	}{
		{distribution: "", want: algorithmGeneric},
		{distribution: "os", want: algorithmGeneric},
		{distribution: "unknown-linux", want: algorithmGeneric},
		{distribution: "RHEL", want: algorithmRPM},
		{distribution: "CentOS Stream", want: algorithmRPM},
		{distribution: "Anolis_OS", want: algorithmRPM},
		{distribution: "openEuler", want: algorithmRPM},
		{distribution: "openSUSE-Tumbleweed", want: algorithmRPM},
		{distribution: "os:rhel", want: algorithmRPM},
		{distribution: "Ubuntu", want: algorithmDebian},
		{distribution: "os_debian", want: algorithmDebian},
		{distribution: "Alpine Linux", want: algorithmAPK},
		{distribution: "Wolfi", want: algorithmAPK},
		{distribution: "Arch Linux", want: algorithmALPM},
		{distribution: "pacman", want: algorithmALPM},
	}

	for _, test := range tests {
		test := test
		t.Run(test.distribution, func(t *testing.T) {
			t.Parallel()
			if got := resolveAlgorithm(test.distribution); got != test.want {
				t.Fatalf("resolveAlgorithm(%q) = %d, want %d", test.distribution, got, test.want)
			}
		})
	}
}

func TestCompareNativeDispatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		distribution string
		left         string
		right        string
		want         int
	}{
		{name: "RPM", distribution: "sles", left: "1.0~rc1", right: "1.0", want: -1},
		{name: "Debian", distribution: "ubuntu", left: "1.0~rc1", right: "1.0", want: -1},
		{name: "APK", distribution: "alpine", left: "1.0_rc1", right: "1.0", want: -1},
		{name: "ALPM", distribution: "arch", left: "1.0-1", right: "1.0", want: 0},
		{name: "generic", distribution: "os", left: "252.4-6.an23/aarch64", right: "255-10", want: -1},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := compareNative(test.distribution, test.left, test.right)
			if err != nil {
				t.Fatalf("compareNative() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("compareNative() = %d, want %d", got, test.want)
			}
		})
	}
}

func TestCompareNumericStrings(t *testing.T) {
	t.Parallel()

	left := "100000000000000000000000000000000000000000000000000"
	right := "99999999999999999999999999999999999999999999999999"
	if got := compareNumericStrings(left, right); got != 1 {
		t.Fatalf("compareNumericStrings() = %d, want 1", got)
	}
	if got := compareNumericStrings("000", "0"); got != 0 {
		t.Fatalf("compareNumericStrings() = %d, want 0", got)
	}
}
