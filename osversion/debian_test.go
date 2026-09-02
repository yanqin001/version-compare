package osversion

import "testing"

func TestCompareDebian(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		left  string
		right string
		want  int
	}{
		{name: "相同版本", left: "1.2.3-1ubuntu1", right: "1.2.3-1ubuntu1", want: 0},
		{name: "Epoch 优先", left: "2:1.0-1", right: "1:9999-99", want: 1},
		{name: "Epoch 任意精度", left: "100000000000000000000:1", right: "99999999999999999999:9", want: 1},
		{name: "缺失 Revision 等于零", left: "1.0", right: "1.0-0", want: 0},
		{name: "Revision", left: "1.0-1", right: "1.0", want: 1},
		{name: "Ubuntu Revision", left: "1.0-0ubuntu0.22.04.2", right: "1.0-0ubuntu0.22.04.1", want: 1},
		{name: "Tilde 早于正式版", left: "1.0~rc1", right: "1.0", want: -1},
		{name: "重复 Tilde", left: "1.0~~", right: "1.0~", want: -1},
		{name: "字母早于标点", left: "1.0a", right: "1.0+", want: -1},
		{name: "数字字段前导零", left: "1.0000000000000000002", right: "1.2", want: 0},
		{name: "数字字段任意精度", left: "1.100000000000000000000", right: "1.99999999999999999999", want: 1},
		{name: "最后连字符分隔 Revision", left: "1.0-beta-2", right: "1.0-beta-1", want: 1},
		{name: "Upstream 中允许冒号", left: "1:2:4-1", right: "1:2:3-9", want: 1},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := compareDebian(test.left, test.right)
			if err != nil {
				t.Fatalf("compareDebian(%q, %q) error = %v", test.left, test.right, err)
			}
			if got != test.want {
				t.Fatalf("compareDebian(%q, %q) = %d, want %d", test.left, test.right, got, test.want)
			}
		})
	}
}

func TestCompareDebianRejectsInvalidVersion(t *testing.T) {
	t.Parallel()

	invalid := []string{
		"",
		":1.0",
		"x:1.0",
		"release",
		"1.0-",
		"1.0-1_ubuntu",
		"1.0/1",
	}
	for _, version := range invalid {
		version := version
		t.Run(version, func(t *testing.T) {
			t.Parallel()
			if _, err := compareDebian(version, "1.0"); err == nil {
				t.Fatalf("compareDebian(%q, %q) error = nil, want validation error", version, "1.0")
			}
		})
	}
}
