package index

import (
	"cmp"
	"math"
	"regexp"
	"slices"
	"strings"

	"github.com/brunodonatelli/lg-mcp/internal/parse/markdown"
)

// BM25 parameters.
const (
	bm25K1          = 1.2
	bm25B           = 0.75
	expansionWeight = 0.6
)

// Field names shown to the model when explaining a match.
const (
	FieldName        = "name"
	FieldDescription = "description"
	FieldGuidance    = "design guidance"
	FieldDesign      = "design docs"
	FieldReadme      = "README"
	FieldProps       = "props"
	FieldHeading     = "heading"
	FieldBody        = "body"
)

var fieldWeights = map[string]float64{
	FieldName: 3, FieldDescription: 2, FieldGuidance: 2, FieldDesign: 1,
	FieldReadme: 1, FieldProps: 0.5, FieldHeading: 2, FieldBody: 1,
}

// corpus is a weighted-field BM25 index over a fixed set of documents.
type corpus struct {
	docs   []docStats
	df     map[string]int
	avgLen float64
}

type docStats struct {
	tf     map[string]float64 // field-weighted term frequency
	length float64
	fields map[string]map[string]bool // field → terms present
}

func newCorpus(docs []map[string][]string) *corpus {
	c := &corpus{df: map[string]int{}}
	var total float64
	for _, fields := range docs {
		ds := docStats{tf: map[string]float64{}, fields: map[string]map[string]bool{}}
		for name, toks := range fields {
			w := fieldWeights[name]
			ds.fields[name] = map[string]bool{}
			for _, t := range toks {
				ds.tf[t] += w
				ds.fields[name][t] = true
			}
			ds.length += w * float64(len(toks))
		}
		for t := range ds.tf {
			c.df[t]++
		}
		total += ds.length
		c.docs = append(c.docs, ds)
	}
	if len(docs) > 0 {
		c.avgLen = total / float64(len(docs))
	}
	return c
}

func (c *corpus) score(i int, q []queryTerm) float64 {
	d := c.docs[i]
	n := float64(len(c.docs))
	var s float64
	for _, qt := range q {
		tf := d.tf[qt.term]
		if tf == 0 {
			continue
		}
		df := float64(c.df[qt.term])
		idf := math.Log(1 + (n-df+0.5)/(df+0.5))
		norm := tf + bm25K1*(1-bm25B+bm25B*d.length/max(c.avgLen, 1))
		s += qt.weight * idf * tf * (bm25K1 + 1) / norm
	}
	return s
}

// nameBonusWeight scales the extra credit a package gets when a query word
// is part of its name (expansion words count at their reduced weight): someone asking to
// "toggle a setting" most likely wants Toggle, even if other READMEs say
// "toggle" more often.
const nameBonusWeight = 1.5

func (c *corpus) nameBonus(i int, q []queryTerm) float64 {
	names := c.docs[i].fields[FieldName]
	n := float64(len(c.docs))
	var s float64
	for _, qt := range q {
		if !names[qt.term] {
			continue
		}
		df := float64(c.df[qt.term])
		s += qt.weight * nameBonusWeight * math.Log(1+(n-df+0.5)/(df+0.5))
	}
	return s
}

// TermMatch explains which fields of a hit contained a query term.
type TermMatch struct {
	Term   string
	Fields []string
}

func (c *corpus) explain(i int, q []queryTerm) []TermMatch {
	var out []TermMatch
	for _, qt := range q {
		var fields []string
		for name, terms := range c.docs[i].fields {
			if terms[qt.term] {
				fields = append(fields, name)
			}
		}
		if len(fields) > 0 {
			slices.SortFunc(fields, func(a, b string) int {
				return cmp.Or(cmp.Compare(fieldWeights[b], fieldWeights[a]), strings.Compare(a, b))
			})
			out = append(out, TermMatch{Term: qt.surface, Fields: fields})
		}
	}
	return out
}

// Hit is a ranked package result.
type Hit struct {
	Package *Package
	Score   float64
	Matched []TermMatch
}

// SearchOptions filters package search.
type SearchOptions struct {
	Limit int
	Scope string // "leafygreen-ui", "lg-chat", "lg-charts" or "" / "all"
	// MinRelative drops hits scoring below this fraction of the best hit
	// (0 keeps everything).
	MinRelative float64
}

// Search ranks packages against a free-text query or UI intent.
func (ix *Index) Search(query string, opts SearchOptions) []Hit {
	q := parseQuery(query)
	if len(q) == 0 {
		return nil
	}
	var hits []Hit
	for i, p := range ix.Packages {
		if opts.Scope != "" && opts.Scope != "all" && p.Scope != opts.Scope {
			continue
		}
		if s := ix.pkgDocs.score(i, q) + ix.pkgDocs.nameBonus(i, q); s > 0 {
			hits = append(hits, Hit{Package: p, Score: s, Matched: ix.pkgDocs.explain(i, q)})
		}
	}
	slices.SortStableFunc(hits, func(a, b Hit) int { return cmp.Compare(b.Score, a.Score) })
	if opts.MinRelative > 0 && len(hits) > 0 {
		floor := hits[0].Score * opts.MinRelative
		hits = slices.DeleteFunc(hits, func(h Hit) bool { return h.Score < floor })
	}
	if opts.Limit > 0 && len(hits) > opts.Limit {
		hits = hits[:opts.Limit]
	}
	return hits
}

// SectionHit is a ranked documentation section.
type SectionHit struct {
	Ref   SectionRef
	Score float64
}

// SearchSections ranks README, design and foundation sections. If pkg is
// non-nil only that package's sections are considered.
func (ix *Index) SearchSections(query string, limit int, pkg *Package) []SectionHit {
	q := parseQuery(query)
	if len(q) == 0 {
		return nil
	}
	var hits []SectionHit
	for i, ref := range ix.secRefs {
		if pkg != nil && ref.Package != pkg {
			continue
		}
		if s := ix.sections.score(i, q); s > 0 {
			hits = append(hits, SectionHit{Ref: ref, Score: s})
		}
	}
	slices.SortStableFunc(hits, func(a, b SectionHit) int { return cmp.Compare(b.Score, a.Score) })
	if limit > 0 && len(hits) > limit {
		hits = hits[:limit]
	}
	return hits
}

var guidanceHeadingRe = regexp.MustCompile(`(?i)overview|how to use|when to use|usage|best practice`)

func (ix *Index) buildCorpora() {
	pkgDocs := make([]map[string][]string, len(ix.Packages))
	ix.secRefs = nil
	for i, p := range ix.Packages {
		pkgDocs[i] = packageFields(p)
		for _, s := range p.Readme {
			ix.addSection(SectionRef{Package: p, Origin: "README", Section: s})
		}
		if p.Design != nil {
			for _, s := range p.Design.Sections {
				ix.addSection(SectionRef{Package: p, Origin: "design", Section: s})
			}
		}
	}
	for _, slug := range sortedFoundationSlugs(ix.Foundations) {
		for _, s := range ix.Foundations[slug].Sections {
			ix.addSection(SectionRef{Origin: "foundation:" + slug, Section: s})
		}
	}
	ix.pkgDocs = newCorpus(pkgDocs)
	secDocs := make([]map[string][]string, len(ix.secRefs))
	for i, ref := range ix.secRefs {
		secDocs[i] = map[string][]string{
			FieldHeading: tokenize(ref.Section.Title()),
			FieldBody:    tokenize(markdown.Prose(ref.Section.Body)),
		}
	}
	ix.sections = newCorpus(secDocs)
}

func (ix *Index) addSection(ref SectionRef) {
	if ref.Section.Heading == "" || strings.TrimSpace(markdown.Prose(ref.Section.Body)) == "" {
		return
	}
	ix.secRefs = append(ix.secRefs, ref)
}

func sortedFoundationSlugs(m map[string]Foundation) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	slices.Sort(out)
	return out
}

func packageFields(p *Package) map[string][]string {
	f := map[string][]string{
		FieldName:        tokenize(p.Short),
		FieldDescription: tokenize(p.Description),
	}
	for _, d := range p.Decls {
		if d.Kind == "interface" && strings.HasSuffix(d.Name, "Props") {
			f[FieldName] = append(f[FieldName], tokenize(strings.TrimSuffix(d.Name, "Props"))...)
			for _, m := range d.Members {
				f[FieldProps] = append(f[FieldProps], tokenize(m.Name+" "+m.JSDoc)...)
			}
		}
	}
	for _, s := range p.Readme {
		f[FieldReadme] = append(f[FieldReadme], tokenize(s.Heading+" "+markdown.Prose(s.Body))...)
	}
	if p.Design != nil {
		for _, s := range p.Design.Sections {
			field := FieldDesign
			if guidanceHeadingRe.MatchString(s.Title()) {
				field = FieldGuidance
			}
			f[field] = append(f[field], tokenize(s.Heading+" "+markdown.Prose(s.Body))...)
		}
	}
	return f
}
