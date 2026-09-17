package planning

import (
	"slices"
	"strings"
)

type selectionDecision struct {
	reviewNeeded bool
	reasonCode   string
	reason       string
	warning      string
}

type actionDecision struct {
	action         string
	reasonCode     string
	reason         string
	requiresCLI    bool
	quarantinePath *string
	warnings       []string
}

func sortMembers(members []annotatedRecord) {
	slices.SortStableFunc(members, func(left annotatedRecord, right annotatedRecord) int {
		return candidateComparison(left, right)
	})
}

func choosePreferred(members []annotatedRecord) (annotatedRecord, selectionDecision) {
	preferred := members[0]
	if len(members) == 1 {
		return preferred, selectionDecision{
			reasonCode: "single-candidate-group",
			reason:     "重复组内只有一条候选，保留当前记录",
		}
	}
	reasonCode, reason := reasonForSelection(preferred, members[1])
	if reasonCode != "" {
		return preferred, selectionDecision{
			reasonCode: reasonCode,
			reason:     reason,
		}
	}
	return preferred, selectionDecision{
		reviewNeeded: true,
		reasonCode:   "review-needed",
		reason:       "候选缺少足够证据，需人工确认保留本",
		warning:      "当前重复组没有稳定的保留本判定依据，已降级为 review-needed。",
	}
}

func candidateComparison(left annotatedRecord, right annotatedRecord) int {
	if diff := compareBool(left.cliVisible, right.cliVisible); diff != 0 {
		return diff
	}
	if diff := compareBool(left.canonicalPrimary, right.canonicalPrimary); diff != 0 {
		return diff
	}
	if diff := compareBool(!left.archived, !right.archived); diff != 0 {
		return diff
	}
	if diff := compareBool(left.directPath, right.directPath); diff != 0 {
		return diff
	}
	if diff := compareTimestamp(left, right); diff != 0 {
		return diff
	}
	return strings.Compare(left.sourceNorm, right.sourceNorm)
}

func compareBool(left bool, right bool) int {
	switch {
	case left == right:
		return 0
	case left:
		return -1
	default:
		return 1
	}
}

func compareTimestamp(left annotatedRecord, right annotatedRecord) int {
	if diff := compareBool(left.updatedAtOK, right.updatedAtOK); diff != 0 {
		return diff
	}
	if !left.updatedAtOK || left.updatedAt.Equal(right.updatedAt) {
		return 0
	}
	if left.updatedAt.After(right.updatedAt) {
		return -1
	}
	return 1
}

func reasonForSelection(preferred annotatedRecord, challenger annotatedRecord) (string, string) {
	if preferred.cliVisible != challenger.cliVisible && preferred.cliVisible {
		return "cli-visible-preferred", "保留 CLI 可见的候选作为权威保留本"
	}
	if preferred.canonicalPrimary != challenger.canonicalPrimary && preferred.canonicalPrimary {
		return "canonical-path-preferred", "保留 canonical_path 与 source_path 一致的候选"
	}
	if preferred.archived != challenger.archived && !preferred.archived {
		return "live-record-preferred", "优先保留非归档候选，避免把 archived 副本当作主记录"
	}
	if preferred.directPath != challenger.directPath && preferred.directPath {
		return "direct-path-preferred", "优先保留非 reparse path 的直接入口"
	}
	if preferred.updatedAtOK != challenger.updatedAtOK && preferred.updatedAtOK {
		return "timestamped-record-preferred", "优先保留带有效更新时间的候选"
	}
	if preferred.updatedAtOK && challenger.updatedAtOK && preferred.updatedAt.After(challenger.updatedAt) {
		return "newest-record-preferred", "优先保留更新时间更新的候选"
	}
	return "", ""
}

func preferredReasonCode(preferred annotatedRecord, reviewNeeded bool) string {
	if reviewNeeded {
		return "review-needed"
	}
	if preferred.cliVisible {
		return "cli-visible-preferred"
	}
	if preferred.canonicalPrimary {
		return "canonical-path-preferred"
	}
	return "preferred-retained"
}

func preferredReasonText(preferred annotatedRecord, reviewNeeded bool) string {
	if reviewNeeded {
		return "当前候选暂列为保留本占位，等待人工确认"
	}
	if preferred.cliVisible {
		return "当前记录具备 CLI 可见性，作为优先保留本"
	}
	if preferred.canonicalPrimary {
		return "当前记录位于 canonical_path，作为优先保留本"
	}
	return "当前记录被选为保留本"
}
