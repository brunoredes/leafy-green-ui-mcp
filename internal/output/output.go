// Package output holds truncation and Markdown formatting helpers shared by
// tool and resource handlers. It imports nothing internal.
package output

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// DefaultMax is the default character budget for a single tool response.
const DefaultMax = 12000

// Truncate caps s at limit characters (runes). When it cuts, it breaks at the
// last newline inside the budget if there is one, and appends an explicit
// marker telling the model how much was dropped. limit <= 0 disables the cap.
func Truncate(s string, limit int) string {
	if limit <= 0 || utf8.RuneCountInString(s) <= limit {
		return s
	}
	runes := []rune(s)
	cut := string(runes[:limit])
	if i := strings.LastIndexByte(cut, '\n'); i > len(cut)/2 {
		cut = cut[:i]
	}
	dropped := len(runes) - utf8.RuneCountInString(cut)
	return fmt.Sprintf("%s\n\n[truncated: %d more characters — narrow the query]", strings.TrimRight(cut, "\n"), dropped)
}

// Clip shortens s to at most limit runes without a marker, adding "…" when it
// cuts. It is meant for excerpts inside a larger response.
func Clip(s string, limit int) string {
	s = strings.TrimSpace(s)
	if limit <= 0 || utf8.RuneCountInString(s) <= limit {
		return s
	}
	runes := []rune(s)
	cut := string(runes[:limit])
	if i := strings.LastIndexAny(cut, "\n "); i > len(cut)/2 {
		cut = cut[:i]
	}
	return strings.TrimRight(cut, " \n") + " …"
}

// CodeBlock renders a fenced code block, widening the fence if the body
// itself contains a triple backtick.
func CodeBlock(lang, body string) string {
	fence := "```"
	for strings.Contains(body, fence) {
		fence += "`"
	}
	return fence + lang + "\n" + strings.TrimRight(body, "\n") + "\n" + fence + "\n"
}
