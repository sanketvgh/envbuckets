package cli

type jsonPlanResult struct {
	Bucket    string          `json:"bucket"`
	DryRun    bool            `json:"dry_run"`
	Changed   int             `json:"changed"`
	Unchanged int             `json:"unchanged"`
	Failed    int             `json:"failed"`
	Scopes    []jsonPlanScope `json:"scopes"`
}

type jsonPlanScope struct {
	Name   string `json:"name"`
	Action string `json:"action"`
	Reason string `json:"reason,omitempty"`
}
