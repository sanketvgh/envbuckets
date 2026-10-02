package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// The envelope is versioned independently of the human-readable output.
// Fields in data are defined for readiness and bulk link plans.
type jsonResponse struct {
	SchemaVersion int        `json:"schema_version"`
	Command       string     `json:"command"`
	OK            bool       `json:"ok"`
	ExitCode      int        `json:"exit_code"`
	Data          any        `json:"data,omitempty"`
	Error         *jsonError `json:"error,omitempty"`
	Output        string     `json:"output,omitempty"`
	Warnings      string     `json:"warnings,omitempty"`
}

type jsonError struct {
	Category string `json:"category"`
	Message  string `json:"message"`
	Next     string `json:"next,omitempty"`
}

func extractJSONFlag(args []string) ([]string, bool) {
	filtered := make([]string, 0, len(args))
	jsonMode := false
	positionalOnly := false
	for _, arg := range args {
		if arg == "--" {
			positionalOnly = true
		}
		if arg == "--json" && !positionalOnly {
			jsonMode = true
			continue
		}
		filtered = append(filtered, arg)
	}
	return filtered, jsonMode
}

func runJSON(args []string, env Env) int {
	var out, warnings bytes.Buffer
	var data any
	var failure *exitError
	originalOut, originalErr := env.Stdout, env.Stderr
	env.Stdout, env.Stderr = &out, &warnings
	// JSON mode never waits for an interactive answer. Commands that require
	// typed confirmation report a blocked operation instead.
	env.Stdin = strings.NewReader("")
	env.jsonData, env.jsonError = &data, &failure
	code := runText(args, env)
	command := "help"
	if len(args) > 0 {
		command = args[0]
		if len(args) > 1 && (command == "bucket" || command == "map" || command == "scope") {
			command += " " + args[1]
		}
	}
	response := jsonResponse{
		SchemaVersion: 1,
		Command:       command,
		OK:            code == ExitOK,
		ExitCode:      code,
		Data:          data,
		Output:        out.String(),
	}
	if failure != nil {
		response.Error = &jsonError{Category: className(failure.code), Message: failure.msg, Next: failure.next}
		reportText := fmt.Sprintf("envbuckets: %s: %s\n", className(failure.code), failure.msg)
		if failure.next != "" {
			reportText += fmt.Sprintf("  next: %s\n", failure.next)
		}
		response.Warnings = strings.TrimSuffix(warnings.String(), reportText)
	} else {
		response.Warnings = warnings.String()
	}
	encoder := json.NewEncoder(originalOut)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(response); err != nil {
		fmt.Fprintf(originalErr, "envbuckets: environment: cannot write JSON response: %v\n", err)
		return ExitEnv
	}
	return code
}
