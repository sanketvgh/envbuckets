package pattern

import "testing"

func TestMatch(t *testing.T) {
	cases := []struct {
		pattern, name string
		want          bool
	}{
		{"main", "main", true},
		{"main", "maine", false},
		{"*", "anything/at/all", true},
		{"*", "", true},
		{"release/*", "release/1.2/x", true},
		{"release/*", "release/", true},
		{"release/*", "release", false},
		{"feature/*", "release/1", false},
		{"*-hotfix", "v1.2-hotfix", true},
		{"a*b*c", "aXXbYYc", true},
		{"a*b*c", "acb", false},
		{"a**c", "abc", true},
	}
	for _, c := range cases {
		if got := Match(c.pattern, c.name); got != c.want {
			t.Errorf("Match(%q, %q) = %v, want %v", c.pattern, c.name, got, c.want)
		}
	}
}
