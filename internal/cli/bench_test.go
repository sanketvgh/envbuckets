package cli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// BenchmarkReports includes Git process startup. Hooks inspect only the current
// branch and working tree; branches enumerates every local branch once.
func BenchmarkReports(b *testing.B) {
	if testing.Short() {
		b.Skip("branch benchmarks are excluded from short runs")
	}
	for _, count := range []int{1, 1000} {
		for _, command := range []string{"branches", "hook"} {
			b.Run(fmt.Sprintf("%s/%d", command, count), func(b *testing.B) {
				r := newRepo(b)
				r.write(".envbuckets.json", switchConfig)
				if err := os.MkdirAll(r.path(".env.d/dev"), 0o755); err != nil {
					b.Fatal(err)
				}
				var refs strings.Builder
				for i := 1; i < count; i++ {
					fmt.Fprintf(&refs, "create refs/heads/feature/%04d HEAD\n", i)
				}
				cmd := exec.Command("git", "update-ref", "--stdin")
				cmd.Dir, cmd.Stdin = r.root, strings.NewReader(refs.String())
				if out, err := cmd.CombinedOutput(); err != nil {
					b.Fatalf("update-ref: %v %s", err, out)
				}
				args := []string{"branches", "**"}
				if command == "hook" {
					args = []string{"hook", "old", "new", "1"}
				}
				env := Env{Cwd: r.root, Stdout: io.Discard, Stderr: io.Discard}
				b.ReportAllocs()
				for b.Loop() {
					if code := Run(args, env); code != ExitOK {
						b.Fatalf("exit %d", code)
					}
				}
			})
		}
	}
}
