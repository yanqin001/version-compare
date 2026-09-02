package rangeexpr

import (
	"reflect"
	"testing"
)

func TestParseValidExpressions(t *testing.T) {
	exact := func(version string) Interval {
		bound := Bound{Version: version, Inclusive: true, Set: true}
		return Interval{Lower: bound, Upper: bound}
	}
	lower := func(version string, inclusive bool) Bound {
		return Bound{Version: version, Inclusive: inclusive, Set: true}
	}
	upper := lower

	tests := []struct {
		name  string
		input string
		want  Expression
	}{
		{name: "通配符", input: "*", want: Expression{Any: true}},
		{name: "裸精确值", input: "1.2.3", want: Expression{Intervals: []Interval{exact("1.2.3")}}},
		{name: "方括号精确值", input: "[1.2.3]", want: Expression{Intervals: []Interval{exact("1.2.3")}}},
		{name: "单等号精确值", input: "=1.2.3", want: Expression{Intervals: []Interval{exact("1.2.3")}}},
		{name: "双等号精确值", input: "==1.2.3", want: Expression{Intervals: []Interval{exact("1.2.3")}}},
		{
			name:  "闭区间",
			input: "[1.0,2.0]",
			want: Expression{Intervals: []Interval{{
				Lower: lower("1.0", true),
				Upper: upper("2.0", true),
			}}},
		},
		{
			name:  "左闭右开区间",
			input: "[1.0,2.0)",
			want: Expression{Intervals: []Interval{{
				Lower: lower("1.0", true),
				Upper: upper("2.0", false),
			}}},
		},
		{
			name:  "左开右闭区间",
			input: "(1.0,2.0]",
			want: Expression{Intervals: []Interval{{
				Lower: lower("1.0", false),
				Upper: upper("2.0", true),
			}}},
		},
		{
			name:  "开区间",
			input: "(1.0,2.0)",
			want: Expression{Intervals: []Interval{{
				Lower: lower("1.0", false),
				Upper: upper("2.0", false),
			}}},
		},
		{
			name:  "空下界",
			input: "[,2.0)",
			want: Expression{Intervals: []Interval{{
				Upper: upper("2.0", false),
			}}},
		},
		{
			name:  "空上界",
			input: "(1.0,]",
			want: Expression{Intervals: []Interval{{
				Lower: lower("1.0", false),
			}}},
		},
		{
			name:  "双空端点仍保留为区间",
			input: "(,)",
			want:  Expression{Intervals: []Interval{{}}},
		},
		{
			name:  "小于",
			input: "<2.0",
			want:  Expression{Intervals: []Interval{{Upper: upper("2.0", false)}}},
		},
		{
			name:  "小于等于",
			input: "<= 2.0",
			want:  Expression{Intervals: []Interval{{Upper: upper("2.0", true)}}},
		},
		{
			name:  "大于",
			input: ">1.0",
			want:  Expression{Intervals: []Interval{{Lower: lower("1.0", false)}}},
		},
		{
			name:  "大于等于",
			input: ">= 1.0",
			want:  Expression{Intervals: []Interval{{Lower: lower("1.0", true)}}},
		},
		{
			name:  "逗号双条件",
			input: ">=1.0, <2.0",
			want: Expression{Intervals: []Interval{{
				Lower: lower("1.0", true),
				Upper: upper("2.0", false),
			}}},
		},
		{
			name:  "空格双条件",
			input: ">=1.0 <2.0",
			want: Expression{Intervals: []Interval{{
				Lower: lower("1.0", true),
				Upper: upper("2.0", false),
			}}},
		},
		{
			name:  "逆序双条件",
			input: "<2.0 >=1.0",
			want: Expression{Intervals: []Interval{{
				Lower: lower("1.0", true),
				Upper: upper("2.0", false),
			}}},
		},
		{
			name:  "显式OR",
			input: "1.0 || [2.0,3.0)",
			want: Expression{Intervals: []Interval{
				exact("1.0"),
				{Lower: lower("2.0", true), Upper: upper("3.0", false)},
			}},
		},
		{
			name:  "逗号连接区间",
			input: "(,1.0],[2.0,3.0)",
			want: Expression{Intervals: []Interval{
				{Upper: upper("1.0", true)},
				{Lower: lower("2.0", true), Upper: upper("3.0", false)},
			}},
		},
		{
			name:  "直接相邻区间",
			input: "(,1.0][2.0,3.0)",
			want: Expression{Intervals: []Interval{
				{Upper: upper("1.0", true)},
				{Lower: lower("2.0", true), Upper: upper("3.0", false)},
			}},
		},
		{
			name:  "逗号精确值列表",
			input: "1.0, 2.0,v3",
			want:  Expression{Intervals: []Interval{exact("1.0"), exact("2.0"), exact("v3")}},
		},
		{
			name:  "等号精确值列表",
			input: "=1.0, ==2.0,=v3",
			want:  Expression{Intervals: []Interval{exact("1.0"), exact("2.0"), exact("v3")}},
		},
		{
			name:  "复杂版本字符",
			input: "1:2.0~rc1-3+ubuntu1/amd64",
			want:  Expression{Intervals: []Interval{exact("1:2.0~rc1-3+ubuntu1/amd64")}},
		},
		{
			name:  "通配符规范化OR结果",
			input: "1.0||*||[2.0,3.0)",
			want:  Expression{Any: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("Parse(%q) = %#v, want %#v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseInvalidExpressions(t *testing.T) {
	tests := []string{
		"",
		"   ",
		"||1.0",
		"1.0||",
		"1.0|2.0",
		"1.0||||2.0",
		"[1.0,2.0",
		"1.0,2.0]",
		"[1.0,[2.0]]",
		"(1.0)",
		"[]",
		"[1.0,2.0,3.0]",
		"[1.0 <2.0]",
		"<",
		">=",
		"=>1.0",
		"=<1.0",
		"===1.0",
		"!=1.0",
		">=1.0<=2.0",
		">=1.0,",
		">=1.0,2.0",
		">=1.0,>=2.0",
		"<1.0,<=2.0",
		">=1.0,=2.0",
		">=1.0,<2.0,<3.0",
		"=1.0 =2.0",
		"=1.0,>=2.0",
		"1.0,",
		",1.0",
		"1.0,,2.0",
		"1.0 2.0",
		"1.0[2.0]",
		"[1.0],2.0 [3.0]",
		"* ,1.0",
		"[1.0||2.0]",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			if got, err := Parse(input); err == nil {
				t.Fatalf("Parse(%q) = %#v, want error", input, got)
			}
		})
	}
}
