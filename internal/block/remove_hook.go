package block

import (
	"bytes"
	"errors"
	"os"

	"github.com/sanketvgh/envbuckets/internal/fsx"
	"github.com/sanketvgh/envbuckets/internal/metadata"
)

// Unlike installation's version migration, uninstall accepts only these exact
// marker pairs. Unknown versions, mismatched pairs, and truncated ranges must
// never consume user content.
func removeHookBlocks(content []byte) ([]byte, bool, error) {
	var out []byte
	inside := false
	closing := ""
	changed := false
	for _, line := range bytes.SplitAfter(content, []byte("\n")) {
		text := string(bytes.TrimSuffix(bytes.TrimSuffix(line, []byte("\n")), []byte("\r")))
		switch text {
		case Begin, "# >>> envbuckets v1 >>>":
			if inside {
				return nil, false, errors.New("nested envbuckets hook markers; repair the block before uninstalling")
			}
			inside = true
			closing = End
			if text != Begin {
				closing = "# <<< envbuckets v1 <<<"
			}
		case End, "# <<< envbuckets v1 <<<":
			if !inside || text != closing {
				return nil, false, errors.New("mismatched envbuckets hook markers; repair the block before uninstalling")
			}
			inside = false
			changed = true
			continue
		}
		if !inside {
			out = append(out, line...)
		}
	}
	if inside {
		return nil, false, errors.New("incomplete envbuckets hook block; repair its markers before uninstalling")
	}
	return out, changed, nil
}

func removeHookRoot(root *os.Root, name string, dry bool) (HookResult, error) {
	if err := fsx.ValidateInternalPath(root, name); err != nil {
		return HookUnchanged, err
	}
	info, err := root.Lstat(name)
	if os.IsNotExist(err) {
		return HookUnchanged, nil
	}
	if err != nil {
		return HookUnchanged, err
	}
	if !info.Mode().IsRegular() {
		return HookUnchanged, errors.New("hook is not a regular file")
	}
	content, err := metadata.ReadRegular(root, name)
	if err != nil {
		return HookUnchanged, err
	}
	out, changed, err := removeHookBlocks(content)
	if err != nil || !changed {
		return HookUnchanged, err
	}
	result := HookWritten
	if onlyShebang(out) {
		result = HookDeleted
	}
	if dry {
		return result, nil
	}
	if err := fsx.ValidateInternalPath(root, name); err != nil {
		return HookUnchanged, err
	}
	if result == HookDeleted {
		err = root.Remove(name)
	} else {
		// Preserve mode bits and executable hooks; the atomic writer stages beside
		// the hook, syncs, and renames, leaving every byte outside our blocks intact.
		err = fsx.WriteFileAtomicRoot(root, name, out, info.Mode().Perm())
	}
	if err != nil {
		return HookUnchanged, err
	}
	return result, nil
}
