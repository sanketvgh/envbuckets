package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/sanketvgh/envbuckets/internal/block"
	"github.com/sanketvgh/envbuckets/internal/fsx"
)

func ignoreLines(scopes []scope) []string {
	lines := make([]string, 0, 3+2*len(scopes))
	lines = append(lines, envFile, ".env.local", bucketsDir+"/")
	for _, s := range scopes {
		lines = append(lines, scopeIgnoreLines(s)...)
	}
	return lines
}

func scopeIgnoreLines(s scope) []string {
	if s.Path == "." {
		return nil
	}
	return []string{s.Path + "/" + envFile, s.Path + "/" + bucketsDir + "/"}
}

func gitignorePath(root string) string {
	return filepath.Join(root, ".gitignore")
}

func readGitignore(root string) ([]byte, error) {
	data, err := os.ReadFile(gitignorePath(root))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	return data, err
}

func ensureIgnored(root string, required []string) (bool, error) {
	data, err := readGitignore(root)
	if err != nil {
		return false, err
	}
	lines := block.MergeLines(block.Body(data), required)
	out, changed := block.Upsert(data, strings.Join(lines, "\n"))
	if !changed {
		return false, nil
	}
	return true, fsx.WriteFileAtomic(gitignorePath(root), out, 0o644)
}

func dropIgnored(root string, drop []string) (bool, error) {
	data, err := readGitignore(root)
	if err != nil {
		return false, err
	}
	body := block.Body(data)
	if body == nil {
		return false, nil
	}
	lines := block.DropLines(body, drop)
	var out []byte
	var changed bool
	if len(lines) == 0 {
		out, changed = block.Remove(data)
	} else {
		out, changed = block.Upsert(data, strings.Join(lines, "\n"))
	}
	if !changed {
		return false, nil
	}
	return true, fsx.WriteFileAtomic(gitignorePath(root), out, 0o644)
}

func removeIgnoreBlock(root string) (bool, error) {
	data, err := readGitignore(root)
	if err != nil {
		return false, err
	}
	out, found := block.Remove(data)
	if !found {
		return false, nil
	}
	if strings.TrimSpace(string(out)) == "" {
		return true, os.Remove(gitignorePath(root))
	}
	return true, fsx.WriteFileAtomic(gitignorePath(root), out, 0o644)
}
