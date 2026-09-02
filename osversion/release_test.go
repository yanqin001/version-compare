package osversion

import "testing"

func TestNormalizeOSRelease(t *testing.T) {
	t.Parallel()

	tests := []struct {
		distribution Distribution
		version      string
		want         string
	}{
		{distribution: DistributionAnolis, version: "Anolis OS 23", want: "23"},
		{distribution: DistributionUbuntu, version: "Ubuntu 22.04 LTS", want: "22.04"},
		{distribution: DistributionRHEL, version: "Red Hat Enterprise Linux 9.4", want: "9.4"},
		{distribution: DistributionOpenSUSE, version: "openSUSE Leap 15.6", want: "15.6"},
		{distribution: DistributionAlibabaCloudLinux, version: "Alibaba Cloud Linux 3", want: "3"},
		{distribution: DistributionAlpine, version: "Alpine Linux v3.20", want: "3.20"},
		{distribution: DistributionManjaro, version: "Manjaro Linux 24.0", want: "24.0"},
	}
	for _, test := range tests {
		test := test
		t.Run(string(test.distribution), func(t *testing.T) {
			t.Parallel()
			if got := normalizeOSRelease(string(test.distribution), test.version); got != test.want {
				t.Fatalf("normalizeOSRelease(%q, %q) = %q, want %q", test.distribution, test.version, got, test.want)
			}
		})
	}
}

func TestDistributionAliasesMatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		candidate  Distribution
		constraint Distribution
	}{
		{candidate: "ol", constraint: DistributionOracleLinux},
		{candidate: "amzn", constraint: DistributionAmazonLinux},
		{candidate: "alibabacloudlinux", constraint: DistributionAlibabaCloudLinux},
		{candidate: "mint", constraint: DistributionLinuxMint},
		{candidate: DistributionEulerOS, constraint: DistributionOpenEuler},
		{candidate: DistributionOpenSUSE, constraint: DistributionSUSE},
		{candidate: DistributionUbuntu, constraint: "os:ubuntu"},
		{candidate: DistributionRHEL, constraint: "os_rpm"},
		{candidate: DistributionUbuntu, constraint: DistributionDPKG},
		{candidate: DistributionAlpine, constraint: DistributionAPK},
		{candidate: DistributionManjaro, constraint: DistributionALPM},
	}
	for _, test := range tests {
		if !distributionMatches(test.candidate, test.constraint) {
			t.Errorf("distributionMatches(%q, %q) = false, want true", test.candidate, test.constraint)
		}
	}
}
