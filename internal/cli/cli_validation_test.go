package cli

import (
	"strings"
	"testing"
)

func TestAffectedCommandHelpNeedsNoProject(t *testing.T) {
	r := newRepo(t)
	for _, args := range [][]string{{"status", "--help"}, {"check", "--help"}, {"apply", "--help"}, {"use", "--help"}, {"link", "--help"}, {"unlink", "--help"}} {
		res := r.run(args...)
		if res.code != ExitOK || !strings.Contains(res.stdout, "Usage: envbuckets "+args[0]) {
			t.Errorf("%v: %d %s", args, res.code, res.all())
		}
	}
}

func TestAffectedCommandRejectsExtraInput(t *testing.T) {
	r := newRepo(t)
	for _, args := range [][]string{{"status", "extra"}, {"check", "extra"}, {"apply", "extra"}, {"check", "--irrelevant"}, {"apply", "--all"}} {
		res := r.run(args...)
		if res.code != ExitUsage {
			t.Errorf("%v: want usage, got %d %s", args, res.code, res.all())
		}
	}
}
