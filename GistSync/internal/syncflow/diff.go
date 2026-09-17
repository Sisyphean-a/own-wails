package syncflow

import "strings"

// diff 行类型，供前端按语义着色渲染。
const (
	diffKindContext = "context"
	diffKindAdd     = "add"
	diffKindDelete  = "delete"
)

// diffLine 是 DiffLine 的内部别名，保持算法代码简洁。
type diffLine = DiffLine

// computeLineDiff 基于 LCS（最长公共子序列）计算 local→remote 的最小行级编辑序列。
// 相比逐行硬比对，插入/删除单行不会导致后续全部错位。
func computeLineDiff(local string, remote string) []diffLine {
	left := splitLines(local)
	right := splitLines(remote)
	lcs := lcsTable(left, right)
	return backtrackDiff(left, right, lcs)
}

// lcsTable 构建 LCS 动态规划表，lcs[i][j] 表示 left[i:] 与 right[j:] 的最长公共子序列长度。
func lcsTable(left []string, right []string) [][]int {
	rows := len(left) + 1
	cols := len(right) + 1
	table := make([][]int, rows)
	for i := range table {
		table[i] = make([]int, cols)
	}
	for i := len(left) - 1; i >= 0; i-- {
		for j := len(right) - 1; j >= 0; j-- {
			if left[i] == right[j] {
				table[i][j] = table[i+1][j+1] + 1
				continue
			}
			table[i][j] = maxInt(table[i+1][j], table[i][j+1])
		}
	}
	return table
}

// backtrackDiff 沿 LCS 表回溯，生成 context/add/delete 序列。
func backtrackDiff(left []string, right []string, table [][]int) []diffLine {
	lines := make([]diffLine, 0, len(left)+len(right))
	i, j := 0, 0
	for i < len(left) && j < len(right) {
		switch {
		case left[i] == right[j]:
			lines = append(lines, diffLine{Kind: diffKindContext, Text: left[i]})
			i++
			j++
		case table[i+1][j] >= table[i][j+1]:
			lines = append(lines, diffLine{Kind: diffKindDelete, Text: left[i]})
			i++
		default:
			lines = append(lines, diffLine{Kind: diffKindAdd, Text: right[j]})
			j++
		}
	}
	for ; i < len(left); i++ {
		lines = append(lines, diffLine{Kind: diffKindDelete, Text: left[i]})
	}
	for ; j < len(right); j++ {
		lines = append(lines, diffLine{Kind: diffKindAdd, Text: right[j]})
	}
	return lines
}

// diffCounts 统计新增/删除行数。
func diffCounts(lines []diffLine) (added int, removed int) {
	for _, line := range lines {
		switch line.Kind {
		case diffKindAdd:
			added++
		case diffKindDelete:
			removed++
		}
	}
	return added, removed
}

// buildUnifiedDiff 输出 unified 文本格式，保留与历史调用方兼容的 ---/+++/@@ 头。
func buildUnifiedDiff(local string, remote string) string {
	lines := computeLineDiff(local, remote)
	var out strings.Builder
	out.WriteString("--- local\n")
	out.WriteString("+++ remote\n")
	out.WriteString("@@ conflict @@\n")
	changed := false
	for _, line := range lines {
		switch line.Kind {
		case diffKindAdd:
			out.WriteString("+")
			out.WriteString(line.Text)
			out.WriteString("\n")
			changed = true
		case diffKindDelete:
			out.WriteString("-")
			out.WriteString(line.Text)
			out.WriteString("\n")
			changed = true
		default:
			out.WriteString(" ")
			out.WriteString(line.Text)
			out.WriteString("\n")
		}
	}
	if !changed {
		out.WriteString(" (no textual difference)\n")
	}
	return out.String()
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
