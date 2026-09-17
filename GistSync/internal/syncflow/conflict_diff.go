package syncflow

import (
	"bytes"
	"context"
	"os"
	"strings"

	"GistSync/internal/security"
)

const (
	diffStatusReady      = "ready"
	diffStatusBinary     = "binary_unsupported"
	diffStatusTooLarge   = "too_large"
	diffStatusDecodeFail = "decode_failed"
	diffStatusReadFail   = "read_failed"
)

const (
	maxDiffSourceBytes  = 200 * 1024
	maxDiffPreviewRunes = 12 * 1024
	maxDiffPreviewLines = 600
)

func (s *Service) buildConflict(
	ctx context.Context,
	gistID string,
	item manifestSnapshotItem,
	req ApplySnapshotRequest,
	targetPath string,
) ApplyConflict {
	conflict := ApplyConflict{
		ItemID:     item.ItemID,
		TargetPath: targetPath,
	}
	local, remote, status := s.loadDiffSources(ctx, gistID, item, req.MasterPassword, targetPath)
	conflict.DiffStatus = status
	if status != diffStatusReady {
		return conflict
	}
	lines := computeLineDiff(local, remote)
	conflict.DiffLines = trimDiffLines(lines, maxDiffPreviewLines)
	conflict.AddedLines, conflict.RemovedLines = diffCounts(lines)
	conflict.DiffPreview = trimRunes(buildUnifiedDiff(local, remote), maxDiffPreviewRunes)
	return conflict
}

// loadDiffSources 读取本地与解密后的远端文本，返回内容及可预览状态。
func (s *Service) loadDiffSources(
	ctx context.Context,
	gistID string,
	item manifestSnapshotItem,
	password string,
	targetPath string,
) (local string, remote string, status string) {
	if empty(password) {
		return "", "", diffStatusDecodeFail
	}
	localRaw, err := os.ReadFile(targetPath)
	if err != nil {
		return "", "", diffStatusReadFail
	}
	if len(localRaw) > maxDiffSourceBytes {
		return "", "", diffStatusTooLarge
	}
	if isLikelyBinary(localRaw) {
		return "", "", diffStatusBinary
	}
	encrypted, err := s.cloud.GetFileContent(ctx, FileRequest{GistID: gistID, FileName: item.BlobFile})
	if err != nil {
		return "", "", diffStatusReadFail
	}
	remoteText, err := security.DecryptString(encrypted, password)
	if err != nil {
		return "", "", diffStatusDecodeFail
	}
	remoteRaw := []byte(remoteText)
	if len(remoteRaw) > maxDiffSourceBytes {
		return "", "", diffStatusTooLarge
	}
	if isLikelyBinary(remoteRaw) {
		return "", "", diffStatusBinary
	}
	return string(localRaw), remoteText, diffStatusReady
}

// trimDiffLines 限制返回给前端的结构化行数，超出部分以截断提示收尾。
func trimDiffLines(lines []diffLine, limit int) []diffLine {
	if limit <= 0 || len(lines) <= limit {
		return lines
	}
	trimmed := make([]diffLine, 0, limit+1)
	trimmed = append(trimmed, lines[:limit]...)
	trimmed = append(trimmed, diffLine{Kind: diffKindContext, Text: "... [diff truncated]"})
	return trimmed
}

func splitLines(raw string) []string {
	normalized := strings.ReplaceAll(raw, "\r\n", "\n")
	return strings.Split(normalized, "\n")
}

func trimRunes(input string, limit int) string {
	if limit <= 0 {
		return ""
	}
	runes := []rune(input)
	if len(runes) <= limit {
		return input
	}
	return string(runes[:limit]) + "\n... [diff truncated]"
}

func isLikelyBinary(raw []byte) bool {
	if len(raw) == 0 {
		return false
	}
	if bytes.IndexByte(raw, 0x00) >= 0 {
		return true
	}
	printable := 0
	for _, b := range raw {
		if b == '\n' || b == '\r' || b == '\t' || (b >= 0x20 && b <= 0x7E) {
			printable++
		}
	}
	return float64(printable)/float64(len(raw)) < 0.9
}
