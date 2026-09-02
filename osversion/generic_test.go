package osversion

import "testing"

func TestCompareGeneric(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		left  string
		right string
		want  int
	}{
		{name: "Anolis 版本和架构", left: "252.4-6.an23/aarch64", right: "255-10", want: -1},
		{name: "尾零等价", left: "7.1", right: "7.1.0.0", want: 0},
		{name: "点分数字", left: "7.0.3.1", right: "7.0.3", want: 1},
		{name: "正式版高于后缀", left: "1.0", right: "1.0-rc1", want: 1},
		{name: "后缀数字自然排序", left: "1.0-rc10", right: "1.0-rc2", want: 1},
		{name: "忽略架构", left: "1.2.3/x86_64", right: "1.2.3/aarch64", want: 0},
		{name: "忽略大小写", left: "1.0-RC1", right: "1.0-rc1", want: 0},
		{name: "忽略 Epoch", left: "1:2", right: "1", want: 1},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := compareGeneric(test.left, test.right)
			if err != nil {
				t.Fatalf("compareGeneric(%q, %q) error = %v", test.left, test.right, err)
			}
			if got != test.want {
				t.Fatalf("compareGeneric(%q, %q) = %d, want %d", test.left, test.right, got, test.want)
			}
		})
	}
}

func TestCompareGenericRejectsEmptyVersion(t *testing.T) {
	t.Parallel()

	if _, err := compareGeneric("", "1.0"); err == nil {
		t.Fatal("compareGeneric() error = nil, want validation error")
	}
}
