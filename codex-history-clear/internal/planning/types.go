package planning

type Summary struct {
	GroupCount     int `json:"group_count"`
	CandidateCount int `json:"candidate_count"`
	ReviewCount    int `json:"review_count"`
	PlannedCount   int `json:"planned_count"`
}

type GroupCandidate struct {
	SessionUID     *string  `json:"session_uid"`
	ThreadUID      *string  `json:"thread_uid"`
	StorageKind    string   `json:"storage_kind"`
	SourcePath     string   `json:"source_path"`
	CanonicalPath  string   `json:"canonical_path"`
	RealPath       string   `json:"real_path"`
	UpdatedAt      string   `json:"updated_at"`
	Preferred      bool     `json:"preferred"`
	Relation       string   `json:"relation"`
	Action         string   `json:"action"`
	ReasonCode     string   `json:"reason_code"`
	Reason         string   `json:"reason"`
	RequiresCLI    bool     `json:"requires_cli"`
	ReviewNeeded   bool     `json:"review_needed"`
	QuarantinePath *string  `json:"quarantine_path,omitempty"`
	Warnings       []string `json:"warnings"`
}

type DuplicateGroup struct {
	DuplicateGroup string           `json:"duplicate_group"`
	PreferredPath  string           `json:"preferred_path"`
	ReviewNeeded   bool             `json:"review_needed"`
	Warning        string           `json:"warning,omitempty"`
	Candidates     []GroupCandidate `json:"candidates"`
}

type DeletePlanItem struct {
	DuplicateGroup string   `json:"duplicate_group"`
	SessionUID     *string  `json:"session_uid"`
	SourcePath     string   `json:"source_path"`
	PreferredPath  string   `json:"preferred_path"`
	Action         string   `json:"action"`
	ReasonCode     string   `json:"reason_code"`
	Reason         string   `json:"reason"`
	RequiresCLI    bool     `json:"requires_cli"`
	ReviewNeeded   bool     `json:"review_needed"`
	QuarantinePath *string  `json:"quarantine_path,omitempty"`
	Warnings       []string `json:"warnings"`
}

type DeletePlanDocument struct {
	RunID    string           `json:"run_id"`
	Approved bool             `json:"approved"`
	Items    []DeletePlanItem `json:"items"`
	Warnings []string         `json:"warnings"`
}

type Result struct {
	RunID               string           `json:"run_id"`
	ManifestPath        string           `json:"manifest_path"`
	DuplicateGroupsPath string           `json:"duplicate_groups_path"`
	DeletePlanPath      string           `json:"delete_plan_path"`
	Summary             Summary          `json:"summary"`
	Groups              []DuplicateGroup `json:"groups"`
	Items               []DeletePlanItem `json:"items"`
	Warnings            []string         `json:"warnings"`
}
