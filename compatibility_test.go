package versioncompare

import (
	"errors"
	"testing"
)

func TestLegacyRangeCompatibility(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		exp  VersionExp
		want bool
	}{
		{
			name: "普通无界区间",
			exp:  VersionExp{Version: "0.9.1", Expression: "(,0.9.5]"},
			want: true,
		},
		{
			name: "普通版本超出区间",
			exp:  VersionExp{Version: "1.3.1", Expression: "(,1.3]"},
			want: false,
		},
		{
			name: "RC 下界",
			exp:  VersionExp{Version: "6.7-rc1", Expression: "[6.7-rc1,6.7)"},
			want: true,
		},
		{
			name: "RC 自然序",
			exp:  VersionExp{Version: "6.7-rc10", Expression: "[6.7-rc2,6.7)"},
			want: true,
		},
		{
			name: "RC 窗口排除正式版",
			exp:  VersionExp{Version: "6.7", Expression: "[6.7-rc1,6.7)"},
			want: false,
		},
		{
			name: "RC 窗口排除下一补丁版",
			exp:  VersionExp{Version: "6.7.1-rc1", Expression: "[6.7-rc1,6.7)"},
			want: false,
		},
		{
			name: "显式 MatchMain",
			exp: VersionExp{
				Version:    "6.7-rc2",
				Expression: "[6.7-rc1]",
				MatchMain:  true,
			},
			want: true,
		},
		{
			name: "严格匹配预发布",
			exp: VersionExp{
				Version:    "6.7-rc2",
				Expression: "[6.7-rc1]",
			},
			want: false,
		},
		{
			name: "OpenSSL 单字母补丁",
			exp: VersionExp{
				Version:    "1.1.1a",
				Expression: "[1.1.1,1.1.1t)",
				Type:       "openssl",
			},
			want: true,
		},
		{
			name: "OpenSSL 多字母补丁",
			exp: VersionExp{
				Version:    "1.0.2zj",
				Expression: "[1.0.2z,1.0.2zk)",
			},
			want: true,
		},
		{
			name: "OpenSSL 排他上界",
			exp: VersionExp{
				Version:    "1.1.1t-pre1",
				Expression: "[1.1.1,1.1.1t)",
			},
			want: false,
		},
		{
			name: "OpenSSL 标签前缀",
			exp: VersionExp{
				Version:    "OpenSSL_1_1_1d",
				Expression: "[1.1.1c,1.1.1e]",
			},
			want: true,
		},
		{
			name: "OpenHarmony 标签区间",
			exp: VersionExp{
				Version:    "OpenHarmony-v3.0.3-LTS",
				Expression: "[OpenHarmony-v3.0-LTS,OpenHarmony-v3.0.3-LTS]",
			},
			want: true,
		},
		{
			name: "OpenHarmony 标签超出区间",
			exp: VersionExp{
				Version:    "OpenHarmony-v3.0.5-LTS",
				Expression: "[OpenHarmony-v3.0-LTS,OpenHarmony-v3.0.3-LTS]",
			},
			want: false,
		},
		{
			name: "OpenHarmony 多范围精确标签",
			exp: VersionExp{
				Version:    "OpenHarmony-v3.1-Release",
				Expression: "[OpenHarmony-v3.0-LTS,OpenHarmony-v3.0.3-LTS]||OpenHarmony-v3.1-Release",
			},
			want: true,
		},
		{
			name: "尾零等价",
			exp:  VersionExp{Version: "21", Expression: "[21.0.0]"},
			want: true,
		},
		{
			name: "非零尾段不等价",
			exp:  VersionExp{Version: "21.0.0.1", Expression: "[21.0.0]"},
			want: false,
		},
		{
			name: "干净端点兼容后缀",
			exp:  VersionExp{Version: "21.0.0-pre1-1", Expression: "[21.0.0,21.0.0]"},
			want: true,
		},
		{
			name: "npm scoped 包名前缀",
			exp: VersionExp{
				Version:    "@mattermost/types@8.0.0",
				Expression: "[8.0.0,8.1.3]",
			},
			want: true,
		},
		{
			name: "npm scoped 包超出范围",
			exp: VersionExp{
				Version:    "@mattermost/types@9.0.0",
				Expression: "[8.0.0,8.1.3]",
			},
			want: false,
		},
		{
			name: "npm 预发布包名前缀",
			exp: VersionExp{
				Version:    "@gatsbyjs/parcel-namer-relative-to-cwd@2.2.0-next.0",
				Expression: "[,4.0.8)",
			},
			want: true,
		},
		{
			name: "Conan 仓库后缀",
			exp: VersionExp{
				Version:    "1.5.2@jgsogo/stable#0",
				Expression: "[,4.0.8)",
			},
			want: true,
		},
		{
			name: "通用 OS 版本",
			exp: VersionExp{
				Version:    "252.4-6.an23/aarch64",
				Expression: "(,255-10)",
				Type:       VersionTypeOS,
			},
			want: true,
		},
		{
			name: "空版本不匹配通配符",
			exp:  VersionExp{Version: "", Expression: "*"},
			want: false,
		},
	}

	svc := NewVersionSvc()
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := svc.InExp(test.exp)
			if err != nil {
				t.Fatalf("InExp(%#v) error = %v", test.exp, err)
			}
			if got != test.want {
				t.Fatalf("InExp(%#v) = %v, want %v", test.exp, got, test.want)
			}
		})
	}
}

func TestExpressionSyntaxCompatibility(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		version    string
		expression string
		want       bool
	}{
		{name: "通配符", version: "9.9.9", expression: "*", want: true},
		{name: "裸精确值", version: "0.5.1", expression: "0.5.1", want: true},
		{name: "方括号精确值", version: "12.14", expression: "[12.14]", want: true},
		{name: "单等号", version: "0.97", expression: "=0.97", want: true},
		{name: "双等号", version: "0.97", expression: "==0.97", want: true},
		{name: "小于", version: "1.6.4", expression: "<1.6.5", want: true},
		{name: "小于等于", version: "1.6.5", expression: "<=1.6.5", want: true},
		{name: "大于", version: "1.6.6", expression: ">1.6.5", want: true},
		{name: "大于等于", version: "1.6.5", expression: ">=1.6.5", want: true},
		{name: "逗号双条件", version: "2.2.3", expression: ">=2.2.0, <=2.2.4", want: true},
		{name: "空格双条件", version: "2.60.1", expression: ">=2.60.0 <=2.62.4", want: true},
		{name: "逆序双条件", version: "2.0.1", expression: "<2.1.7 >=2.0.0-beta0", want: true},
		{name: "显式 OR", version: "5.5.8", expression: "[4.0.0,4.0.7)||[5.5.0,5.5.16)", want: true},
		{name: "逗号连接区间", version: "3.1.2", expression: "[,1.0),[3.0,4.0)", want: true},
		{name: "直接相邻区间", version: "3.1.2", expression: "[,1.0)[3.0,4.0)", want: true},
		{name: "逗号精确值", version: "1.35", expression: "2.0.0,3.0.0,v1.35", want: true},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			matcher, err := Compile(test.expression, CompileOptions{})
			if err != nil {
				t.Fatalf("Compile(%q) error = %v", test.expression, err)
			}
			got, err := matcher.Match(test.version)
			if err != nil {
				t.Fatalf("Match(%q) error = %v", test.version, err)
			}
			if got != test.want {
				t.Fatalf("Match(%q) = %v, want %v", test.version, got, test.want)
			}
		})
	}
}

func TestFixedVersionCompatibility(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		exp  VersionExp
		want bool
	}{
		{
			name: "分支修复前",
			exp: VersionExp{
				Version:    "3.16.34-rc1",
				Expression: "[3.13-rc1,4.3-rc1)",
				Fixed:      []string{"3.16:3.16.35", "3.18:3.18.54"},
			},
			want: true,
		},
		{
			name: "分支修复版本排除",
			exp: VersionExp{
				Version:    "3.16.35",
				Expression: "[3.13-rc1,4.3-rc1)",
				Fixed:      []string{"3.16:3.16.35", "3.18:3.18.54"},
			},
			want: false,
		},
		{
			name: "自动推导维护分支",
			exp: VersionExp{
				Version:    "4.19.234",
				Expression: "[4.0,5.0)",
				Fixed:      []string{"4.19.235"},
			},
			want: true,
		},
		{
			name: "自动推导修复版本排除",
			exp: VersionExp{
				Version:    "4.19.235",
				Expression: "[4.0,5.0)",
				Fixed:      []string{"4.19.235"},
			},
			want: false,
		},
	}

	svc := NewVersionSvc()
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := svc.InExp(test.exp)
			if err != nil {
				t.Fatalf("InExp(%#v) error = %v", test.exp, err)
			}
			if got != test.want {
				t.Fatalf("InExp(%#v) = %v, want %v", test.exp, got, test.want)
			}
		})
	}
}

func TestFixedVersionUsesMostSpecificBranchAndOnlyTightens(t *testing.T) {
	t.Parallel()

	matcher, err := Compile("[4.0,5.0)", CompileOptions{FixedVersions: []FixedVersion{
		{Branch: "4", Version: "4.99"},
		{Branch: "4.14", Version: "4.14.272"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	for version, want := range map[string]bool{
		"4.14.271": true,
		"4.14.272": false,
		"4.15.0":   true,
	} {
		got, matchErr := matcher.Match(version)
		if matchErr != nil {
			t.Fatalf("Match(%q) error = %v", version, matchErr)
		}
		if got != want {
			t.Errorf("Match(%q) = %v, want %v", version, got, want)
		}
	}

	nonExpanding, err := Compile("[4.0,4.5)", CompileOptions{FixedVersions: []FixedVersion{
		{Branch: "4", Version: "4.9"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	got, err := nonExpanding.Match("4.6")
	if err != nil {
		t.Fatal(err)
	}
	if got {
		t.Fatal("a fixed version later than the original upper bound expanded the affected range")
	}
}

func TestCompareCompatibilityResults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		left, right string
		want        int
	}{
		{name: "小于", left: "1.0.9", right: "1.0.10", want: int(CompareLess)},
		{name: "尾零相等", left: "21", right: "21.0.0", want: int(CompareEqual)},
		{name: "大于", left: "2.0", right: "1.9.9", want: int(CompareGreater)},
		{name: "RC 小于正式版", left: "6.7-rc1", right: "6.7", want: int(CompareLess)},
		{name: "OpenSSL 补丁字母更大", left: "1.1.1a", right: "1.1.1", want: int(CompareGreater)},
		{name: "宽松包名前缀", left: "@mattermost/types@9.0.0", right: "8.1.3", want: int(CompareGreater)},
		{name: "宽松仓库后缀", left: "1.5.2@jgsogo/stable#0", right: "1.5.1", want: int(CompareGreater)},
	}

	svc := NewVersionSvc()
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := svc.Compare(test.left, test.right)
			if err != nil {
				t.Fatalf("Compare(%q, %q) error = %v", test.left, test.right, err)
			}
			if got != test.want {
				t.Fatalf("Compare(%q, %q) = %d, want %d", test.left, test.right, got, test.want)
			}
		})
	}
}

func TestCompatibilityErrors(t *testing.T) {
	t.Parallel()

	invalidExpressions := []string{
		"",
		"=>1.0",
		"[1.0,2.0",
		"1.0,,2.0",
		">=1.0,>=2.0",
		"[2.0,1.0]",
		"(,)",
	}
	for _, expression := range invalidExpressions {
		expression := expression
		t.Run("表达式_"+expression, func(t *testing.T) {
			t.Parallel()
			if _, err := Compile(expression, CompileOptions{}); !errors.Is(err, ErrInvalidExpression) {
				t.Fatalf("Compile(%q) error = %v, want ErrInvalidExpression", expression, err)
			}
		})
	}

	svc := NewVersionSvc()
	if _, err := svc.InExp(VersionExp{Version: "1.0", Expression: "*", Type: "semver"}); !errors.Is(err, ErrUnsupportedVersionType) {
		t.Fatalf("InExp() error = %v, want ErrUnsupportedVersionType", err)
	}
	if _, err := svc.Compare("", "1.0"); !errors.Is(err, ErrEmptyVersion) {
		t.Fatalf("Compare() error = %v, want ErrEmptyVersion", err)
	}
	if _, err := svc.Compare("master", "main"); !errors.Is(err, ErrIncomparableVersion) {
		t.Fatalf("Compare() error = %v, want ErrIncomparableVersion", err)
	}

	matcher, err := Compile("[1.0,2.0)", CompileOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = matcher.Match("???"); !errors.Is(err, ErrInvalidVersion) {
		t.Fatalf("Matcher.Match() error = %v, want ErrInvalidVersion", err)
	}
	var zero *Matcher
	if _, err = zero.Match("1.0"); !errors.Is(err, ErrInvalidExpression) {
		t.Fatalf("nil Matcher.Match() error = %v, want ErrInvalidExpression", err)
	}

	if _, err = Compile("[1.0,2.0)", CompileOptions{FixedVersions: []FixedVersion{
		{Branch: "1.0", Version: "1.5"},
		{Branch: "01.00", Version: "1.6"},
	}}); !errors.Is(err, ErrInvalidExpression) {
		t.Fatalf("Compile() duplicate normalized fixed branch error = %v, want ErrInvalidExpression", err)
	}
}
