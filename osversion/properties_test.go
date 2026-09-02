package osversion

import "testing"

func TestNativeComparatorsOrderingProperties(t *testing.T) {
	t.Parallel()

	versionsByDistribution := map[string][]string{
		"rpm": {
			"0:1.0~rc1-1", "1.0-1", "1.0-2", "1.0^git1-1", "1.1-0", "2:0.1-1",
		},
		"debian": {
			"0:1.0~~-1", "1.0~rc1-1", "1.0-0", "1.0-1", "1.0-1ubuntu1", "1:0.1-1",
		},
		"alpine": {
			"1.0_alpha", "1.0_beta", "1.0_rc1", "1.0", "1.0_p1", "1.0-r1",
		},
		"arch": {
			"1.0alpha-1", "1.0-1", "1.0.1-1", "1.1-1", "2:0.1-1",
		},
		"os": {
			"1.0-rc1", "1.0", "1.0.1", "2.0", "10.0",
		},
	}

	for distribution, versions := range versionsByDistribution {
		distribution := distribution
		versions := versions
		t.Run(distribution, func(t *testing.T) {
			t.Parallel()
			assertOrderingProperties(t, distribution, versions)
		})
	}
}

func assertOrderingProperties(t *testing.T, distribution string, versions []string) {
	t.Helper()

	for _, left := range versions {
		self, err := compareNative(distribution, left, left)
		if err != nil {
			t.Fatalf("compareNative(%q, %q, %q) error = %v", distribution, left, left, err)
		}
		if self != 0 {
			t.Fatalf("compareNative(%q, %q, %q) = %d, want reflexive equality", distribution, left, left, self)
		}

		for _, right := range versions {
			forward, err := compareNative(distribution, left, right)
			if err != nil {
				t.Fatalf("compareNative(%q, %q, %q) error = %v", distribution, left, right, err)
			}
			reverse, err := compareNative(distribution, right, left)
			if err != nil {
				t.Fatalf("compareNative(%q, %q, %q) error = %v", distribution, right, left, err)
			}
			if forward != -reverse {
				t.Fatalf("comparison is not antisymmetric for %q and %q: %d, %d", left, right, forward, reverse)
			}

			for _, third := range versions {
				middleToThird, err := compareNative(distribution, right, third)
				if err != nil {
					t.Fatalf("compareNative(%q, %q, %q) error = %v", distribution, right, third, err)
				}
				if forward <= 0 && middleToThird <= 0 {
					leftToThird, err := compareNative(distribution, left, third)
					if err != nil {
						t.Fatalf("compareNative(%q, %q, %q) error = %v", distribution, left, third, err)
					}
					if leftToThird > 0 {
						t.Fatalf("comparison is not transitive: %q <= %q <= %q, but result is %d", left, right, third, leftToThird)
					}
				}
			}
		}
	}
}
