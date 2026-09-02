package versioncompare

import (
	"errors"
	"reflect"
	"testing"
)

func TestSortIsStableAndInPlace(t *testing.T) {
	t.Parallel()

	versions := []string{
		"2.0.0",
		"1.0.0",
		"1",
		"1.0",
		"2",
		"1.0.0.0",
		"1.0.1",
		"2.0",
	}
	want := []string{
		"1.0.0",
		"1",
		"1.0",
		"1.0.0.0",
		"1.0.1",
		"2.0.0",
		"2",
		"2.0",
	}

	sorted, err := NewVersionSvc().Sort(versions)
	if err != nil {
		t.Fatalf("Sort() error = %v", err)
	}
	if &sorted[0] != &versions[0] {
		t.Fatal("Sort() returned a different backing slice")
	}
	if !reflect.DeepEqual(sorted, want) {
		t.Fatalf("Sort() = %v, want %v", sorted, want)
	}
}

func TestSortDoesNotPartiallyMutateOnError(t *testing.T) {
	t.Parallel()

	versions := []string{"2.0", "master", "1.0"}
	original := append([]string(nil), versions...)
	sorted, err := NewVersionSvc().Sort(versions)
	if !errors.Is(err, ErrIncomparableVersion) {
		t.Fatalf("Sort() error = %v, want ErrIncomparableVersion", err)
	}
	if sorted != nil {
		t.Fatalf("Sort() result = %v, want nil on error", sorted)
	}
	if !reflect.DeepEqual(versions, original) {
		t.Fatalf("Sort() mutated input on error: got %v, want %v", versions, original)
	}
}

func TestMinAndMaxVersion(t *testing.T) {
	t.Parallel()

	svc := NewVersionSvc()
	versions := []string{"1.0.1", "3.0.2", "2.0.3", "3.0.1"}
	minimum, err := svc.GetMinVersion(versions)
	if err != nil {
		t.Fatalf("GetMinVersion() error = %v", err)
	}
	if minimum != "1.0.1" {
		t.Fatalf("GetMinVersion() = %q, want %q", minimum, "1.0.1")
	}
	maximum, err := svc.GetMaxVersion(versions)
	if err != nil {
		t.Fatalf("GetMaxVersion() error = %v", err)
	}
	if maximum != "3.0.2" {
		t.Fatalf("GetMaxVersion() = %q, want %q", maximum, "3.0.2")
	}

	patches := []string{"1.1.1a", "1.1.1t", "1.1.1f"}
	maximum, err = svc.GetMaxVersion(patches)
	if err != nil {
		t.Fatalf("GetMaxVersion(OpenSSL) error = %v", err)
	}
	if maximum != "1.1.1t" {
		t.Fatalf("GetMaxVersion(OpenSSL) = %q, want %q", maximum, "1.1.1t")
	}

	// 数值等价版本必须保留最先出现的原始文本。
	equivalent := []string{"1.0.0", "1", "1.0"}
	minimum, err = svc.GetMinVersion(equivalent)
	if err != nil {
		t.Fatal(err)
	}
	maximum, err = svc.GetMaxVersion(equivalent)
	if err != nil {
		t.Fatal(err)
	}
	if minimum != equivalent[0] || maximum != equivalent[0] {
		t.Fatalf("equivalent Min/Max = %q/%q, want first value %q", minimum, maximum, equivalent[0])
	}
}

func TestAggregateEmptyListErrors(t *testing.T) {
	t.Parallel()

	svc := NewVersionSvc()
	if _, err := svc.GetMinVersion(nil); !errors.Is(err, ErrEmptyVersionList) {
		t.Fatalf("GetMinVersion(nil) error = %v, want ErrEmptyVersionList", err)
	}
	if _, err := svc.GetMaxVersion(nil); !errors.Is(err, ErrEmptyVersionList) {
		t.Fatalf("GetMaxVersion(nil) error = %v, want ErrEmptyVersionList", err)
	}
	if _, err := svc.GetClosestVersion("1.0", nil); !errors.Is(err, ErrEmptyVersionList) {
		t.Fatalf("GetClosestVersion(nil) error = %v, want ErrEmptyVersionList", err)
	}
}

func TestClosestVersionCompatibility(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		target   string
		versions []string
		want     string
	}{
		{name: "完全匹配", target: "1.2.3", versions: []string{"1.2.1", "1.2.3", "1.2.5"}, want: "1.2.3"},
		{name: "较高版本更近", target: "1.2.5", versions: []string{"1.2.3", "1.2.6", "1.3.0"}, want: "1.2.6"},
		{name: "较低版本更近", target: "1.2.5", versions: []string{"1.2.4", "1.2.0", "1.3.0"}, want: "1.2.4"},
		{name: "低于全部候选", target: "1.0.0", versions: []string{"1.2.3", "2.0.0", "3.0.0"}, want: "1.2.3"},
		{name: "高于全部候选", target: "5.0.0", versions: []string{"1.2.3", "2.0.0", "3.0.0"}, want: "3.0.0"},
		{name: "单个候选", target: "1.5.0", versions: []string{"2.0.0"}, want: "2.0.0"},
		{name: "等距时取较高版本", target: "1.2.5", versions: []string{"1.2.4", "1.2.6"}, want: "1.2.6"},
		{name: "多段版本", target: "2.3.4", versions: []string{"1.0.0", "2.3.3", "2.3.5", "3.0.0"}, want: "2.3.5"},
		{name: "主版本权重", target: "2.0.0", versions: []string{"1.9.9", "3.0.0"}, want: "3.0.0"},
		{name: "尾零等价取首个", target: "1", versions: []string{"1.0.0", "1.0", "2.0"}, want: "1.0.0"},
	}

	svc := NewVersionSvc()
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			original := append([]string(nil), test.versions...)
			got, err := svc.GetClosestVersion(test.target, test.versions)
			if err != nil {
				t.Fatalf("GetClosestVersion(%q, %v) error = %v", test.target, test.versions, err)
			}
			if got != test.want {
				t.Fatalf("GetClosestVersion(%q, %v) = %q, want %q", test.target, test.versions, got, test.want)
			}
			if !reflect.DeepEqual(test.versions, original) {
				t.Fatalf("GetClosestVersion() mutated versions: got %v, want %v", test.versions, original)
			}
		})
	}
}
