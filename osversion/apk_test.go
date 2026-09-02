package osversion

import "testing"

func TestCompareAPK(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		left  string
		right string
		want  int
	}{
		{name: "相同版本", left: "1.2.3-r1", right: "1.2.3-r1", want: 0},
		{name: "首字段前导零按数值", left: "0002", right: "2", want: 0},
		{name: "附加字母", left: "1.2a", right: "1.2", want: 1},
		{name: "字母顺序", left: "1.2b", right: "1.2a", want: 1},
		{name: "Alpha", left: "1.0_alpha", right: "1.0_beta", want: -1},
		{name: "Beta", left: "1.0_beta", right: "1.0_pre", want: -1},
		{name: "Pre", left: "1.0_pre", right: "1.0_rc", want: -1},
		{name: "RC", left: "1.0_rc1", right: "1.0", want: -1},
		{name: "Suffix 数字", left: "1.0_rc10", right: "1.0_rc2", want: 1},
		{name: "VCS 后缀晚于正式版", left: "1.0_git1", right: "1.0", want: 1},
		{name: "Patch 后缀", left: "1.0_p1", right: "1.0_hg1", want: 1},
		{name: "Revision", left: "1.0-r2", right: "1.0-r1", want: 1},
		{name: "缺失 Revision 更早", left: "1.0", right: "1.0-r0", want: -1},
		{name: "Revision 任意精度", left: "1-r100000000000000000000", right: "1-r99999999999999999999", want: 1},
		{name: "点分前导零使用原字符串", left: "1.02", right: "1.010", want: 1},
		{name: "Commit Hash", left: "1.0~deadbeef", right: "1.0~cafebabe", want: 1},
		{name: "Commit 后可跟 Revision", left: "1.0~deadbeef-r2", right: "1.0~deadbeef-r1", want: 1},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := compareAPK(test.left, test.right)
			if err != nil {
				t.Fatalf("compareAPK(%q, %q) error = %v", test.left, test.right, err)
			}
			if got != test.want {
				t.Fatalf("compareAPK(%q, %q) = %d, want %d", test.left, test.right, got, test.want)
			}
		})
	}
}

func TestAPKSuffixOrder(t *testing.T) {
	t.Parallel()

	ordered := []string{
		"1_alpha",
		"1_beta",
		"1_pre",
		"1_rc",
		"1",
		"1_cvs",
		"1_svn",
		"1_git",
		"1_hg",
		"1_p",
	}
	for index := 0; index+1 < len(ordered); index++ {
		got, err := compareAPK(ordered[index], ordered[index+1])
		if err != nil {
			t.Fatalf("compareAPK(%q, %q) error = %v", ordered[index], ordered[index+1], err)
		}
		if got != -1 {
			t.Fatalf("compareAPK(%q, %q) = %d, want -1", ordered[index], ordered[index+1], got)
		}
	}
}

func TestCompareAPKRejectsInvalidVersion(t *testing.T) {
	t.Parallel()

	invalid := []string{
		"",
		"v1.0",
		"1..0",
		"1.0_rcx",
		"1.0~rc1",
		"1.0~DEADBEEF",
		"1.0-r",
		"1.0-1",
		"1:1.0",
		"1.0/1",
	}
	for _, version := range invalid {
		version := version
		t.Run(version, func(t *testing.T) {
			t.Parallel()
			if _, err := compareAPK(version, "1.0"); err == nil {
				t.Fatalf("compareAPK(%q, %q) error = nil, want validation error", version, "1.0")
			}
		})
	}
}
