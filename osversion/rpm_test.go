package osversion

import "testing"

func TestCompareRPM(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		left  string
		right string
		want  int
	}{
		{name: "相同版本", left: "1.0-1.el8", right: "1.0-1.el8", want: 0},
		{name: "Epoch 优先", left: "1:1.0-1", right: "0:9999-99", want: 1},
		{name: "Epoch 前导零", left: "0001:1", right: "1:1", want: 0},
		{name: "Release", left: "1.0-2.el8", right: "1.0-1.el8", want: 1},
		{name: "缺失 Release 更早", left: "1.0", right: "1.0-1", want: -1},
		{name: "数字段任意精度", left: "1.999999999999999999999999999999", right: "1.100000000000000000000000000000", want: 1},
		{name: "数字段前导零", left: "10.0001", right: "10.1", want: 0},
		{name: "数字段高于字母段", left: "1.0", right: "1.fc40", want: 1},
		{name: "额外字段更新", left: "1.0.0", right: "1.0", want: 1},
		{name: "普通分隔符等价", left: "1+0_2", right: "1.0.2", want: 0},
		{name: "Tilde 预发布", left: "2.0~beta", right: "2.0", want: -1},
		{name: "连续 Tilde", left: "1.0~~rc1", right: "1.0~rc1", want: -1},
		{name: "Caret 晚于正式版", left: "2.0^20260901", right: "2.0", want: 1},
		{name: "Caret 早于下一字段", left: "2.0^20260901", right: "2.0.1", want: -1},
		{name: "大小写敏感", left: "1.0a", right: "1.0A", want: 1},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := compareRPM(test.left, test.right)
			if err != nil {
				t.Fatalf("compareRPM(%q, %q) error = %v", test.left, test.right, err)
			}
			if got != test.want {
				t.Fatalf("compareRPM(%q, %q) = %d, want %d", test.left, test.right, got, test.want)
			}
		})
	}
}

func TestCompareRPMRejectsInvalidVersion(t *testing.T) {
	t.Parallel()

	invalid := []string{
		"",
		":1.0",
		"x:1.0",
		"1.0-",
		"1.0-1-extra",
		"1.0/1",
		"1.版本",
	}
	for _, version := range invalid {
		version := version
		t.Run(version, func(t *testing.T) {
			t.Parallel()
			if _, err := compareRPM(version, "1.0"); err == nil {
				t.Fatalf("compareRPM(%q, %q) error = nil, want validation error", version, "1.0")
			}
		})
	}
}
