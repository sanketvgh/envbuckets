package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

func decodeJSON(t *testing.T, res result) jsonResponse {
	t.Helper()
	if res.stderr != "" {
		t.Fatalf("JSON command wrote stderr: %q", res.stderr)
	}
	var response jsonResponse
	if err := json.Unmarshal([]byte(res.stdout), &response); err != nil {
		t.Fatalf("invalid JSON %q: %v", res.stdout, err)
	}
	if response.SchemaVersion != 1 || response.ExitCode != res.code || response.OK != (res.code == ExitOK) {
		t.Fatalf("inconsistent envelope: %+v, exit %d", response, res.code)
	}
	return response
}

func TestJSONReadinessAndFailure(t *testing.T) {
	r := newRepo(t)
	r.write(".envbuckets.toml", "schema = 1\n[[rules]]\npattern = \"*\"\nbucket = \"dev\"\n")
	r.write(".env.d/dev/.env", "secret-value-must-not-appear\n")

	status := r.run("status", "--json")
	got := decodeJSON(t, status)
	if got.Command != "status" || got.Error != nil {
		t.Fatalf("unexpected status envelope: %+v", got)
	}
	data, ok := got.Data.(map[string]any)
	if !ok || data["ready"] != false || data["resolved"] != true {
		t.Fatalf("unexpected status data: %#v", got.Data)
	}
	scopes, ok := data["scopes"].([]any)
	if !ok || len(scopes) != 1 {
		t.Fatalf("unexpected scopes: %#v", data["scopes"])
	}
	scope := scopes[0].(map[string]any)
	if scope["link_state"] != "missing" || scope["expected_file_exists"] != true {
		t.Fatalf("unexpected scope: %#v", scope)
	}
	if strings.Contains(status.stdout, "secret-value-must-not-appear") {
		t.Fatal("JSON output exposed env values")
	}

	check := r.run("--json", "check")
	got = decodeJSON(t, check)
	if check.code != ExitBlocked || got.Command != "check" || got.Error == nil || got.Error.Category != "blocked" {
		t.Fatalf("unexpected check envelope: %+v", got)
	}
	if strings.Contains(check.stdout, "secret-value-must-not-appear") {
		t.Fatal("JSON check exposed env values")
	}

	preview := r.run("apply", "--dry-run", "--json")
	got = decodeJSON(t, preview)
	plan := got.Data.(map[string]any)
	rows := plan["scopes"].([]any)
	if preview.code != ExitOK || plan["dry_run"] != true || rows[0].(map[string]any)["action"] != "would_change" || r.exists(".env") {
		t.Fatalf("unexpected non-writing plan: %+v", got)
	}
}

func TestJSONUsageAndHelp(t *testing.T) {
	r := newRepo(t)
	bad := decodeJSON(t, r.run("--json", "not-a-command"))
	if bad.ExitCode != ExitUsage || bad.Error == nil || bad.Error.Category != "usage" {
		t.Fatalf("unexpected usage envelope: %+v", bad)
	}
	help := decodeJSON(t, r.run("help", "--json"))
	if help.ExitCode != ExitOK || !strings.Contains(help.Output, "Global option: --json") {
		t.Fatalf("unexpected help envelope: %+v", help)
	}
}

func TestJSONNeverAcceptsPurgeConfirmation(t *testing.T) {
	r := newRepo(t)
	r.write(".envbuckets.toml", "schema = 1\n")
	r.write(".env.d/old/.env", "keep-me\n")
	res := r.runIn(r.root, "DELETE\n", "bucket", "rm", "old", "--purge", "--json")
	got := decodeJSON(t, res)
	if res.code != ExitBlocked || got.Error == nil || !r.exists(".env.d/old/.env") {
		t.Fatalf("JSON purge must be blocked without an interactive confirmation: %+v", got)
	}
}
