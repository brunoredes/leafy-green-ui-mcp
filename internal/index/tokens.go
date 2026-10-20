package index

import (
	"strings"
)

// queryTerm is a stemmed query term with its weight and the word the model
// typed (or the expansion word), used in match explanations.
type queryTerm struct {
	term    string
	surface string
	weight  float64
}

// stopwords is read-only after initialization.
var stopwords = wordSet(`a an the to of for and or in on with from by is are be as at it its this that
		these those i we you me my our your their them they he she his her can could should would will shall may
		might must do does did have has had how what which when where who why want need let lets like into onto
		than then so if but not no yes some any all each per via about over under up down out only just also very
		there here use used using make build show see get set user users`)

func wordSet(s string) map[string]bool {
	m := map[string]bool{}
	for _, w := range strings.Fields(s) {
		m[w] = true
	}
	return m
}

// expansions map a user word to LeafyGreen vocabulary. Each entry adds the
// listed words to the query at reduced weight. Keep this table small and
// covered by tests; it is query expansion, not component data.
var expansions = map[string][]string{
	"dropdown":     {"select", "combobox", "menu"},
	"picker":       {"select", "combobox"},
	"pick":         {"select", "choose"},
	"choose":       {"select"},
	"autocomplete": {"combobox", "search"},
	"typeahead":    {"combobox", "search"},
	"filter":       {"combobox", "search"},
	"tag":          {"chip", "multiselect"},
	"tags":         {"chip", "multiselect"},
	"several":      {"multiple", "multiselect"},
	"many":         {"multiple", "multiselect"},
	"multiple":     {"multiselect"},
	"dialog":       {"modal"},
	"popup":        {"modal", "popover"},
	"overlay":      {"modal", "popover"},
	"confirm":      {"confirmation"},
	"delete":       {"danger", "destructive", "confirmation"},
	"remove":       {"danger", "destructive"},
	"destructive":  {"danger"},
	"irreversible": {"danger", "confirmation"},
	"toast":        {"notification"},
	"snackbar":     {"toast", "notification"},
	"notification": {"toast", "banner"},
	"alert":        {"banner", "callout", "toast"},
	"warning":      {"banner", "callout"},
	"hint":         {"tooltip", "info"},
	"tab":          {"tabs"},
	"datagrid":     {"table"},
	"grid":         {"table"},
	"calendar":     {"date", "picker"},
	"switch":       {"toggle"},
	"spinner":      {"loading"},
	"loader":       {"loading", "skeleton"},
	"sidebar":      {"side", "nav"},
	"navigation":   {"nav"},
	"textbox":      {"text", "input"},
	"field":        {"input", "form"},
	"upload":       {"file"},
	"snippet":      {"code"},
	"copy":         {"clipboard", "copyable"},
	"wizard":       {"stepper"},
	"steps":        {"stepper"},
	"chart":        {"charts"},
	"graph":        {"charts"},
	"accordion":    {"expandable", "collapsible"},
	"collapse":     {"expandable", "collapsible"},
	"collapsible":  {"expandable"},
	"temporary":    {"toast", "notification"},
	"transient":    {"toast", "notification"},
	"dismiss":      {"toast", "banner"},
	"feedback":     {"toast", "banner"},
}

// parseQuery tokenizes a query and adds expansion terms.
func parseQuery(q string) []queryTerm {
	var out []queryTerm
	seen := map[string]bool{}
	add := func(word string, weight float64) {
		for _, t := range tokenize(word) {
			if !seen[t] {
				seen[t] = true
				out = append(out, queryTerm{term: t, surface: strings.ToLower(word), weight: weight})
			}
		}
	}
	words := words(q)
	for _, w := range words {
		add(w, 1)
	}
	for _, w := range words {
		for _, e := range expansionsFor(w) {
			add(e, expansionWeight)
		}
	}
	return out
}

// words lowercases s and splits it into alphanumeric words, splitting
// camelCase identifiers first.
func words(s string) []string {
	s = camelBoundaryRe.ReplaceAllString(s, "$1 $2")
	return strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < '0' || r > '9')
	})
}

// tokenize returns stemmed, stopword-free tokens.
func tokenize(s string) []string {
	var out []string
	for _, w := range words(s) {
		if len(w) < 2 || stopwords[w] {
			continue
		}
		out = append(out, stem(w))
	}
	return out
}

// stem is a deliberately small suffix stripper: enough to match
// plural/verb forms ("tags"/"tag", "selected"/"select",
// "disabled"/"disable") and common derivations ("confirmation"/"confirm",
// "navigation"/"navigate", "collapsible"/"collapse") without a full Porter stemmer.
func stem(w string) string {
	w = stemInflection(w)
	switch {
	case len(w) > 7 && strings.HasSuffix(w, "ation"):
		w = w[:len(w)-5]
	case len(w) > 7 && (strings.HasSuffix(w, "able") || strings.HasSuffix(w, "ible")):
		w = w[:len(w)-4]
	case len(w) > 6 && strings.HasSuffix(w, "ate"):
		w = w[:len(w)-3]
	}
	if len(w) > 4 && strings.HasSuffix(w, "e") {
		w = w[:len(w)-1]
	}
	return w
}

func stemInflection(w string) string {
	switch {
	case len(w) > 4 && strings.HasSuffix(w, "ies"):
		w = w[:len(w)-3] + "y"
	case len(w) > 5 && strings.HasSuffix(w, "ing"):
		w = w[:len(w)-3]
	case len(w) > 4 && strings.HasSuffix(w, "ed"):
		w = w[:len(w)-2]
	case len(w) > 4 && (strings.HasSuffix(w, "xes") || strings.HasSuffix(w, "ches") || strings.HasSuffix(w, "shes") || strings.HasSuffix(w, "sses")):
		w = w[:len(w)-2]
	case len(w) > 3 && strings.HasSuffix(w, "s") && !strings.HasSuffix(w, "ss") && !strings.HasSuffix(w, "us"):
		w = w[:len(w)-1]
	}
	return w
}

// Terms returns the set of stemmed terms in query, including expansions.
func Terms(query string) map[string]bool {
	out := map[string]bool{}
	for _, qt := range parseQuery(query) {
		out[qt.term] = true
	}
	return out
}

// Overlap counts the distinct terms of text that appear in terms.
func Overlap(terms map[string]bool, text string) int {
	seen := map[string]bool{}
	for _, t := range tokenize(text) {
		if terms[t] && !seen[t] {
			seen[t] = true
		}
	}
	return len(seen)
}

// expansionsFor returns the expansion words for w, matching table keys by
// exact word first and then by stem ("collapsed" uses "collapse").
func expansionsFor(w string) []string {
	if e, ok := expansions[w]; ok {
		return e
	}
	s := stem(w)
	for k, e := range expansions {
		if stem(k) == s {
			return e
		}
	}
	return nil
}
