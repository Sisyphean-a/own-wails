package discovery

type DiscoveryItem struct {
	SourceRoot string   `json:"source_root"`
	Path       string   `json:"path"`
	Kind       string   `json:"kind"`
	Size       int64    `json:"size"`
	MTimeUTC   string   `json:"mtime_utc"`
	Attributes []string `json:"attributes"`
	LinkType   *string  `json:"link_type"`
	Target     *string  `json:"target"`
}

type ManifestRecord struct {
	SessionUID     *string  `json:"session_uid"`
	ThreadUID      *string  `json:"thread_uid"`
	StorageKind    string   `json:"storage_kind"`
	SourcePath     string   `json:"source_path"`
	CanonicalPath  string   `json:"canonical_path"`
	RealPath       string   `json:"real_path"`
	ReparseKind    string   `json:"reparse_kind"`
	CwdRaw         *string  `json:"cwd_raw"`
	CwdNorm        string   `json:"cwd_norm"`
	UpdatedAt      string   `json:"updated_at"`
	ContentHash    string   `json:"content_hash"`
	DuplicateGroup *string  `json:"duplicate_group"`
	Preferred      bool     `json:"preferred"`
	Evidence       []string `json:"evidence"`
}

type UnknownItem struct {
	SourceRoot string `json:"source_root"`
	Path       string `json:"path"`
	Kind       string `json:"kind"`
	Reason     string `json:"reason"`
}

type ScanSummary struct {
	RootCount    int
	ItemCount    int
	UnknownCount int
}

type ScanResult struct {
	RunID            string          `json:"run_id"`
	Roots            []string        `json:"roots"`
	DiscoveryPath    string          `json:"discovery_path"`
	ManifestPath     string          `json:"manifest_path"`
	UnknownItemsPath string          `json:"unknown_items_path"`
	Summary          ScanSummary     `json:"summary"`
	Items            []DiscoveryItem `json:"items"`
}
