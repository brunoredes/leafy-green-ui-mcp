// Package markdown splits Markdown documents into heading-delimited sections
// and extracts fenced code blocks. It does not render Markdown.
package markdown

import (
	"regexp"
	"strings"
)

// CodeBlock is a fenced code block with its info-string language tag.
type CodeBlock struct {
	Lang string
	Text string
}

// Section is the content between one ATX heading and the next heading of
// any level. Path holds the headings of enclosing sections, outermost first,
// excluding this section's own heading and level-1 document titles. Content
// before the first heading is returned as a level-0 section with an empty
// heading.
type Section struct {
	Level   int
	Heading string
	Path    []string
	Body    string
	Code    []CodeBlock
}

// Title returns the heading prefixed by its ancestors, e.g.
// "Additional notes > Initial focus behavior".
func (s Section) Title() string {
	return strings.Join(append(append([]string{}, s.Path...), s.Heading), " > ")
}

var headingRe = regexp.MustCompile(`^(#{1,6})[ \t]+(.+?)[ \t#]*$`)

// Sections splits md into sections. Headings inside fenced code blocks are
// ignored.
func Sections(md string) []Section {
	var (
		out   []Section
		cur   = Section{}
		body  strings.Builder
		stack []heading
		fence string
	)
	flush := func() {
		cur.Body = strings.TrimSpace(body.String())
		cur.Code = CodeBlocks(cur.Body)
		if cur.Heading != "" || cur.Body != "" {
			out = append(out, cur)
		}
		body.Reset()
	}
	for _, line := range strings.Split(strings.ReplaceAll(md, "\r\n", "\n"), "\n") {
		if f, ok := fenceMarker(line); ok {
			switch {
			case fence == "":
				fence = f
			case strings.HasPrefix(f, fence):
				fence = ""
			}
		}
		m := headingRe.FindStringSubmatch(line)
		if fence != "" || m == nil {
			body.WriteString(line)
			body.WriteByte('\n')
			continue
		}
		flush()
		level := len(m[1])
		for len(stack) > 0 && stack[len(stack)-1].level >= level {
			stack = stack[:len(stack)-1]
		}
		var path []string
		for _, h := range stack {
			if h.level > 1 {
				path = append(path, h.text)
			}
		}
		cur = Section{Level: level, Heading: cleanHeading(m[2]), Path: path}
		stack = append(stack, heading{level, cur.Heading})
	}
	flush()
	return out
}

type heading struct {
	level int
	text  string
}

// fenceMarker reports whether line opens or closes a fence, returning the
// fence characters (``` or ~~~, possibly longer).
func fenceMarker(line string) (string, bool) {
	t := strings.TrimLeft(line, " ")
	if len(line)-len(t) > 3 {
		return "", false
	}
	for _, ch := range []string{"`", "~"} {
		if strings.HasPrefix(t, ch+ch+ch) {
			n := len(t) - len(strings.TrimLeft(t, ch))
			return strings.Repeat(ch, n), true
		}
	}
	return "", false
}

var mdLinkRe = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)

func cleanHeading(h string) string {
	h = mdLinkRe.ReplaceAllString(h, "$1")
	return strings.TrimSpace(h)
}

// CodeBlocks extracts fenced code blocks from md in document order.
func CodeBlocks(md string) []CodeBlock {
	var (
		out   []CodeBlock
		fence string
		lang  string
		buf   strings.Builder
	)
	for _, line := range strings.Split(md, "\n") {
		f, ok := fenceMarker(line)
		switch {
		case fence == "" && ok:
			fence = f
			lang = strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(line), f[:1]))
			if i := strings.IndexAny(lang, " \t{"); i >= 0 {
				lang = lang[:i]
			}
			buf.Reset()
		case fence != "" && ok && strings.HasPrefix(f, fence) && strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(line), f[:1])) == "":
			out = append(out, CodeBlock{Lang: lang, Text: strings.TrimRight(buf.String(), "\n")})
			fence = ""
		case fence != "":
			buf.WriteString(line)
			buf.WriteByte('\n')
		}
	}
	return out
}

// Prose returns md with fenced code blocks, images, HTML tags and table
// separator rows removed, for indexing and excerpts.
func Prose(md string) string {
	var (
		b     strings.Builder
		fence string
	)
	for _, line := range strings.Split(md, "\n") {
		if f, ok := fenceMarker(line); ok {
			if fence == "" {
				fence = f
			} else if strings.HasPrefix(f, fence) {
				fence = ""
			}
			continue
		}
		if fence != "" || tableSepRe.MatchString(line) {
			continue
		}
		line = imageRe.ReplaceAllString(line, "")
		line = htmlTagRe.ReplaceAllString(line, "")
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return strings.TrimSpace(blankRunsRe.ReplaceAllString(b.String(), "\n\n"))
}

var (
	imageRe     = regexp.MustCompile(`!\[[^\]]*\]\([^)]*\)`)
	htmlTagRe   = regexp.MustCompile(`</?[a-zA-Z][^>]*>`)
	tableSepRe  = regexp.MustCompile(`^\s*\|?\s*:?-{3,}`)
	blankRunsRe = regexp.MustCompile(`\n{3,}`)
	a11yRe      = regexp.MustCompile(`(?i)accessib|keyboard|\bfocus|\baria-|\baria\b|screen ?reader|\ba11y\b|\bwcag\b|\btab ?index\b|\brole\b`)
)

// IsA11yText reports whether s mentions an accessibility concern.
func IsA11yText(s string) bool { return a11yRe.MatchString(s) }

// A11ySections returns the sections whose heading mentions accessibility,
// keyboard, focus, ARIA or screen readers.
func A11ySections(sections []Section) []Section {
	var out []Section
	for _, s := range sections {
		if IsA11yText(s.Heading) {
			out = append(out, s)
		}
	}
	return out
}

// Find returns the first section whose heading matches one of names
// (case-insensitive), or false.
func Find(sections []Section, names ...string) (Section, bool) {
	for _, s := range sections {
		for _, n := range names {
			if strings.EqualFold(s.Heading, n) {
				return s, true
			}
		}
	}
	return Section{}, false
}
