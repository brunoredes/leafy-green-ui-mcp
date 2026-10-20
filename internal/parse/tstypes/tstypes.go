// Package tstypes extracts exported interface and type declarations, with
// their JSDoc, from TypeScript source or declaration files. It works on text
// and does not resolve types: extends clauses are recorded verbatim.
package tstypes

import (
	"regexp"
	"strings"
)

// Kind is the declaration keyword.
type Kind string

// Declaration kinds.
const (
	Interface Kind = "interface"
	TypeAlias Kind = "type"
)

// Decl is one exported declaration.
type Decl struct {
	Name       string
	Kind       Kind
	TypeParams string // e.g. "<M extends boolean>", verbatim
	Extends    string // interface extends clause, verbatim
	JSDoc      string // cleaned text of the preceding /** */ comment
	Source     string // the full declaration text, without JSDoc
	Members    []Member
}

// Member is one property or method of an interface (or of an object-literal
// type alias).
type Member struct {
	Name     string
	Optional bool
	Type     string
	JSDoc    string
}

var declRe = regexp.MustCompile(`(?m)^[ \t]*export[ \t]+(?:declare[ \t]+)?(interface|type)[ \t]+([A-Za-z_$][\w$]*)`)

// Extract returns the exported interface and type declarations in src, in
// source order.
func Extract(src string) []Decl {
	var out []Decl
	for _, m := range declRe.FindAllStringSubmatchIndex(src, -1) {
		d, ok := extractAt(src, m[0], m[1], Kind(src[m[2]:m[3]]), src[m[4]:m[5]])
		if ok {
			out = append(out, d)
		}
	}
	return out
}

func extractAt(src string, start, nameEnd int, kind Kind, name string) (Decl, bool) {
	d := Decl{Name: name, Kind: kind, JSDoc: precedingJSDoc(src[:start])}
	i := nameEnd
	if j := skipSpace(src, i); j < len(src) && src[j] == '<' {
		end := matchAngle(src, j)
		if end < 0 {
			return Decl{}, false
		}
		d.TypeParams = src[j : end+1]
		i = end + 1
	}
	switch kind {
	case Interface:
		open := scanTo(src, i, '{')
		if open < 0 {
			return Decl{}, false
		}
		if h := strings.TrimSpace(src[i:open]); strings.HasPrefix(h, "extends") {
			d.Extends = collapseSpace(strings.TrimSpace(strings.TrimPrefix(h, "extends")))
		}
		closeIdx := matchBrace(src, open)
		if closeIdx < 0 {
			return Decl{}, false
		}
		d.Source = strings.TrimSpace(src[start : closeIdx+1])
		d.Members = Members(src[open+1 : closeIdx])
	case TypeAlias:
		end := scanTo(src, i, ';')
		if end < 0 {
			end = len(src) - 1
		}
		d.Source = strings.TrimSpace(src[start : end+1])
		if eq := strings.IndexByte(src[i:end+1], '='); eq >= 0 {
			rhs := strings.TrimSpace(src[i+eq+1 : end])
			if strings.HasPrefix(rhs, "{") {
				if c := matchBrace(rhs, 0); c > 0 {
					d.Members = Members(rhs[1:c])
				}
			}
		}
	}
	return d, true
}

// precedingJSDoc returns the cleaned /** */ comment that ends immediately
// (modulo whitespace) before the end of before.
func precedingJSDoc(before string) string {
	t := strings.TrimRight(before, " \t\r\n")
	if !strings.HasSuffix(t, "*/") {
		return ""
	}
	open := strings.LastIndex(t, "/**")
	if open < 0 {
		return ""
	}
	return cleanJSDoc(t[open:])
}

// cleanJSDoc strips comment delimiters and leading asterisks.
func cleanJSDoc(c string) string {
	c = strings.TrimSuffix(strings.TrimPrefix(c, "/**"), "*/")
	lines := strings.Split(c, "\n")
	for i, l := range lines {
		l = strings.TrimSpace(l)
		l = strings.TrimPrefix(l, "*")
		lines[i] = strings.TrimPrefix(l, " ")
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

var spaceRe = regexp.MustCompile(`\s+`)

func collapseSpace(s string) string { return spaceRe.ReplaceAllString(s, " ") }

func skipSpace(s string, i int) int {
	for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r') {
		i++
	}
	return i
}
