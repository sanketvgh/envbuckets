package pattern

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type gitCase struct {
	Pattern string `json:"pattern"`
	Branch  string `json:"branch"`
	Match   bool   `json:"match"`
}

var gitCases = []gitCase{
	{"main", "main", true},
	{"main", "main-2", false},
	{"*", "anything", true},
	{"*", "anything/at/all", false},
	{"?", "x", true},
	{"?", "xx", false},
	{"release/*", "release/1.2", true},
	{"release/*", "release/1.2/rc1", false},
	{"release/**", "release/1.2/rc1", true},
	{"release/**", "release", false},
	{"release/", "release/1.2/rc1", true},
	{"release/", "release", false},
	{"**/hotfix", "hotfix", true},
	{"**/hotfix", "team/hotfix", true},
	{"**/hotfix", "hotfix-2", false},
	{"**/spike-*", "spike-cache", true},
	{"**/spike-*", "alice/spike-cache", true},
	{"**/spike-*", "alice/fix-spikes", false},
	{"v?.?", "v1.2", true},
	{"v?.?", "v1.10", false},
	{"hotfix/[0-9]*", "hotfix/42-login", true},
	{"hotfix/[0-9]*", "hotfix/login", false},
	{"hotfix/[!a-c]x", "hotfix/zx", true},
	{"hotfix/[!a-c]x", "hotfix/bx", false},
	{"hotfix/[^a-c]x", "hotfix/zx", true},
	{"hotfix/[^a-c]x", "hotfix/bx", false},
	{"x/[[:digit:]]", "x/7", true},
	{"x/[[:digit:]]", "x/a", false},
	{"a**b", "axb", true},
	{"a**b", "a/x/b", false},
	{"a**/b", "a/x/b", false},
	{"a/**/b", "a/x/y/b", true},
	{"a/**/b", "a/b", true},
	{"[]a]", "]", true},
	{"[]a]", "a", true},
	{"[]a]", "b", false},
	{"[!]a]", "]", false},
	{"[!]a]", "a", false},
	{"[!]a]", "b", true},
	{"{main,master}", "main", false},
}

func TestGitFixture(t *testing.T) {
	fixture := filepath.Join("testdata", "git-patterns.json")
	if os.Getenv("UPDATE_GIT_FIXTURE") == "1" {
		answers, err := runGitCases(t)
		if err != nil {
			t.Fatal(err)
		}
		data, err := json.MarshalIndent(answers, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fixture, append(data, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	data, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	var recorded []gitCase
	if err := json.Unmarshal(data, &recorded); err != nil {
		t.Fatal(err)
	}
	if len(recorded) != len(gitCases) {
		t.Fatalf("fixture has %d cases, want %d; regenerate with task pattern:fixture", len(recorded), len(gitCases))
	}
	for i, tc := range recorded {
		if tc.Pattern != gitCases[i].Pattern || tc.Branch != gitCases[i].Branch {
			t.Fatalf("fixture case %d = %#v, source = %#v", i, tc, gitCases[i])
		}
		if got := Match(tc.Pattern, tc.Branch); got != tc.Match {
			t.Errorf("Match(%q, %q) = %v, Git fixture says %v", tc.Pattern, tc.Branch, got, tc.Match)
		}
	}
}

func runGitCases(t *testing.T) ([]gitCase, error) {
	t.Helper()
	dir := t.TempDir()
	if err := gitRun(dir, "init", "-q"); err != nil {
		return nil, err
	}
	mainConfig := filepath.Join(dir, "main.gitconfig")
	matchedConfig := filepath.Join(dir, "matched.gitconfig")
	if err := os.WriteFile(matchedConfig, []byte("[envbuckets]\n matched = true\n"), 0o600); err != nil {
		return nil, err
	}
	answers := append([]gitCase(nil), gitCases...)
	for i := range answers {
		tc := &answers[i]
		sectionPattern := strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(tc.Pattern)
		config := "[includeIf \"onbranch:" + sectionPattern + "\"]\n path = matched.gitconfig\n"
		if err := os.WriteFile(mainConfig, []byte(config), 0o600); err != nil {
			return nil, err
		}
		if err := gitRun(dir, "symbolic-ref", "HEAD", "refs/heads/"+tc.Branch); err != nil {
			return nil, err
		}
		cmd := exec.Command("git", "-C", dir, "-c", "include.path="+mainConfig, "config", "--bool", "envbuckets.matched") //nolint:gosec // fixed Git argv and temp paths
		var stderr strings.Builder
		cmd.Stderr = &stderr
		out, err := cmd.Output()
		if err == nil {
			tc.Match = strings.TrimSpace(string(out)) == "true"
			continue
		}
		var exit *exec.ExitError
		if errors.As(err, &exit) && exit.ExitCode() == 1 {
			tc.Match = false
			continue
		}
		return nil, &gitCommandError{args: []string{"config", tc.Pattern, tc.Branch}, output: stderr.String(), err: err}
	}
	return answers, nil
}

func gitRun(dir string, args ...string) error {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...) //nolint:gosec // test-only Git command with fixed test arguments
	if out, err := cmd.CombinedOutput(); err != nil {
		return &gitCommandError{args: args, output: string(out), err: err}
	}
	return nil
}

type gitCommandError struct {
	args   []string
	output string
	err    error
}

func (e *gitCommandError) Error() string {
	return strings.Join(e.args, " ") + ": " + e.err.Error() + ": " + e.output
}
func (e *gitCommandError) Unwrap() error { return e.err }
