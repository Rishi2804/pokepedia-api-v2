package main

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// bulbapediaOnlyItemIDs assigns ids to the Showdown items PokeAPI has no
// item for (Gen 2's Berry/Mint Berry/..., the bows, Mail, and "Stick", the
// old name of Leek). Ids start at 20000 so they can never collide with a
// PokeAPI id. The table is explicit, not computed from a sorted list, so a
// later Showdown update can't shift existing ids: a new Showdown-only item
// shows up in the report until someone adds it here.
var bulbapediaOnlyItemIDs = map[string]int32{
	"berry":        20001,
	"berserkgene":  20002,
	"bitterberry":  20003,
	"burntberry":   20004,
	"goldberry":    20005,
	"iceberry":     20006,
	"mail":         20007,
	"mintberry":    20008,
	"miracleberry": 20009,
	"mysteryberry": 20010,
	"pinkbow":      20011,
	"polkadotbow":  20012,
	"przcureberry": 20013,
	"psncureberry": 20014,
	"stick":        20015,
}

// bulbapediaOnlyItemSlugs overrides the slug derived from the Showdown name
// where it would collide with a PokeAPI item. Showdown calls the gen 2-7 item
// "Stick" and the gen 8+ one "Leek"; PokeAPI has one item, slug "stick", that
// matches the latter, so the older id gets its own slug.
var bulbapediaOnlyItemSlugs = map[string]string{
	"stick": "stick-legacy",
}

// bulbapediaGameCodes maps a {{gameabbrevN|CODE}} token to the public.game
// values it covers. Bulbapedia groups games that share text, so one code can
// cover several (GSC, SMUSUMPE, SVZA, ...).
var bulbapediaGameCodes = map[string][]string{
	"RB": {"red", "blue"}, "Y": {"yellow"}, "RBY": {"red", "blue", "yellow"},
	"GS": {"gold", "silver"}, "C": {"crystal"}, "GSC": {"gold", "silver", "crystal"},
	"RS": {"ruby", "sapphire"}, "E": {"emerald"}, "RSE": {"ruby", "sapphire", "emerald"},
	"FRLG": {"firered", "leafgreen"},
	"DP":   {"diamond", "pearl"}, "Pt": {"platinum"}, "DPPt": {"diamond", "pearl", "platinum"},
	"HGSS": {"heartgold", "soulsilver"}, "PtHGSS": {"platinum", "heartgold", "soulsilver"},
	"DPPtHGSS": {"diamond", "pearl", "platinum", "heartgold", "soulsilver"},
	"BW":       {"black", "white"}, "B2W2": {"black-2", "white-2"},
	"BWB2W2": {"black", "white", "black-2", "white-2"},
	"XY":     {"x", "y"}, "ORAS": {"omega-ruby", "alpha-sapphire"},
	"XYORAS": {"x", "y", "omega-ruby", "alpha-sapphire"},
	"SM":     {"sun", "moon"}, "USUM": {"ultra-sun", "ultra-moon"},
	"SMUSUM":   {"sun", "moon", "ultra-sun", "ultra-moon"},
	"PE":       {"lets-go-pikachu", "lets-go-eevee"},
	"SMUSUMPE": {"sun", "moon", "ultra-sun", "ultra-moon", "lets-go-pikachu", "lets-go-eevee"},
	"SwSh":     {"sword", "shield"}, "BDSP": {"brilliant-diamond", "shining-pearl"},
	"LA": {"legends-arceus"}, "SV": {"scarlet", "violet"}, "ZA": {"legends-za"},
	"SVZA": {"scarlet", "violet", "legends-za"},
	// Combined groups Bulbapedia also uses. "ID" is the Indigo Disk DLC,
	// folded onto Scarlet/Violet like every other DLC (see 000003).
	"BDSPLA":     {"brilliant-diamond", "shining-pearl", "legends-arceus"},
	"SwShBDSP":   {"sword", "shield", "brilliant-diamond", "shining-pearl"},
	"SwShLA":     {"sword", "shield", "legends-arceus"},
	"SwShBDSPLA": {"sword", "shield", "brilliant-diamond", "shining-pearl", "legends-arceus"},
	"USUMPE":     {"ultra-sun", "ultra-moon", "lets-go-pikachu", "lets-go-eevee"},
	"ID":         {"scarlet", "violet"},
}

// bulbapediaIgnoredCodes are game abbreviations for spin-offs this project
// does not model (Stadium, Colosseum, Pokemon GO, Champions, ...).
var bulbapediaIgnoredCodes = map[string]bool{
	"CHP": true, "Champ": true, "Champs": true, "Conq": true, "GO": true,
	"GTI": true, "Masters": true, "Pokopia": true, "RTDX": true, "SMD": true,
	"Sleep": true, "Stad": true, "Stad2": true, "TDS": true, "Colo": true,
	"XD": true, "ColoXD": true, "Quest": true, "S": true, "PBR": true, "Rumble": true, "Ranger": true, "Snap": true,
}

// bulbapediaItemTitles are display names whose Bulbapedia page title is not
// the name itself: the Gen 2 cure berries are titled without spaces.
var bulbapediaItemTitles = map[string]string{
	"PRZ Cure Berry": "PRZCureBerry",
	"PSN Cure Berry": "PSNCureBerry",
}

var bulbapediaRomanGen = map[string]int32{
	"I": 1, "II": 2, "III": 3, "IV": 4, "V": 5, "VI": 6, "VII": 7, "VIII": 8, "IX": 9,
}

var infoboxGenRe = regexp.MustCompile(`(?m)^\|\s*gen\s*=\s*([IVX]+)\s*$`)

// bulbapediaItemInfo is what the supplement reads off one item page.
type bulbapediaItemInfo struct {
	Gen          *int32
	Descriptions map[string]string // public.game -> cleaned text
	UnknownCodes []string
}

// parseBulbapediaItem pulls the introduction generation (ItemInfobox's
// |gen=IX) and every {{movedescentry|games|text}} description out of an item
// page. Game codes that are neither mapped nor deliberately ignored come back
// in UnknownCodes so the report can surface a new abbreviation instead of
// silently dropping its text.
func parseBulbapediaItem(wikitext string) bulbapediaItemInfo {
	info := bulbapediaItemInfo{Descriptions: map[string]string{}}

	if m := infoboxGenRe.FindStringSubmatch(wikitext); m != nil {
		if g, ok := bulbapediaRomanGen[m[1]]; ok {
			info.Gen = &g
		}
	}

	unknown := map[string]bool{}
	for _, occ := range findTemplates(wikitext, "movedescentry") {
		parts := splitTopLevel(occ.Body)
		if len(parts) < 2 {
			continue
		}
		text := cleanWikitext(strings.Join(parts[1:], "|"))
		if text == "" {
			continue
		}
		for _, code := range gameAbbrevCodes(parts[0]) {
			games, ok := bulbapediaGameCodes[code]
			if !ok {
				if !bulbapediaIgnoredCodes[code] {
					unknown[code] = true
				}
				continue
			}
			for _, g := range games {
				if _, dup := info.Descriptions[g]; !dup {
					info.Descriptions[g] = text
				}
			}
		}
	}
	for c := range unknown {
		info.UnknownCodes = append(info.UnknownCodes, c)
	}
	sort.Strings(info.UnknownCodes)
	return info
}

// newBulbapediaOnlyRow builds the row for a Showdown item PokeAPI doesn't
// have. Showdown only models items that can be held, so these are holdable by
// definition; category and pocket are the closest PokeAPI equivalents.
func newBulbapediaOnlyRow(id int32, sd showdownItem) itemRow {
	slug := strings.NewReplacer("'", "", ".", "").Replace(strings.ToLower(sd.Name))
	slug = strings.Join(strings.Fields(slug), "-")
	if override, ok := bulbapediaOnlyItemSlugs[sd.ID]; ok {
		slug = override
	}
	gen := int32(sd.Gen)
	sdID := sd.ID
	spriteKey := "items/" + slug + ".png"

	category, pocket := "held-items", "misc"
	switch {
	case strings.HasSuffix(sd.ID, "berry"):
		category, pocket = "medicine", "berries"
	case sd.ID == "mail":
		category, pocket = "all-mail", "mail"
	}

	return itemRow{
		ID: id, Name: slug, DisplayName: sd.Name, Category: category, Pocket: pocket,
		Gen: &gen, Sprite: &spriteKey, ShowdownID: &sdID, Holdable: true, LegalFormats: sd.LegalFormats,
		Source: "bulbapedia",
	}
}

// supplementItemsFromBulbapedia fills what PokeAPI lacks: the generation of
// items with no game_indices, and flavor text for games PokeAPI has no entry
// for (all of Legends: Z-A, plus holes like Metal Alloy). It is gap-fill
// only -- a PokeAPI description is never replaced -- and returns notes for
// anything it could not resolve.
func supplementItemsFromBulbapedia(rows []itemRow, cacheDir string, batchSize int) ([]unresolvedItem, error) {
	pageCache, err := newDiskCache(cacheDir + "/bulbapedia")
	if err != nil {
		return nil, err
	}

	titleFor := func(r itemRow) string {
		title := titleCharReplacements.Replace(r.DisplayName)
		if override, ok := bulbapediaItemTitles[title]; ok {
			return override
		}
		return title
	}
	titles := make([]string, 0, len(rows))
	for _, r := range rows {
		if r.DisplayName != "" {
			titles = append(titles, titleFor(r))
		}
	}
	pages, err := fetchBulbapediaBatch(pageCache, uniqueStrings(titles), batchSize)
	if err != nil {
		return nil, err
	}

	// Items that share their name with a disambiguation or hub page
	// ("Berry", "Leek") live at "<name> (item)"; retry only those that came
	// back missing or without a single description or infobox generation.
	var retry []string
	for _, r := range rows {
		if r.DisplayName == "" {
			continue
		}
		p := pages[titleFor(r)]
		if p.Missing || parseBulbapediaItem(p.Content).isEmpty() {
			retry = append(retry, titleFor(r)+" (item)")
		}
	}
	if len(retry) > 0 {
		more, err := fetchBulbapediaBatch(pageCache, uniqueStrings(retry), batchSize)
		if err != nil {
			return nil, err
		}
		for t, p := range more {
			pages[t] = p
		}
	}

	var unresolved []unresolvedItem
	missingMinor := 0
	unknownCodes := map[string]bool{}

	for i := range rows {
		r := &rows[i]
		if r.DisplayName == "" {
			continue
		}
		title := titleFor(*r)
		info := parseBulbapediaItem(pages[title].Content)
		usedTitle := title
		if pages[title].Missing || info.isEmpty() {
			alt := title + " (item)"
			if altInfo := parseBulbapediaItem(pages[alt].Content); !pages[alt].Missing && !altInfo.isEmpty() {
				info, usedTitle = altInfo, alt
			}
		}

		if pages[usedTitle].Missing || info.isEmpty() {
			// A hub page with no description (Mail) or a plain miss. Only worth
			// a new report line where the row actually depends on Bulbapedia.
			if r.Gen == nil || r.Holdable || r.Source == "bulbapedia" {
				unresolved = append(unresolved, unresolvedf("item %d (%s): no usable Bulbapedia page %q", r.ID, r.Name, title))
			} else {
				missingMinor++
			}
			continue
		}
		for _, c := range info.UnknownCodes {
			unknownCodes[c] = true
		}

		if r.Gen == nil && info.Gen != nil {
			g := *info.Gen
			r.Gen = &g
		}

		have := map[string]bool{}
		for _, d := range r.Descriptions {
			have[d.Version] = true
		}
		games := make([]string, 0, len(info.Descriptions))
		for g := range info.Descriptions {
			games = append(games, g)
		}
		sort.Strings(games)
		for _, g := range games {
			if have[g] {
				continue
			}
			r.Descriptions = append(r.Descriptions, descriptionRow{
				Entity: entityItem, ID: r.ID, Version: g, Text: info.Descriptions[g],
				Source: "bulbapedia", SourceTitle: usedTitle,
			})
		}
		sort.Slice(r.Descriptions, func(a, b int) bool { return r.Descriptions[a].Version < r.Descriptions[b].Version })
	}

	if missingMinor > 0 {
		unresolved = append(unresolved, unresolvedItem{
			Message: fmt.Sprintf("%d items have no usable Bulbapedia page under their display name", missingMinor),
			Known:   true,
			Reason:  "not holdable and already dated by PokeAPI; PokeAPI data is used as is",
		})
	}
	if len(unknownCodes) > 0 {
		codes := make([]string, 0, len(unknownCodes))
		for c := range unknownCodes {
			codes = append(codes, c)
		}
		sort.Strings(codes)
		unresolved = append(unresolved, unresolvedf("unmapped Bulbapedia game codes (add to bulbapediaGameCodes or bulbapediaIgnoredCodes): %s", strings.Join(codes, ", ")))
	}
	return unresolved, nil
}

func (i bulbapediaItemInfo) isEmpty() bool {
	return i.Gen == nil && len(i.Descriptions) == 0
}

func uniqueStrings(in []string) []string {
	seen := make(map[string]bool, len(in))
	var out []string
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
