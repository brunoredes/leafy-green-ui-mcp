package tstypes

import (
	"regexp"
	"strings"
)

var (
	propRe   = regexp.MustCompile(`(?s)^(?:readonly\s+)?(['"]?[A-Za-z_$][\w$-]*['"]?)(\?)?\s*:\s*(.+)$`)
	methodRe = regexp.MustCompile(`(?s)^([A-Za-z_$][\w$]*)(\?)?\s*(?:<[^(]*>)?\s*(\(.*)$`)
)

// Members splits the inside of an interface or object-literal type into its
// top-level members. Index signatures and unparseable members are skipped.
func Members(body string) []Member {
	var (
		out   []Member
		doc   string
		buf   strings.Builder
		depth int
	)
	emit := func() {
		if m, ok := parseMember(buf.String()); ok {
			m.JSDoc = doc
			out = append(out, m)
		}
		buf.Reset()
		doc = ""
	}
	for i := 0; i < len(body); {
		if j := skipNonCode(body, i); j != i {
			if depth == 0 && strings.HasPrefix(body[i:], "/**") && strings.TrimSpace(buf.String()) == "" {
				doc = cleanJSDoc(body[i:j])
			} else if !strings.HasPrefix(body[i:], "//") && !strings.HasPrefix(body[i:], "/*") {
				buf.WriteString(body[i:j])
			}
			i = j
			continue
		}
		ch := body[i]
		switch {
		case ch == '(' || ch == '[' || ch == '{' || ch == '<':
			depth++
		case ch == ')' || ch == ']' || ch == '}':
			depth--
		case ch == '>' && (i == 0 || body[i-1] != '='):
			depth--
		}
		if depth == 0 && (ch == ';' || ch == ',' || (ch == '\n' && memberComplete(buf.String(), body[i+1:]))) {
			emit()
			i++
			continue
		}
		buf.WriteByte(ch)
		i++
	}
	emit()
	return out
}

// memberComplete reports whether a newline ends the member accumulated in
// cur, given the text that follows. Union and intersection types spread
// over several lines continue the member.
func memberComplete(cur, rest string) bool {
	t := strings.TrimSpace(cur)
	if t == "" {
		return false
	}
	for _, suffix := range []string{":", "|", "&", "=>", "?"} {
		if strings.HasSuffix(t, suffix) {
			return false
		}
	}
	next := strings.TrimLeft(rest, " \t\r\n")
	return !strings.HasPrefix(next, "|") && !strings.HasPrefix(next, "&") && !strings.HasPrefix(next, "=>")
}

func parseMember(s string) (Member, bool) {
	s = strings.TrimSpace(s)
	if s == "" || strings.HasPrefix(s, "[") {
		return Member{}, false
	}
	if m := methodRe.FindStringSubmatch(s); m != nil && !strings.Contains(m[1], ":") {
		return Member{Name: m[1], Optional: m[2] == "?", Type: collapseSpace(m[3])}, true
	}
	m := propRe.FindStringSubmatch(s)
	if m == nil {
		return Member{}, false
	}
	return Member{
		Name:     strings.Trim(m[1], `'"`),
		Optional: m[2] == "?",
		Type:     collapseSpace(strings.TrimSpace(m[3])),
	}, true
}

var a11yDocRe = regexp.MustCompile(`(?i)accessib|screen ?reader|\baria\b|aria-|keyboard|\bfocus|\brole\b|\blabel(?:led)?\b`)

// AriaProps returns the members of d that carry accessibility meaning:
// aria-* and role props, and members whose JSDoc mentions accessibility,
// focus, keyboard use or labelling.
func AriaProps(d Decl) []Member {
	var out []Member
	for _, m := range d.Members {
		if strings.HasPrefix(m.Name, "aria-") || m.Name == "role" || a11yDocRe.MatchString(m.JSDoc) {
			out = append(out, m)
		}
	}
	return out
}
