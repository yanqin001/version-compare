package osversion

import "testing"

func TestCompareALPM(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		left  string
		right string
		want  int
	}{
		{name: "相同版本", left: "1.2.3-1", right: "1.2.3-1", want: 0},
		{name: "Epoch 优先", left: "2:1.0-1", right: "1:9999-99", want: 1},
		{name: "Epoch 任意精度", left: "100000000000000000000:1", right: "99999999999999999999:9", want: 1},
		{name: "单侧 pkgrel 不比较", left: "1.5-1", right: "1.5", want: 0},
		{name: "两侧 pkgrel", left: "1.5-2", right: "1.5-1", want: 1},
		{name: "点分 pkgrel", left: "1.5-1.2", right: "1.5-1.1", want: 1},
		{name: "前导零", left: "1.0002", right: "1.2", want: 0},
		{name: "任意长度数字", left: "1.100000000000000000000", right: "1.99999999999999999999", want: 1},
		{name: "数字段高于字母段", left: "1.1", right: "1.a", want: 1},
		{name: "附着字母早于正式版", left: "1.0rc", right: "1.0", want: -1},
		{name: "分隔字母晚于正式版", left: "1.0.rc", right: "1.0", want: 1},
		{name: "额外数字段", left: "1.0.1", right: "1.0", want: 1},
		{name: "同数量分隔符等价", left: "1+0", right: "1.0", want: 0},
		{name: "连续分隔符数量", left: "1...0", right: "1.2", want: 1},
		{name: "Tilde 只是分隔符", left: "1.0~rc1", right: "1.0", want: 1},
		{name: "大小写敏感", left: "1.0a", right: "1.0A", want: 1},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := compareALPM(test.left, test.right)
			if err != nil {
				t.Fatalf("compareALPM(%q, %q) error = %v", test.left, test.right, err)
			}
			if got != test.want {
				t.Fatalf("compareALPM(%q, %q) = %d, want %d", test.left, test.right, got, test.want)
			}
		})
	}
}

func TestALPMDocumentedOrder(t *testing.T) {
	t.Parallel()

	ordered := []string{
		"1.0a",
		"1.0b",
		"1.0beta",
		"1.0p",
		"1.0pre",
		"1.0rc",
		"1.0",
		"1.0.a",
		"1.0.1",
	}
	for index := 0; index+1 < len(ordered); index++ {
		got, err := compareALPM(ordered[index], ordered[index+1])
		if err != nil {
			t.Fatalf("compareALPM(%q, %q) error = %v", ordered[index], ordered[index+1], err)
		}
		if got != -1 {
			t.Fatalf("compareALPM(%q, %q) = %d, want -1", ordered[index], ordered[index+1], got)
		}
	}
}

func TestCompareALPMRejectsInvalidVersion(t *testing.T) {
	t.Parallel()

	invalid := []string{
		"",
		":1.0",
		"x:1.0",
		"1:2:3",
		"1.0-",
		"1.0-rc1",
		"1.0-1-2",
		"1.0/1",
		"1.0 1",
		"1.0<2",
		"1.0-1.2.3",
	}
	for _, version := range invalid {
		version := version
		t.Run(version, func(t *testing.T) {
			t.Parallel()
			if _, err := compareALPM(version, "1.0"); err == nil {
				t.Fatalf("compareALPM(%q, %q) error = nil, want validation error", version, "1.0")
			}
		})
	}
}

func TestALPMDelimiterOnlyVersion(t *testing.T) {
	t.Parallel()

	got, err := compareALPM(".", "...")
	if err != nil {
		t.Fatalf("compareALPM() error = %v", err)
	}
	if got != 0 {
		t.Fatalf("compareALPM(%q, %q) = %d, want 0", ".", "...", got)
	}
}
