package planning

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"codex-history-manager/internal/discovery"
)

type annotatedRecord struct {
	record           discovery.ManifestRecord
	groupKey         string
	sourceNorm       string
	canonicalNorm    string
	realNorm         string
	baseName         string
	updatedAt        time.Time
	updatedAtOK      bool
	cliVisible       bool
	canonicalPrimary bool
	archived         bool
	directPath       bool
}

func buildDuplicateGroups(records []discovery.ManifestRecord, runID string) []DuplicateGroup {
	buckets := bucketRecords(annotateRecords(records))
	keys := sortedGroupKeys(buckets)
	groups := make([]DuplicateGroup, 0, len(keys))
	for index, key := range keys {
		groups = append(groups, buildGroup(index+1, buckets[key], runID))
	}
	return groups
}

func annotateRecords(records []discovery.ManifestRecord) []annotatedRecord {
	annotated := make([]annotatedRecord, 0, len(records))
	for _, record := range records {
		annotated = append(annotated, annotateRecord(record))
	}
	return annotated
}

func annotateRecord(record discovery.ManifestRecord) annotatedRecord {
	updatedAt, updatedAtOK := parseUpdatedAt(record.UpdatedAt)
	sourceNorm := normalizePath(record.SourcePath)
	canonicalNorm := normalizePath(record.CanonicalPath)
	realNorm := normalizePath(record.RealPath)
	return annotatedRecord{
		record:           record,
		groupKey:         groupKeyFor(record),
		sourceNorm:       sourceNorm,
		canonicalNorm:    canonicalNorm,
		realNorm:         realNorm,
		baseName:         pathBase(record),
		updatedAt:        updatedAt,
		updatedAtOK:      updatedAtOK,
		cliVisible:       hasEvidence(record, "cli-visible"),
		canonicalPrimary: sourceNorm != "" && sourceNorm == canonicalNorm,
		archived:         isArchivedRecord(record),
		directPath:       strings.EqualFold(record.ReparseKind, "none"),
	}
}

func bucketRecords(records []annotatedRecord) map[string][]annotatedRecord {
	buckets := make(map[string][]annotatedRecord)
	for _, record := range records {
		buckets[record.groupKey] = append(buckets[record.groupKey], record)
	}
	for key, members := range buckets {
		if len(members) < 2 {
			delete(buckets, key)
		}
	}
	return buckets
}

func sortedGroupKeys(buckets map[string][]annotatedRecord) []string {
	keys := make([]string, 0, len(buckets))
	for key := range buckets {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func buildGroup(index int, members []annotatedRecord, runID string) DuplicateGroup {
	sortMembers(members)
	preferred, decision := choosePreferred(members)
	groupID := fmt.Sprintf("dup-%06d", index)
	candidates := decorateCandidates(members, preferred, decision, groupID, runID)
	group := DuplicateGroup{
		DuplicateGroup: groupID,
		PreferredPath:  preferred.record.SourcePath,
		ReviewNeeded:   decision.reviewNeeded,
		Candidates:     candidates,
	}
	if decision.warning != "" {
		group.Warning = decision.warning
	}
	return group
}

func decorateCandidates(
	members []annotatedRecord,
	preferred annotatedRecord,
	decision selectionDecision,
	groupID string,
	runID string,
) []GroupCandidate {
	candidates := make([]GroupCandidate, 0, len(members))
	for _, member := range members {
		action := actionForCandidate(member, preferred, decision.reviewNeeded, groupID, runID)
		candidates = append(candidates, GroupCandidate{
			SessionUID:     member.record.SessionUID,
			ThreadUID:      member.record.ThreadUID,
			StorageKind:    member.record.StorageKind,
			SourcePath:     member.record.SourcePath,
			CanonicalPath:  member.record.CanonicalPath,
			RealPath:       member.record.RealPath,
			UpdatedAt:      member.record.UpdatedAt,
			Preferred:      member.sourceNorm == preferred.sourceNorm,
			Relation:       relationFor(member, preferred),
			Action:         action.action,
			ReasonCode:     action.reasonCode,
			Reason:         action.reason,
			RequiresCLI:    action.requiresCLI,
			ReviewNeeded:   decision.reviewNeeded,
			QuarantinePath: action.quarantinePath,
			Warnings:       cloneStrings(action.warnings),
		})
	}
	return candidates
}

func actionForCandidate(
	member annotatedRecord,
	preferred annotatedRecord,
	reviewNeeded bool,
	groupID string,
	runID string,
) actionDecision {
	if member.sourceNorm == preferred.sourceNorm {
		return actionDecision{
			action:     "keep",
			reasonCode: preferredReasonCode(preferred, reviewNeeded),
			reason:     preferredReasonText(preferred, reviewNeeded),
		}
	}
	if reviewNeeded {
		return actionDecision{
			action:     "keep",
			reasonCode: "review-needed",
			reason:     "候选缺少足够证据，暂不建议 destructive 动作",
			warnings:   []string{"当前重复组需要人工复核"},
		}
	}
	if member.realNorm == preferred.realNorm {
		return actionDecision{
			action:      "repair_index",
			reasonCode:  "path-alias-nonpreferred",
			reason:      "与保留本指向同一 real_path，建议只修复入口而不删文件",
			requiresCLI: true,
		}
	}
	return actionDecision{
		action:         "quarantine",
		reasonCode:     "duplicate-copy-nonpreferred",
		reason:         "存在更高置信度保留本，当前副本进入隔离计划",
		quarantinePath: quarantinePathFor(runID, groupID, member.record.SourcePath),
	}
}

func relationFor(member annotatedRecord, preferred annotatedRecord) string {
	if member.sourceNorm == preferred.sourceNorm {
		return "preferred"
	}
	if member.realNorm == preferred.realNorm {
		return "path-alias"
	}
	return "physical-copy"
}

func quarantinePathFor(runID string, groupID string, sourcePath string) *string {
	baseName := filepath.Base(sourcePath)
	path := filepath.Join(".codex", ".trash", runID, groupID, baseName)
	return &path
}

func groupKeyFor(record discovery.ManifestRecord) string {
	parts := identityParts(record)
	if len(parts) > 0 {
		return strings.Join(parts, "|")
	}
	keyParts := []string{record.StorageKind, pathBase(record), record.ContentHash}
	if record.StorageKind == "codex_history_jsonl" {
		keyParts = []string{record.StorageKind, record.ContentHash}
	}
	return strings.Join(keyParts, "|")
}

func identityParts(record discovery.ManifestRecord) []string {
	parts := []string{}
	if record.SessionUID != nil && *record.SessionUID != "" {
		parts = append(parts, "session="+*record.SessionUID)
	}
	if record.ThreadUID != nil && *record.ThreadUID != "" {
		parts = append(parts, "thread="+*record.ThreadUID)
	}
	if record.CwdNorm != "" {
		parts = append(parts, "cwd="+strings.ToLower(record.CwdNorm))
	}
	return parts
}

func pathBase(record discovery.ManifestRecord) string {
	path := record.RealPath
	if path == "" {
		path = record.SourcePath
	}
	return strings.ToLower(filepath.Base(path))
}

func normalizePath(path string) string {
	cleaned := filepath.Clean(path)
	return strings.ToLower(strings.ReplaceAll(cleaned, "/", `\`))
}

func parseUpdatedAt(value string) (time.Time, bool) {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, false
	}
	return parsed, true
}

func hasEvidence(record discovery.ManifestRecord, token string) bool {
	for _, evidence := range record.Evidence {
		if evidence == token {
			return true
		}
	}
	return false
}

func isArchivedRecord(record discovery.ManifestRecord) bool {
	if hasEvidence(record, "archived-rollout-file") {
		return true
	}
	lowerPath := strings.ToLower(record.SourcePath)
	return strings.Contains(lowerPath, `\archive\`) ||
		strings.Contains(lowerPath, `\archives\`) ||
		strings.Contains(lowerPath, `\archived\`) ||
		strings.Contains(lowerPath, `\archived_sessions\`)
}
