package pattern

import "testing"

func TestSetMatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		patterns []string
		path     string
		want     bool
	}{
		{"double star matches nested file", []string{"**/*.pyc"}, "a/b/cache.pyc", true},
		{"double star matches root file", []string{"**/*.pyc"}, "cache.pyc", true},
		{"directory tail matches directory", []string{"cache/**"}, "cache", true},
		{"single star does not cross slash", []string{"*.log"}, "nested/a.log", false},
		{"question mark", []string{"hooks/hook?.sh"}, "hooks/hook1.sh", true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			set, err := Compile(test.patterns)
			if err != nil {
				t.Fatalf("Compile(%v): %v", test.patterns, err)
			}
			if got := set.Match(test.path); got != test.want {
				t.Errorf("Set.Match(%q) = %v, want %v", test.path, got, test.want)
			}
		})
	}
}

func TestCompileRejectsPartialDoubleStar(t *testing.T) {
	t.Parallel()

	if _, err := Compile([]string{"foo**bar"}); err == nil {
		t.Fatal("Compile(foo**bar) succeeded, want error")
	}
}
