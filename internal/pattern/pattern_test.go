package pattern

import "testing"

func TestMatch(t *testing.T) {
	cases := []struct {
		pattern, name string
		want          bool
	}{
		{"main", "main", true},
		{"main", "maine", false},
		{"*", "anything/at/all", false},
		{"*", "anything", true},
		{"?", "x", true},
		{"?", "/", false},
		{"release/*", "release/1.2", true},
		{"release/*", "release/1.2/x", false},
		{"release/**", "release/1.2/x", true},
		{"**/spike-*", "spike-cache", true},
		{"**/spike-*", "alice/spike-cache", true},
		{"a**b", "a/x/b", false},
		{"a**/b", "a/x/b", false},
		{"a/**/b", "a/x/y/b", true},
		{"a/**/b", "a/b", true},
		{"a/**", "a/", true},
		{"release/", "release/1.2/rc", true},
		{"hotfix/[0-9]*", "hotfix/88-timeout", true},
		{"hotfix/[!a-c]x", "hotfix/zx", true},
		{"hotfix/[!a-c]x", "hotfix/bx", false},
		{"[]a]", "]", true},
		{"[]a]", "a", true},
		{"[]a]", "b", false},
		{"[!]a]", "]", false},
		{"[!]a]", "b", true},
		{"x/[[:digit:]]", "x/7", true},
		{`release/\*`, "release/*", true},
		{`\[`, "[", true},
		{"{main,master}", "main", false},
	}
	for _, tc := range cases {
		if got := Match(tc.pattern, tc.name); got != tc.want {
			t.Errorf("Match(%q, %q) = %v, want %v", tc.pattern, tc.name, got, tc.want)
		}
	}
}

func TestValidate(t *testing.T) {
	for _, p := range []string{"[", "[]", "trailing\\"} {
		if Validate(p) == nil {
			t.Errorf("Validate(%q) unexpectedly succeeded", p)
		}
	}
	for _, p := range []string{"release/*", "[!a-z]", "[[:digit:]]", `release/\*`} {
		if err := Validate(p); err != nil {
			t.Errorf("Validate(%q): %v", p, err)
		}
	}
}
