package syncer

import (
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/sergi/go-diff/diffmatchpatch"
)

type edit struct {
	start, end int
	text       string
}

func lineEdits(base, changed string) []edit {
	d := diffmatchpatch.New()
	a, b, lines := d.DiffLinesToRunes(base, changed)
	diffs := d.DiffCharsToLines(d.DiffMainRunes(a, b, false), lines)
	out := []edit{}
	position := 0
	start := -1
	end := 0
	var text strings.Builder
	flush := func() {
		if start >= 0 {
			out = append(out, edit{start, end, text.String()})
			start = -1
			text.Reset()
		}
	}
	for _, v := range diffs {
		switch v.Type {
		case diffmatchpatch.DiffEqual:
			flush()
			position += len(v.Text)
		case diffmatchpatch.DiffDelete:
			if start < 0 {
				start = position
				end = position
			}
			position += len(v.Text)
			end = position
		case diffmatchpatch.DiffInsert:
			if start < 0 {
				start = position
				end = position
			}
			text.WriteString(v.Text)
		}
	}
	flush()
	return out
}

// Rule：只合并不重叠的行级编辑；同一插入点的不同内容也属于冲突。
func MergeMarkdown(base, left, right []byte) ([]byte, bool) {
	if len(base) > MarkdownLimit || len(left) > MarkdownLimit || len(right) > MarkdownLimit || !utf8.Valid(base) || !utf8.Valid(left) || !utf8.Valid(right) {
		return nil, false
	}
	a, b := lineEdits(string(base), string(left)), lineEdits(string(base), string(right))
	all := append([]edit{}, a...)
	for _, v := range b {
		duplicate := false
		for _, w := range a {
			if v == w {
				duplicate = true
				break
			}
			if v.start <= w.end && w.start <= v.end {
				if v.start == v.end || w.start == w.end || v.start < w.end && w.start < v.end {
					return nil, false
				}
			}
		}
		if !duplicate {
			all = append(all, v)
		}
	}
	sort.Slice(all, func(i, j int) bool { return all[i].start < all[j].start })
	var out strings.Builder
	pos := 0
	for _, v := range all {
		out.Write(base[pos:v.start])
		out.WriteString(v.text)
		pos = v.end
	}
	out.Write(base[pos:])
	return []byte(out.String()), true
}
