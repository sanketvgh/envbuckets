package cli

import (
	"regexp"
	"strings"
	"testing"
)

func TestGroupHelp(t *testing.T) {
	cases := map[string][]string{
		"Usage: envbuckets map move <pattern>":   {"map", "move", "x", "--help"},
		"Usage: envbuckets map add|update":       {"map", "--help"},
		"Usage: envbuckets bucket list":          {"bucket", "list", "--help"},
		"Usage: envbuckets scope purge <path>":   {"scope", "purge"},
		"Usage: envbuckets init [--into":         {"init"},
		"Usage: envbuckets bucket add|rm|list":   {"bucket", "bogus"},
		"Usage: envbuckets map explain <branch>": {"map", "explain"},
	}
	nonASCII := regexp.MustCompile(`[^\x00-\x7F]`)
	for want, args := range cases {
		text, ok := groupHelp(args[0], args[1:])
		if !ok || !strings.HasPrefix(text, want) {
			t.Fatalf("%v: got %q", args, text)
		}
	}
	for key, text := range groupHelpText {
		if nonASCII.MatchString(text) {
			t.Fatalf("%s help is not ASCII", key)
		}
	}
	if _, ok := groupHelp("status", nil); ok {
		t.Fatal("status is not a B-owned group")
	}
}
