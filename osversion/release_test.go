package osversion

import "testing"

// TestSplitEmbeddedMetadataServicePack 验证带 SP 后缀及 Kylin 标记的发行版信息可从旧版本串提取。
func TestSplitEmbeddedMetadataServicePack(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input        string
		distribution string
		wantRelease  string
		wantArch     string
	}{
		{input: "1.1.1f-9.oe2203sp1/x86_64", distribution: "openeuler", wantRelease: "2203sp1", wantArch: "x86_64"},
		{input: "1.1.1f-9.oe2203sp164kb/aarch64", distribution: "openeuler", wantRelease: "2203sp1", wantArch: "aarch64"},
		{input: "5.10-2.ky10sp1/aarch64", distribution: "kylin", wantRelease: "10", wantArch: "aarch64"},
		{input: "5.10-2.an8plus/x86_64", distribution: "anolis", wantRelease: "8", wantArch: "x86_64"},
	}
	for _, test := range tests {
		_, release, architecture := splitEmbeddedMetadata(test.input)
		release = normalizeOSRelease(test.distribution, release)
		if release != test.wantRelease || architecture != test.wantArch {
			t.Fatalf("normalized metadata %q = release %q, arch %q", test.input, release, architecture)
		}
	}
}

// TestNormalizeOSRelease 验证常见发行版展示值会规范化为可比较的发行版本。
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

// TestDistributionAliasesMatch 验证发行版名称、简称和算法族别名保持兼容。
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
