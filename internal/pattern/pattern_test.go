package pattern

import (
	"strings"
	"testing"
)

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

func TestCompileRejectsExcessiveComponents(t *testing.T) {
	t.Parallel()

	parts := make([]string, maxComponents+1)
	for i := range parts {
		parts[i] = "a"
	}
	pattern := strings.Join(parts, "/")
	if _, err := Compile([]string{pattern}); err == nil {
		t.Fatal("Compile(<excessive components>) succeeded, want error")
	}

	// One at the ceiling is still accepted.
	atLimit := strings.Join(parts[:maxComponents], "/")
	set, err := Compile([]string{atLimit})
	if err != nil {
		t.Fatalf("Compile(at limit): %v", err)
	}
	if !set.Match(atLimit) {
		t.Fatal("Compile(at limit).Match(at limit) = false, want true")
	}
}

func TestMatchRepeatedDoubleStar(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		patterns []string
		path     string
		want     bool
	}{
		{"two ** require both anchors", []string{"**/a/**/b"}, "x/a/y/b", true},
		{"two ** miss a missing anchor", []string{"**/a/**/b"}, "x/a/y/c", false},
		{"repeated adjacent ** collapses", []string{"a/**/**/b"}, "a/b", true},
		{"repeated adjacent ** matches deep", []string{"a/**/**/b"}, "a/1/2/3/b", true},
		{"three ** deep path", []string{"**/x/**/y/**/z"}, "p/x/q/y/r/z", true},
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

func TestMatchDeepPathManyDoubleStars(t *testing.T) {
	t.Parallel()

	// A deep nonmatching path with several repeating ** components would
	// recurse exponentially without memoization; it must stay correct and
	// terminate. This guards the CWE-400 regression in match.
	parts := make([]string, 0, 200)
	for i := 0; i < 200; i++ {
		parts = append(parts, "component")
	}
	name := strings.Join(parts, "/")
	patterns := []string{
		"**/**/x/**/**",
		"**/y/**/**/z/**",
		"a/**/**/**/**/q",
	}
	set, err := Compile(patterns)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if set.Match(name) {
		t.Fatal("Set.Match(deep nonmatching path) = true, want false")
	}
}
