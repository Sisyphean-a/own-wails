package history

type ListRequest struct {
	Limit    int    `json:"limit"`
	CWD      string `json:"cwd"`
	Grep     string `json:"grep"`
	Archived bool   `json:"archived"`
	All      bool   `json:"all"`
}

type ThreadSummary struct {
	ID               string   `json:"id"`
	Title            string   `json:"title"`
	SourceTitle      string   `json:"sourceTitle"`
	Source           string   `json:"source"`
	ModelProvider    string   `json:"modelProvider"`
	ThreadSource     string   `json:"threadSource"`
	RolloutPath      string   `json:"rolloutPath"`
	RolloutPaths     []string `json:"rolloutPaths"`
	IsClone          bool     `json:"isClone"`
	ClonedFrom       string   `json:"clonedFrom"`
	OriginalProvider string   `json:"originalProvider"`
	Registered       bool     `json:"registered"`
	CreatedAt        string   `json:"createdAt"`
	UpdatedAt        string   `json:"updatedAt"`
	CWD              string   `json:"cwd"`
	Archived         bool     `json:"archived"`
	SizeBytes        int64    `json:"sizeBytes"`
	FirstUserMessage string   `json:"firstUserMessage"`
	Preview          string   `json:"preview"`
}

type ListSummary struct {
	Count        int  `json:"count"`
	Limit        int  `json:"limit"`
	HasMore      bool `json:"hasMore"`
	WarningCount int  `json:"warningCount"`
}

type ScanWarning struct {
	Path    string `json:"path"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ListResult struct {
	CodexHome string          `json:"codexHome"`
	Summary   ListSummary     `json:"summary"`
	Items     []ThreadSummary `json:"items"`
	Warnings  []ScanWarning   `json:"warnings"`
}

type BuildPlanRequest struct {
	ThreadIDs []string `json:"threadIds"`
}

type PlanStore struct {
	Store  string `json:"store"`
	Path   string `json:"path"`
	Action string `json:"action"`
	Detail string `json:"detail"`
	Count  int64  `json:"count"`
	Exists bool   `json:"exists"`
}

type PlanTarget struct {
	Thread   ThreadSummary `json:"thread"`
	Stores   []PlanStore   `json:"stores"`
	Warnings []string      `json:"warnings"`
}

type PlanSummary struct {
	TargetCount  int `json:"targetCount"`
	StoreCount   int `json:"storeCount"`
	WarningCount int `json:"warningCount"`
}

type planDocument struct {
	RunID     string       `json:"run_id"`
	CodexHome string       `json:"codex_home"`
	Approved  bool         `json:"approved"`
	Summary   PlanSummary  `json:"summary"`
	Targets   []PlanTarget `json:"targets"`
	Warnings  []string     `json:"warnings"`
}

type PlanResult struct {
	RunID     string       `json:"runId"`
	CodexHome string       `json:"codexHome"`
	PlanPath  string       `json:"planPath"`
	Summary   PlanSummary  `json:"summary"`
	Targets   []PlanTarget `json:"targets"`
	Warnings  []string     `json:"warnings"`
}

type ApproveRequest struct {
	PlanPath string `json:"planPath"`
}

type ApproveResult struct {
	RunID            string       `json:"runId"`
	PlanPath         string       `json:"planPath"`
	ApprovedPlanPath string       `json:"approvedPlanPath"`
	Summary          PlanSummary  `json:"summary"`
	Targets          []PlanTarget `json:"targets"`
	Warnings         []string     `json:"warnings"`
}

type ExecuteRequest struct {
	PlanPath   string `json:"planPath"`
	Confirmed  bool   `json:"confirmed"`
	BackupOnly bool   `json:"backupOnly"`
	SkipBackup bool   `json:"skipBackup"`
}

type BackupArtifact struct {
	OriginalPath string `json:"originalPath"`
	BackupPath   string `json:"backupPath"`
}

type RollbackEntry struct {
	OriginalPath string `json:"originalPath"`
	BackupPath   string `json:"backupPath"`
	Restored     bool   `json:"restored"`
}

type rollbackJournal struct {
	RunID    string          `json:"run_id"`
	PlanPath string          `json:"plan_path"`
	Entries  []RollbackEntry `json:"entries"`
}

type MutationResult struct {
	Store       string `json:"store"`
	Action      string `json:"action"`
	Path        string `json:"path"`
	ChangedRows int64  `json:"changedRows"`
	Changed     bool   `json:"changed"`
}

type VerificationFinding struct {
	Store  string `json:"store"`
	Path   string `json:"path"`
	Detail string `json:"detail"`
}

type VerificationResult struct {
	Status              string                `json:"status"`
	Summary             string                `json:"summary"`
	Success             bool                  `json:"success"`
	RemainingReferences []VerificationFinding `json:"remainingReferences"`
}

type JobEvent struct {
	Phase        string `json:"phase"`
	ItemIndex    int    `json:"itemIndex"`
	ItemTotal    int    `json:"itemTotal"`
	Level        string `json:"level"`
	Message      string `json:"message"`
	ArtifactPath string `json:"artifactPath"`
}

type execResultDocument struct {
	RunID               string             `json:"run_id"`
	PlanPath            string             `json:"plan_path"`
	ApprovedPlanPath    string             `json:"approved_plan_path"`
	RollbackJournalPath string             `json:"rollback_journal_path"`
	ManifestAfterPath   string             `json:"manifest_after_path"`
	Backups             []BackupArtifact   `json:"backups"`
	Mutations           []MutationResult   `json:"mutations"`
	Events              []JobEvent         `json:"events"`
	Verification        VerificationResult `json:"verification"`
}

type ExecuteResult struct {
	RunID               string             `json:"runId"`
	Mode                string             `json:"mode"`
	PlanPath            string             `json:"planPath"`
	ApprovedPlanPath    string             `json:"approvedPlanPath"`
	RollbackJournalPath string             `json:"rollbackJournalPath"`
	ExecResultPath      string             `json:"execResultPath"`
	ManifestAfterPath   string             `json:"manifestAfterPath"`
	Backups             []BackupArtifact   `json:"backups"`
	Mutations           []MutationResult   `json:"mutations"`
	Events              []JobEvent         `json:"events"`
	Verification        VerificationResult `json:"verification"`
}

type RollbackRequest struct {
	JournalPath string `json:"journalPath"`
}

type RollbackResult struct {
	RunID         string          `json:"runId"`
	JournalPath   string          `json:"journalPath"`
	RestoredCount int             `json:"restoredCount"`
	Entries       []RollbackEntry `json:"entries"`
	Events        []JobEvent      `json:"events"`
}

type EvidencePackRequest struct {
	RunID               string `json:"runId"`
	DiscoveryPath       string `json:"discoveryPath"`
	ManifestBeforePath  string `json:"manifestBeforePath"`
	DuplicateGroupsPath string `json:"duplicateGroupsPath"`
	DeletePlanPath      string `json:"deletePlanPath"`
	ApprovedPlanPath    string `json:"approvedPlanPath"`
	RollbackJournalPath string `json:"rollbackJournalPath"`
	ExecResultPath      string `json:"execResultPath"`
	ManifestAfterPath   string `json:"manifestAfterPath"`
	ArchitecturePath    string `json:"architecturePath"`
	ContextPath         string `json:"contextPath"`
	HistoryPath         string `json:"historyPath"`
}

type EvidencePackArtifact struct {
	Label string `json:"label"`
	Path  string `json:"path"`
}

type EvidencePackResult struct {
	RunID            string                 `json:"runId"`
	EvidencePackPath string                 `json:"evidencePackPath"`
	Artifacts        []EvidencePackArtifact `json:"artifacts"`
}
