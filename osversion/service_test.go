package osversion

import (
	"errors"
	"testing"
)

func TestServiceCompareResultCompatibility(t *testing.T) {
	t.Parallel()

	service := NewService()
	tests := []struct {
		left  string
		right string
		want  int
	}{
		{left: "1.0~rc1-1", right: "1.0-1", want: int(CompareLess)},
		{left: "1.0-1", right: "1.0-1", want: int(CompareEqual)},
		{left: "1.0-2", right: "1.0-1", want: int(CompareGreater)},
	}
	for _, test := range tests {
		got, err := service.Compare(DistributionRPM, test.left, test.right)
		if err != nil {
			t.Fatalf("Compare() error = %v", err)
		}
		if got != test.want {
			t.Fatalf("Compare(%q, %q) = %d, want %d", test.left, test.right, got, test.want)
		}
	}
}

func TestServiceMatchOSVersion(t *testing.T) {
	t.Parallel()

	service := NewService()
	matched, err := service.MatchOSVersion(DistributionAnolis, "Anolis OS 23", "an23")
	if err != nil {
		t.Fatal(err)
	}
	if !matched {
		t.Fatal("MatchOSVersion() = false, want true")
	}
}

func TestServiceCompareEmptyVersion(t *testing.T) {
	t.Parallel()

	_, err := NewService().Compare(DistributionRPM, "", "1.0")
	if !errors.Is(err, ErrEmptyVersion) {
		t.Fatalf("Compare() error = %v, want ErrEmptyVersion", err)
	}
}
