package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/sanketvgh/envbuckets/internal/gitx"
)

type scopeReadiness struct {
	scope           scope
	directoryExists bool
	directoryErr    error
	link            linkState
	linkErr         error
	expectedExists  bool
	problems        []string
}

func (r scopeReadiness) healthy() bool { return len(r.problems) == 0 }

type readiness struct {
	branch   string
	target   target
	resolved bool
	scopes   []scopeReadiness
}

func (r readiness) healthy() bool {
	if !r.resolved {
		return false
	}
	for _, s := range r.scopes {
		if !s.healthy() {
			return false
		}
	}
	return true
}

func (p *project) evaluateCurrent(selected []scope) (readiness, error) {
	branch, err := gitx.Branch(p.root)
	if err != nil {
		return readiness{}, envErr("cannot resolve the current branch: %v", err).then("check git status")
	}
	r := readiness{branch: branch}
	if branch != "" {
		r.target, r.resolved, err = p.target(branch)
		if err != nil {
			return readiness{}, envErr("cannot read the branch link: %v", err).then("envbuckets unlink, or check git config --local")
		}
	}
	for _, s := range selected {
		r.scopes = append(r.scopes, evaluateScope(s, r.target.bucket, r.resolved))
	}
	return r, nil
}

func evaluateScope(s scope, expected string, resolved bool) scopeReadiness {
	dirErr := s.checkScopeDir()
	r := scopeReadiness{scope: s, directoryExists: dirErr == nil, directoryErr: dirErr}
	if !r.directoryExists {
		if errors.Is(dirErr, os.ErrNotExist) {
			r.problems = append(r.problems, "scope directory missing")
		} else {
			r.problems = append(r.problems, fmt.Sprintf("unsafe scope path: %v", dirErr))
		}
		return r
	}
	r.link, r.linkErr = s.linkState()
	if r.linkErr != nil {
		r.problems = append(r.problems, fmt.Sprintf("cannot inspect .env: %v", r.linkErr))
	} else {
		switch r.link.kind {
		case linkMissing:
			r.problems = append(r.problems, "managed .env link missing")
		case linkReal:
			r.problems = append(r.problems, "real .env is unmanaged")
		case linkForeign:
			r.problems = append(r.problems, "foreign .env symlink is unmanaged")
		case linkBucket:
			if r.link.dangling {
				r.problems = append(r.problems, "active .env link is broken")
			}
		}
	}
	if resolved {
		r.expectedExists = s.bucketFileExists(expected)
		if !r.expectedExists {
			r.problems = append(r.problems, "expected bucket file missing")
		}
		if r.linkErr == nil && r.link.kind == linkBucket && r.link.bucket != expected {
			r.problems = append(r.problems, "active and expected buckets differ")
		}
	}
	return r
}
