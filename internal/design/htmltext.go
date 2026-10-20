package design

import (
	"fmt"
	"io"
	"regexp"
	"slices"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// page is the result of parsing one mongodb.design HTML page.
type page struct {
	markdown    string
	components  []string
	foundations []string
}

// skipped elements never contribute text: chrome, media and interactive
// widgets.
var skipped = map[atom.Atom]bool{
	atom.Script: true, atom.Style: true, atom.Svg: true, atom.Nav: true,
	atom.Header: true, atom.Footer: true, atom.Noscript: true, atom.Button: true,
	atom.Iframe: true, atom.Img: true, atom.Picture: true, atom.Video: true,
	atom.Form: true, atom.Input: true, atom.Select: true, atom.Textarea: true,
}

var headingLevel = map[atom.Atom]int{atom.H1: 1, atom.H2: 2, atom.H3: 3, atom.H4: 4, atom.H5: 5, atom.H6: 6}

// parsePage extracts the guideline content as Markdown, plus the component
// and foundation slugs linked from the page (used to discover what to fetch).
func parsePage(r io.Reader) (page, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return page{}, fmt.Errorf("parse html: %w", err)
	}
	var b mdBuilder
	for _, root := range contentRoots(doc) {
		b.walk(root)
	}
	comps, founds := links(doc)
	return page{markdown: dropEmptySections(b.String()), components: comps, foundations: founds}, nil
}

// contentRoots picks the guideline content: React streaming boundaries
// (div[id^="S:"]) on component pages, #layout on foundation pages, then
// <main>, then <body> as a last resort.
func contentRoots(doc *html.Node) []*html.Node {
	var streamed []*html.Node
	var layout, main, body *html.Node
	for n := range doc.Descendants() {
		if n.Type != html.ElementNode {
			continue
		}
		id := attr(n, "id")
		switch {
		case n.DataAtom == atom.Div && strings.HasPrefix(id, "S:"):
			streamed = append(streamed, n)
		case id == "layout" && layout == nil:
			layout = n
		case n.DataAtom == atom.Main && main == nil:
			main = n
		case n.DataAtom == atom.Body && body == nil:
			body = n
		}
	}
	switch {
	case len(streamed) > 0:
		return streamed
	case layout != nil:
		return []*html.Node{layout}
	case main != nil:
		return []*html.Node{main}
	case body != nil:
		return []*html.Node{body}
	}
	return []*html.Node{doc}
}

var (
	componentHref  = regexp.MustCompile(`^(?:https://www\.mongodb\.design)?/component/([a-z0-9-]+)/`)
	foundationHref = regexp.MustCompile(`^(?:https://www\.mongodb\.design)?/foundation/([a-z0-9-]+)/?$`)
)

func links(doc *html.Node) (components, foundations []string) {
	comp, found := map[string]bool{}, map[string]bool{}
	for n := range doc.Descendants() {
		if n.Type != html.ElementNode || n.DataAtom != atom.A {
			continue
		}
		href := attr(n, "href")
		if m := componentHref.FindStringSubmatch(href); m != nil {
			comp[m[1]] = true
		} else if m := foundationHref.FindStringSubmatch(href); m != nil {
			found[m[1]] = true
		}
	}
	return sortedKeys(comp), sortedKeys(found)
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

// mdBuilder renders block structure as Markdown. Inline formatting other
// than code spans is dropped.
type mdBuilder struct {
	sb strings.Builder
}

func (b *mdBuilder) String() string { return b.sb.String() }

func (b *mdBuilder) block(s string) {
	if s = strings.TrimSpace(s); s != "" {
		b.sb.WriteString(s)
		b.sb.WriteString("\n\n")
	}
}

func (b *mdBuilder) walk(n *html.Node) {
	if n.Type == html.TextNode {
		b.block(collapse(n.Data))
		return
	}
	if n.Type == html.ElementNode && (skipped[n.DataAtom] || attr(n, "role") == "tablist") {
		return
	}
	if lvl, ok := headingLevel[n.DataAtom]; ok {
		if t := inline(n); t != "" {
			b.block(strings.Repeat("#", lvl) + " " + t)
		}
		return
	}
	switch n.DataAtom {
	case atom.P, atom.Blockquote:
		b.block(inline(n))
		return
	case atom.Ul, atom.Ol:
		b.list(n)
		return
	case atom.Table:
		b.table(n)
		return
	}
	for c := range n.ChildNodes() {
		b.walk(c)
	}
}

func (b *mdBuilder) list(n *html.Node) {
	var items []string
	for c := range n.Descendants() {
		if c.Type == html.ElementNode && c.DataAtom == atom.Li {
			if t := inline(c); t != "" {
				items = append(items, "- "+t)
			}
		}
	}
	b.block(strings.Join(items, "\n"))
}

func (b *mdBuilder) table(n *html.Node) {
	var rows []string
	for tr := range n.Descendants() {
		if tr.Type != html.ElementNode || tr.DataAtom != atom.Tr {
			continue
		}
		var cells []string
		for c := range tr.ChildNodes() {
			if c.Type == html.ElementNode && (c.DataAtom == atom.Td || c.DataAtom == atom.Th) {
				cells = append(cells, strings.ReplaceAll(inline(c), "|", `\|`))
			}
		}
		if len(cells) == 0 {
			continue
		}
		rows = append(rows, "| "+strings.Join(cells, " | ")+" |")
		if len(rows) == 1 {
			rows = append(rows, "|"+strings.Repeat(" --- |", len(cells)))
		}
	}
	b.block(strings.Join(rows, "\n"))
}

// inline returns the whitespace-collapsed text of n, skipping chrome and
// rendering <code> as code spans and <br> as a space.
func inline(n *html.Node) string {
	var sb strings.Builder
	var rec func(*html.Node)
	rec = func(n *html.Node) {
		switch {
		case n.Type == html.TextNode:
			sb.WriteString(n.Data)
			return
		case n.Type != html.ElementNode:
		case skipped[n.DataAtom]:
			return
		case n.DataAtom == atom.Br:
			sb.WriteByte(' ')
			return
		case n.DataAtom == atom.Code:
			sb.WriteString("`" + collapse(textOf(n)) + "`")
			return
		}
		for c := range n.ChildNodes() {
			rec(c)
		}
	}
	rec(n)
	return collapse(sb.String())
}

func textOf(n *html.Node) string {
	var sb strings.Builder
	for c := range n.Descendants() {
		if c.Type == html.TextNode {
			sb.WriteString(c.Data)
		}
	}
	return sb.String()
}

var wsRe = regexp.MustCompile(`\s+`)

func collapse(s string) string { return strings.TrimSpace(wsRe.ReplaceAllString(s, " ")) }

var mdHeadingRe = regexp.MustCompile(`^(#{1,6}) `)

// dropEmptySections removes headings whose section has no text before the
// next heading of the same or higher level (e.g. image-only "Anatomy").
func dropEmptySections(md string) string {
	blocks := strings.Split(strings.TrimSpace(md), "\n\n")
	var out []string
	for i, blk := range blocks {
		m := mdHeadingRe.FindStringSubmatch(blk)
		if m == nil {
			out = append(out, blk)
			continue
		}
		if hasContent(blocks[i+1:], len(m[1])) {
			out = append(out, blk)
		}
	}
	if len(out) == 0 {
		return ""
	}
	return strings.Join(out, "\n\n") + "\n"
}

func hasContent(rest []string, level int) bool {
	for _, blk := range rest {
		m := mdHeadingRe.FindStringSubmatch(blk)
		if m == nil {
			return true
		}
		if len(m[1]) <= level {
			return false
		}
	}
	return false
}
