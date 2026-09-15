// Package pattern matches branch names against rule patterns in which *
// matches any run of characters, including /.
package pattern

import "strings"

// Match reports whether name matches pattern.
func Match(pattern, name string) bool {
	segments := strings.Split(pattern, "*")
	if len(segments) == 1 {
		return pattern == name
	}
	first, last := segments[0], segments[len(segments)-1]
	if !strings.HasPrefix(name, first) {
		return false
	}
	rest := name[len(first):]
	if !strings.HasSuffix(rest, last) {
		return false
	}
	rest = rest[:len(rest)-len(last)]
	for _, seg := range segments[1 : len(segments)-1] {
		if seg == "" {
			continue
		}
		idx := strings.Index(rest, seg)
		if idx < 0 {
			return false
		}
		rest = rest[idx+len(seg):]
	}
	return true
}
