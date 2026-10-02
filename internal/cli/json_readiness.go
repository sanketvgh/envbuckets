package cli

import (
	"errors"
	"os"
)

// Readiness is the stable machine-readable view of status and check.
// Issue codes are intended for decisions; output prose is for display only.
type jsonReadiness struct {
	Branch   string      `json:"branch"`
	Resolved bool        `json:"resolved"`
	Ready    bool        `json:"ready"`
	Target   *jsonTarget `json:"target,omitempty"`
	Scopes   []jsonScope `json:"scopes"`
}

type jsonTarget struct {
	Bucket   string `json:"bucket"`
	Source   string `json:"source"`
	Pattern  string `json:"pattern,omitempty"`
	Priority int    `json:"priority,omitempty"`
}

type jsonScope struct {
	Name               string   `json:"name"`
	Path               string   `json:"path"`
	DirectoryExists    bool     `json:"directory_exists"`
	LinkState          string   `json:"link_state"`
	ActiveBucket       string   `json:"active_bucket,omitempty"`
	ExpectedFileExists bool     `json:"expected_file_exists"`
	Ready              bool     `json:"ready"`
	Issues             []string `json:"issues"`
}

func readinessJSON(r readiness) jsonReadiness {
	view := jsonReadiness{Branch: r.branch, Resolved: r.resolved, Ready: r.healthy(), Scopes: make([]jsonScope, 0, len(r.scopes))}
	if r.resolved {
		view.Target = &jsonTarget{Bucket: r.target.bucket, Source: "rule", Pattern: r.target.via, Priority: r.target.priority}
		if r.target.linked {
			view.Target.Source, view.Target.Pattern = "pin", ""
		}
	}
	for _, state := range r.scopes {
		s := jsonScope{
			Name: state.scope.Name, Path: state.scope.Path,
			DirectoryExists:    state.directoryExists,
			ExpectedFileExists: state.expectedExists,
			Ready:              state.healthy() && r.resolved,
			Issues:             make([]string, 0),
		}
		switch {
		case !state.directoryExists:
			s.LinkState = "unknown"
			if errors.Is(state.directoryErr, os.ErrNotExist) {
				s.Issues = append(s.Issues, "scope_missing")
			} else {
				s.Issues = append(s.Issues, "unsafe_scope_path")
			}
		case state.linkErr != nil:
			s.LinkState = "unknown"
			s.Issues = append(s.Issues, "link_inspection_failed")
		default:
			switch state.link.kind {
			case linkMissing:
				s.LinkState = "missing"
				s.Issues = append(s.Issues, "link_missing")
			case linkReal:
				s.LinkState = "real_file"
				s.Issues = append(s.Issues, "real_env")
			case linkForeign:
				s.LinkState = "foreign_symlink"
				s.Issues = append(s.Issues, "foreign_link")
			case linkBucket:
				s.LinkState, s.ActiveBucket = "managed_symlink", state.link.bucket
				if state.link.dangling {
					s.Issues = append(s.Issues, "broken_link")
				}
			}
		}
		if r.resolved {
			if !state.expectedExists {
				s.Issues = append(s.Issues, "expected_file_missing")
			}
			if state.linkErr == nil && state.link.kind == linkBucket && state.link.bucket != r.target.bucket {
				s.Issues = append(s.Issues, "bucket_mismatch")
			}
		} else {
			s.Issues = append(s.Issues, "unresolved")
		}
		view.Scopes = append(view.Scopes, s)
	}
	return view
}
