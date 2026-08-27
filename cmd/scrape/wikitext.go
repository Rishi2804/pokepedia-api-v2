package main

import (
	"regexp"
	"strings"
)

// templateOccurrence is one {{name|...}} found in wikitext, with its body
// (everything between the first "|" and the matching "}}") and its start
// offset, so callers that need document order -- see the Pokemon dex-entry
// walk in bulba.go -- can interleave occurrences of different template names.
type templateOccurrence struct {
	Name  string
	Body  string // raw text after "name|", or "" if the template took no args
	Start int
}

// findTemplates scans text for every top-level occurrence of {{name|...}} or
// {{name}} for any name in names, respecting brace nesting.
//
// A naive regex like `\{\{movedescentry\|(.*?)\|(.*?)\}\}` is wrong here:
// Bulbapedia nests templates inside the games field --
// {{movedescentry|{{gameabbrev9|ZA}}|text}} -- and the non-greedy `\}\}`
// matches the INNER template's closing braces, truncating the games field
// and silently making every "does this contain ZA" check return false. That
// looks exactly like "Bulbapedia has no Z-A data" rather than a parser bug,
// which is why this is a real scanner instead of a regex.
func findTemplates(text string, names ...string) []templateOccurrence {
	wanted := make(map[string]bool, len(names))
	for _, n := range names {
		wanted[strings.ToLower(n)] = true
	}

	var out []templateOccurrence
	i := 0
	for i < len(text) {
		if !strings.HasPrefix(text[i:], "{{") {
			i++
			continue
		}
		start := i
		// Read the template name up to the first "|" or "}}" at this level.
		j := i + 2
		nameEnd := j
		for nameEnd < len(text) && text[nameEnd] != '|' && !strings.HasPrefix(text[nameEnd:], "}}") {
			nameEnd++
		}
		name := strings.TrimSpace(text[j:nameEnd])

		end, body := scanTemplateBody(text, start)
		if end < 0 {
			// Unterminated template (malformed page); stop scanning it.
			break
		}
		if wanted[strings.ToLower(name)] {
			out = append(out, templateOccurrence{Name: name, Body: body, Start: start})
		}
		i = end
	}
	return out
}

// scanTemplateBody, given the index of the "{{" that opens a template,
// returns the index just past its matching "}}" and everything after the
// first top-level "|" (empty string if the template has no "|"). Nesting of
// {{...}} is tracked so an inner template's braces never end the outer one.
func scanTemplateBody(text string, openAt int) (endIdx int, body string) {
	depth := 0
	i := openAt
	bodyStart := -1
	for i < len(text) {
		switch {
		case strings.HasPrefix(text[i:], "{{"):
			depth++
			i += 2
		case strings.HasPrefix(text[i:], "}}"):
			depth--
			i += 2
			if depth == 0 {
				if bodyStart < 0 {
					return i, ""
				}
				return i, text[bodyStart : i-2]
			}
		default:
			if depth == 1 && bodyStart < 0 && text[i] == '|' {
				bodyStart = i + 1
			}
			i++
		}
	}
	return -1, ""
}

// splitTopLevel splits a template body on "|" characters that are not
// themselves inside a nested {{...}} or [[...]], which is how
// {{movedescentry|{{gameabbrev9|ZA}}|text}}'s body divides into exactly two
// fields (games, text) rather than three.
func splitTopLevel(body string) []string {
	var parts []string
	var cur strings.Builder
	depth := 0
	i := 0
	for i < len(body) {
		switch {
		case strings.HasPrefix(body[i:], "{{"), strings.HasPrefix(body[i:], "[["):
			depth++
			cur.WriteString(body[i : i+2])
			i += 2
		case strings.HasPrefix(body[i:], "}}"), strings.HasPrefix(body[i:], "]]"):
			depth--
			cur.WriteString(body[i : i+2])
			i += 2
		case body[i] == '|' && depth == 0:
			parts = append(parts, cur.String())
			cur.Reset()
			i++
		default:
			cur.WriteByte(body[i])
			i++
		}
	}
	parts = append(parts, cur.String())
	return parts
}

// namedArgs extracts key=value top-level fields from a split template body
// (e.g. Dex/EntryN's v=, v2=, entry=), keyed by the part before "=".
// Positional (unkeyed) parts are ignored since every field this tool reads
// off Dex/Entry and movedescentry is named.
func namedArgs(parts []string) map[string]string {
	out := map[string]string{}
	for _, p := range parts {
		if eq := strings.IndexByte(p, '='); eq >= 0 {
			key := strings.TrimSpace(p[:eq])
			out[key] = p[eq+1:]
		}
	}
	return out
}

var gameAbbrevRe = regexp.MustCompile(`\{\{(gameabbrev\w*)\|([^}]+)\}\}`)

// gameAbbrevCodes extracts every {{gameabbrevN|CODE}} token's CODE from a
// movedescentry's games field. Multiple codes can appear joined by <br> --
// {{gameabbrev8|BDSP}}<br>{{gameabbrev9|SV}} -- so this returns all of them.
func gameAbbrevCodes(gamesField string) []string {
	var codes []string
	for _, m := range gameAbbrevRe.FindAllStringSubmatch(gamesField, -1) {
		codes = append(codes, m[2])
	}
	return codes
}

var (
	scTagRe    = regexp.MustCompile(`</?sc>`)
	smallTagRe = regexp.MustCompile(`</?small>`)
	ttTemplRe  = regexp.MustCompile(`\{\{tt\|([^|}]*)\|[^}]*\}\}`)
	pTemplRe   = regexp.MustCompile(`\{\{[pm]\|([^|}]*?)(?:\|[^}]*)?\}\}`)
	italicsRe  = regexp.MustCompile(`''+`)
	brTagRe    = regexp.MustCompile(`<br\s*/?>`)
)

// cleanWikitext strips the inline markup Bulbapedia uses inside flavor text
// so the stored string reads as plain prose, matching how every existing row
// in *descriptions is stored (verified: no embedded newlines, no doubled
// spaces, no residual wiki markup in the current data).
func cleanWikitext(s string) string {
	s = ttTemplRe.ReplaceAllString(s, "$1") // {{tt|shown|tooltip}} -> shown
	s = pTemplRe.ReplaceAllString(s, "$1")  // {{p|Name}} / {{m|Name}} -> Name
	s = scTagRe.ReplaceAllString(s, "")
	s = smallTagRe.ReplaceAllString(s, "")
	s = italicsRe.ReplaceAllString(s, "")
	s = brTagRe.ReplaceAllString(s, " ")
	s = strings.ReplaceAll(s, "&mdash;", "—")
	s = strings.ReplaceAll(s, "&ndash;", "–")
	s = strings.ReplaceAll(s, "\n", " ")
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	return strings.TrimSpace(s)
}

// section extracts the body of a top-level "==Heading==" section, up to but
// not including the next top-level heading. Sub-headings (===...===) inside
// the section are left in the returned body.
func section(wikitext, heading string) (string, bool) {
	re := regexp.MustCompile(`(?ms)^==\s*` + regexp.QuoteMeta(heading) + `\s*==\s*$`)
	loc := re.FindStringIndex(wikitext)
	if loc == nil {
		return "", false
	}
	rest := wikitext[loc[1]:]
	next := regexp.MustCompile(`(?m)^==[^=]`).FindStringIndex(rest)
	if next == nil {
		return rest, true
	}
	return rest[:next[0]], true
}
