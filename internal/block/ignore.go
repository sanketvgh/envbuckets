package block

import (
	"bytes"
	"errors"
	"os"
	"strings"

	"github.com/sanketvgh/envbuckets/internal/fsx"
	"github.com/sanketvgh/envbuckets/internal/gitx"
)

// IgnorePlan holds a preflighted edit of the marker-guarded .gitignore block.
type IgnorePlan struct {
	Content []byte
	Changed bool
	Mode    os.FileMode
}

// PlanIgnore refuses symlinks before reading .gitignore, replaces all legacy
// blocks with one block, and omits paths already ignored by Git.
func PlanIgnore(repo *fsx.Repo, paths []string) (IgnorePlan, error) {
	p := IgnorePlan{Mode: 0o644}
	if err := fsx.ValidateInternalPath(repo.Root, ".gitignore"); err != nil {
		return p, err
	}
	if info, err := repo.Root.Lstat(".gitignore"); err == nil {
		if !info.Mode().IsRegular() {
			return p, errors.New(".gitignore is not a regular file")
		}
		p.Mode = info.Mode().Perm()
	} else if !os.IsNotExist(err) {
		return p, err
	}
	existing, err := repo.Root.ReadFile(".gitignore")
	if err != nil && !os.IsNotExist(err) {
		return p, err
	}
	content := existing
	var lines []string
	for {
		body := Body(content)
		for i := range body {
			body[i] = strings.TrimSuffix(body[i], "\r")
		}
		lines = MergeLines(lines, body)
		next, removed := Remove(content)
		if !removed {
			break
		}
		content = next
	}
	for line := range strings.SplitSeq(string(content), "\n") {
		if strings.HasPrefix(line, beginPrefix) || strings.HasPrefix(line, endPrefix) {
			return p, errors.New("incomplete envbuckets ignore block; repair its markers first")
		}
	}
	for _, name := range append([]string{".env.d/"}, paths...) {
		ignored, err := gitx.Ignored(repo.Path, name)
		if err != nil {
			return p, err
		}
		if !ignored {
			lines = MergeLines(lines, []string{ignoreLiteral(name)})
		}
	}
	p.Content, _ = Upsert(content, strings.Join(lines, "\n"))
	p.Changed = !bytes.Equal(existing, p.Content)
	return p, nil
}

// Apply writes only when needed, preserving the existing file's permissions.
func (p IgnorePlan) Apply(repo *fsx.Repo) error {
	if !p.Changed {
		return nil
	}
	return fsx.WriteFileAtomicRoot(repo.Root, ".gitignore", p.Content, p.Mode)
}

func ignoreLiteral(name string) string {
	var out strings.Builder
	// Slash-containing patterns are already anchored. Root filenames need an
	// anchor for leading #/! and spaces so Git treats them as literal paths.
	if !strings.Contains(name, "/") && strings.ContainsAny(name, "#! ") {
		out.WriteByte('/')
	}
	for _, ch := range name {
		if strings.ContainsRune("\\*?[] ", ch) {
			out.WriteByte('\\')
		}
		out.WriteRune(ch)
	}
	return out.String()
}
