// Package pattern matches branch names using Git-style wildcard rules.
package pattern

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/git-lfs/wildmatch/v2"
)

var compiled sync.Map

// Validate reports a malformed pattern. wildmatch parses the full Git-style
// syntax; recover its parse panic and return it as a config validation error.
func Validate(pattern string) (err error) {
	if err := validateSyntax(pattern); err != nil {
		return err
	}
	_, err = compile(pattern)
	return err
}

// Match reports whether name matches pattern. Invalid patterns never match.
func Match(pattern, name string) bool {
	matcher, err := compile(pattern)
	return err == nil && matcher.Match(name)
}

func compile(pattern string) (matcher *wildmatch.Wildmatch, err error) {
	if err := validateSyntax(pattern); err != nil {
		return nil, err
	}
	if strings.HasSuffix(pattern, "/") {
		pattern += "**"
	}
	if cached, ok := compiled.Load(pattern); ok {
		return cached.(*wildmatch.Wildmatch), nil
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			matcher = nil
			err = fmt.Errorf("invalid pattern: %v", recovered)
		}
	}()
	matcher = wildmatch.NewWildmatch(pattern)
	actual, _ := compiled.LoadOrStore(pattern, matcher)
	return actual.(*wildmatch.Wildmatch), nil
}

func validateSyntax(pattern string) error {
	for i := 0; i < len(pattern); i++ {
		switch pattern[i] {
		case '\\':
			i++
			if i == len(pattern) {
				return errors.New("trailing escape")
			}
		case '[':
			end := classEnd(pattern, i)
			if end < 0 {
				return errors.New("unclosed or empty character class")
			}
			class := pattern[i+1 : end]
			class = strings.TrimPrefix(strings.TrimPrefix(class, "!"), "^")
			if class == "" {
				return errors.New("empty character class")
			}
			for j := 0; j < len(class); j++ {
				if class[j] == '[' && j+1 < len(class) && class[j+1] == ':' {
					closeAt := strings.Index(class[j+2:], ":]")
					if closeAt < 0 || !isPOSIXClass(class[j+2:j+2+closeAt]) {
						return errors.New("invalid POSIX character class")
					}
					j += closeAt + 3
				}
			}
			i = end
		}
	}
	return nil
}

func classEnd(pattern string, start int) int {
	i := start + 1
	if i < len(pattern) && (pattern[i] == '!' || pattern[i] == '^') {
		i++
	}
	if i < len(pattern) && pattern[i] == ']' {
		i++
	}
	for ; i < len(pattern); i++ {
		if pattern[i] == '\\' {
			i++
			continue
		}
		if pattern[i] == '[' && i+1 < len(pattern) && pattern[i+1] == ':' {
			if closeAt := strings.Index(pattern[i+2:], ":]"); closeAt >= 0 {
				i += closeAt + 3
				continue
			}
		}
		if pattern[i] == ']' {
			return i
		}
	}
	return -1
}

func isPOSIXClass(class string) bool {
	switch class {
	case "alnum", "alpha", "blank", "cntrl", "digit", "graph", "lower", "print", "punct", "space", "upper", "xdigit":
		return true
	default:
		return false
	}
}
