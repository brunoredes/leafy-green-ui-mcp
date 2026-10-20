package tstypes

// The scanner skips string literals, template literals and comments so that
// brackets inside them do not affect depth tracking.

// skipNonCode returns the index just past the string or comment starting at
// i, or i if none starts there.
func skipNonCode(s string, i int) int {
	switch {
	case s[i] == '"' || s[i] == '\'' || s[i] == '`':
		q := s[i]
		for j := i + 1; j < len(s); j++ {
			if s[j] == '\\' {
				j++
				continue
			}
			if s[j] == q {
				return j + 1
			}
		}
		return len(s)
	case i+1 < len(s) && s[i] == '/' && s[i+1] == '/':
		for j := i; j < len(s); j++ {
			if s[j] == '\n' {
				return j
			}
		}
		return len(s)
	case i+1 < len(s) && s[i] == '/' && s[i+1] == '*':
		for j := i + 2; j+1 < len(s); j++ {
			if s[j] == '*' && s[j+1] == '/' {
				return j + 2
			}
		}
		return len(s)
	}
	return i
}

// matchBrace returns the index of the '}' matching the '{' at open, or -1.
func matchBrace(s string, open int) int {
	return matchPair(s, open, '{', '}')
}

// matchAngle returns the index of the '>' matching the '<' at open, or -1.
// Arrow tokens "=>" inside the brackets are not counted as closers.
func matchAngle(s string, open int) int {
	depth := 0
	for i := open; i < len(s); {
		if j := skipNonCode(s, i); j != i {
			i = j
			continue
		}
		switch s[i] {
		case '<':
			depth++
		case '>':
			if i > 0 && s[i-1] == '=' {
				break
			}
			depth--
			if depth == 0 {
				return i
			}
		}
		i++
	}
	return -1
}

func matchPair(s string, open int, o, c byte) int {
	depth := 0
	for i := open; i < len(s); {
		if j := skipNonCode(s, i); j != i {
			i = j
			continue
		}
		switch s[i] {
		case o:
			depth++
		case c:
			depth--
			if depth == 0 {
				return i
			}
		}
		i++
	}
	return -1
}

// scanTo returns the index of the first target byte at bracket depth 0
// (counting (), [] and {}), starting at i, or -1. Brackets opened before
// target are skipped even if target itself is a bracket opener: scanning for
// '{' stops at the first '{' at depth 0.
func scanTo(s string, i int, target byte) int {
	depth := 0
	for i < len(s) {
		if j := skipNonCode(s, i); j != i {
			i = j
			continue
		}
		ch := s[i]
		if ch == target && depth == 0 {
			return i
		}
		switch ch {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
		case '<':
			if end := matchAngle(s, i); end > 0 {
				i = end + 1
				continue
			}
		}
		i++
	}
	return -1
}
