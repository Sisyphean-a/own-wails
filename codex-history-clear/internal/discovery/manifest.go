package discovery

func buildManifest(items []DiscoveryItem) ([]ManifestRecord, error) {
	context, err := loadManifestContext(items)
	if err != nil {
		return nil, err
	}
	records := make([]ManifestRecord, 0, len(items))
	for _, item := range items {
		record, ok, err := buildManifestRecord(item, context)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		records = append(records, record)
	}
	return records, nil
}

func buildManifestRecord(item DiscoveryItem, context manifestContext) (ManifestRecord, bool, error) {
	storageKind, ok := storageKindFor(item.Kind)
	if !ok {
		return ManifestRecord{}, false, nil
	}
	contentHash, err := contentHashFor(item.Path)
	if err != nil {
		return ManifestRecord{}, false, err
	}
	sourcePath := cleanWindowsPath(item.Path)
	realPath := cleanWindowsPath(realPathFor(item))
	metadata, err := buildRecordMetadata(item, context, sourcePath, realPath)
	if err != nil {
		return ManifestRecord{}, false, err
	}
	return ManifestRecord{
		SessionUID:    metadata.SessionUID,
		ThreadUID:     metadata.ThreadUID,
		StorageKind:   storageKind,
		SourcePath:    sourcePath,
		CanonicalPath: metadata.CanonicalPath,
		RealPath:      realPath,
		ReparseKind:   reparseKindFor(item),
		CwdRaw:        metadata.CwdRaw,
		CwdNorm:       metadata.CwdNorm,
		UpdatedAt:     item.MTimeUTC,
		ContentHash:   contentHash,
		Preferred:     false,
		Evidence:      mergeEvidence(evidenceFor(item.Kind), metadata.Evidence),
	}, true, nil
}

func storageKindFor(kind string) (string, bool) {
	switch kind {
	case "history_jsonl":
		return "codex_history_jsonl", true
	case "rollout_jsonl", "archived_rollout_jsonl":
		return "codex_rollout_jsonl", true
	case "state_sqlite", "logs_sqlite":
		return "codex_sqlite", true
	default:
		return "", false
	}
}

func realPathFor(item DiscoveryItem) string {
	if item.Target != nil {
		return *item.Target
	}
	return item.Path
}

func reparseKindFor(item DiscoveryItem) string {
	if item.LinkType == nil {
		return "none"
	}
	return *item.LinkType
}

func evidenceFor(kind string) []string {
	switch kind {
	case "history_jsonl":
		return []string{"history-file"}
	case "rollout_jsonl":
		return []string{"rollout-file"}
	case "archived_rollout_jsonl":
		return []string{"archived-rollout-file"}
	default:
		return []string{"sqlite-file"}
	}
}
