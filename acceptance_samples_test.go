//go:build integration

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

type acceptanceSample struct {
	command string
	output  string
}

// Read the documentation itself, rather than a second set of golden files.
// Git's version-dependent checkout chatter is excluded; hook messages are exact.
func acceptanceSamples() (map[string]acceptanceSample, error) {
	data, err := os.ReadFile("docs-eb/PRODUCT.md")
	if err != nil {
		return nil, err
	}
	samples := make(map[string]acceptanceSample)
	section, command := "", ""
	var lines []string
	console, ordinal, indent := false, 0, 0
	flush := func() {
		output := strings.TrimRight(strings.Join(lines, "\n"), "\n")
		if strings.HasPrefix(command, "envbuckets ") || strings.HasPrefix(command, "git ") && strings.Contains(output, "envbuckets:") {
			ordinal++
			if output != "" {
				output += "\n"
			}
			samples[fmt.Sprintf("%s/%d", section, ordinal)] = acceptanceSample{command, output}
		}
		command, lines = "", nil
	}
	for line := range strings.SplitSeq(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "### ") || strings.HasPrefix(trimmed, "#### "):
			flush()
			section = strings.TrimLeft(trimmed, "# ")
			ordinal = 0
		case trimmed == "```console":
			console = true
			indent = len(line) - len(strings.TrimLeft(line, " "))
		case console && trimmed == "```":
			flush()
			console = false
		case console && strings.HasPrefix(trimmed, "$ "):
			flush()
			command = strings.TrimPrefix(trimmed, "$ ")
		case console && strings.HasPrefix(trimmed, "#"):
			flush()
		case console && command != "":
			if len(line) >= indent {
				line = line[indent:]
			}
			lines = append(lines, line)
		}
	}
	return samples, nil
}

func cmdSample(ts *testscript.TestScript, neg bool, args []string) {
	if neg || len(args) != 2 {
		ts.Fatalf("usage: sample 'PRODUCT heading' ordinal")
	}
	key := args[0] + "/" + args[1]
	want, ok := ts.Value("acceptance-samples").(map[string]acceptanceSample)[key]
	if !ok {
		ts.Fatalf("unknown PRODUCT.md sample %q", key)
	}
	got := ts.ReadFile("stdout") + ts.ReadFile("stderr")
	if strings.HasPrefix(want.command, "git ") {
		got, want.output = hookLines(got), hookLines(want.output)
	}
	// These are presentation substitutions, not changes to message text.
	got = strings.ReplaceAll(strings.ReplaceAll(got, "\r\n", "\n"), "\t", "        ")
	got = strings.ReplaceAll(got, filepath.ToSlash(ts.MkAbs(".")), "/work/shop")
	got = strings.ReplaceAll(got, ts.MkAbs("."), "/work/shop")
	if got != want.output {
		ts.Fatalf("PRODUCT.md %s ($ %s):\ngot:\n%s\nwant:\n%s", key, want.command, got, want.output)
	}
}

func hookLines(output string) string {
	var lines []string
	for line := range strings.SplitSeq(output, "\n") {
		if strings.HasPrefix(line, "envbuckets:") {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n") + "\n"
}

// Fails when a documentation example is added without a script assertion.
func TestAcceptanceSampleCoverage(t *testing.T) {
	samples, err := acceptanceSamples()
	if err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob("testdata/script/acceptance-*.txtar")
	if err != nil {
		t.Fatal(err)
	}
	seen := make(map[string]bool)
	assertion := regexp.MustCompile(`(?m)^(?:! )?exec ([^\n]+)\nsample '((?:[^']|'')+)' ([0-9]+)$`)
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for _, match := range assertion.FindAllStringSubmatch(string(data), -1) {
			key := strings.ReplaceAll(match[2], "''", "'") + "/" + match[3]
			if sample, ok := samples[key]; !ok {
				t.Errorf("%s references missing sample %s", file, key)
			} else if match[1] != sample.command {
				t.Errorf("%s executes %q for %s, documented command is %q", file, match[1], key, sample.command)
			}
			seen[key] = true
		}
	}
	if len(samples) == 0 {
		t.Fatal("PRODUCT.md contains no samples")
	}
	for key, sample := range samples {
		if !seen[key] {
			t.Errorf("sample %s ($ %s) has no integration assertion", key, sample.command)
		}
	}
}

// Git's perf fixture uses one update-ref --stdin process for all extra refs.
func cmdBranches1000(ts *testscript.TestScript, neg bool, args []string) {
	if neg || len(args) != 0 {
		ts.Fatalf("usage: branches1000")
	}
	ts.Check(ts.Exec("git", "for-each-ref", "--format=%(refname)", "refs/heads/"))
	count := len(strings.Fields(ts.ReadFile("stdout")))
	if count > 1000 {
		ts.Fatalf("already have %d branches", count)
	}
	var refs strings.Builder
	for i := count; i < 1000; i++ {
		fmt.Fprintf(&refs, "create refs/heads/feature/%04d HEAD\n", i)
	}
	cmd := exec.Command(ts.Getenv("GIT_EXE"), "update-ref", "--stdin") //nolint:gosec // Setup resolves the trusted Git executable; no shell.
	cmd.Dir, cmd.Stdin = ts.MkAbs("."), strings.NewReader(refs.String())
	for _, key := range []string{"PATH", "SYSTEMROOT", "HOME", "USERPROFILE", "XDG_CONFIG_HOME", "GIT_CONFIG_GLOBAL", "GIT_CONFIG_NOSYSTEM"} {
		cmd.Env = append(cmd.Env, key+"="+ts.Getenv(key))
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		ts.Fatalf("update-ref: %v %s", err, out)
	}
}
