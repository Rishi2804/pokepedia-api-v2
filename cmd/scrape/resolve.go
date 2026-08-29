package main

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// pokemonRow is a pokemon table row plus its species grouping, used to build
// the species -> candidate-forms index that resolveFormMarker matches
// {{Dex/Form|...}} markers against.
type pokemonRow struct {
	ID        int32
	Name      string // DB slug, e.g. "charizard-mega-x"
	SpeciesID int32
}

func fetchPokemonRows(ctx context.Context, pool *pgxpool.Pool) ([]pokemonRow, error) {
	rows, err := pool.Query(ctx, `SELECT id, name, species_id FROM pokemon ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []pokemonRow
	for rows.Next() {
		var r pokemonRow
		if err := rows.Scan(&r.ID, &r.Name, &r.SpeciesID); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// speciesIndex groups pokemon rows by species_id for form resolution.
//
// Root is the species' slug root used to compute each form's distinguishing
// suffix ("charizard-mega-x" minus root "charizard" leaves {mega, x}). It is
// NOT simply the row where id == species_id: cmd/indexer's own popularity
// query documents 28+ species (Deoxys, Giratina, Aegislash, Minior, and --
// discovered scraping live pages -- also Morpeko, Darmanitan, Zacian's
// sibling Zamazenta) that have no plain-named row at all, only named ones
// (deoxys-normal, morpeko-full-belly, ...). For those, Root is instead the
// longest common hyphen-prefix shared by every sibling form, which equals
// the species' display name either way and needs no literal matching row to
// compute. LiteralBase is populated only when such a row actually exists,
// and is used solely for the "marker names nothing extra -> the base form"
// shortcut in resolveFormMarker.
type speciesIndex struct {
	root        map[int32]string       // species_id -> root slug, e.g. "morpeko", "darmanitan"
	literalBase map[int32]pokemonRow   // species_id -> row whose Name == root, if one exists
	forms       map[int32][]pokemonRow // species_id -> every pokemon row for it
}

func buildSpeciesIndex(rows []pokemonRow) speciesIndex {
	idx := speciesIndex{
		root:        map[int32]string{},
		literalBase: map[int32]pokemonRow{},
		forms:       map[int32][]pokemonRow{},
	}
	for _, r := range rows {
		idx.forms[r.SpeciesID] = append(idx.forms[r.SpeciesID], r)
	}
	for sid, forms := range idx.forms {
		root := speciesRootSlug(forms)
		idx.root[sid] = root
		for _, f := range forms {
			if f.Name == root {
				idx.literalBase[sid] = f
			}
		}
	}
	return idx
}

// speciesRootSlug is the longest hyphen-segment prefix common to every
// sibling form's slug. DB slugs always put the species name first
// ("charizard-mega-x", "morpeko-full-belly"), so this recovers the species'
// own slug whether or not any row is literally named just that.
func speciesRootSlug(forms []pokemonRow) string {
	if len(forms) == 0 {
		return ""
	}
	words := strings.Split(forms[0].Name, "-")
	for _, f := range forms[1:] {
		fw := strings.Split(f.Name, "-")
		n := 0
		for n < len(words) && n < len(fw) && words[n] == fw[n] {
			n++
		}
		words = words[:n]
	}
	return strings.Join(words, "-")
}

// formSynonyms translates Bulbapedia's natural-language form adjectives into
// the single-word vocabulary the DB slugs already use, so token overlap (see
// resolveFormMarker) can match "Alolan Form" against "vulpix-alola" and
// "Gigantamax" against "charizard-gmax" without a lookup table keyed on the
// full marker string. Filler words map to "" and are dropped.
var formSynonyms = map[string]string{
	"alolan": "alola", "galarian": "galar", "hisuian": "hisui", "paldean": "paldea",
	"gigantamax": "gmax", "form": "", "forme": "",
}

var wordSplitRe = regexp.MustCompile(`[^a-z0-9%]+`)
var apostropheRe = regexp.MustCompile("['\u2019]")

func tokenize(s string) []string {
	s = strings.ToLower(s)
	// Drop apostrophes rather than treat them as a word boundary: DB slugs
	// do the same (farfetchd, not farfetch-d), and Bulbapedia's "Pa'u Style"
	// marker must tokenize to "pau" -- matching oricorio-pau's suffix -- not
	// split into the two meaningless fragments "pa" and "u".
	s = apostropheRe.ReplaceAllString(s, "")
	var words []string
	for _, w := range wordSplitRe.Split(s, -1) {
		if w == "" {
			continue
		}
		if repl, ok := formSynonyms[w]; ok {
			if repl == "" {
				continue
			}
			w = repl
		}
		w = strings.TrimSuffix(w, "%")
		words = append(words, w)
	}
	return words
}

func tokenSet(words []string) map[string]bool {
	set := make(map[string]bool, len(words))
	for _, w := range words {
		set[w] = true
	}
	return set
}

// suffixTokens is the set of slug words in a form's name that are not also
// in its species' root -- "charizard-mega-x" minus root "charizard" leaves
// {mega, x}, which is exactly what disambiguates it from "charizard-mega-y"
// ({mega, y}) when scored against a marker's token set.
func suffixTokens(form pokemonRow, rootWords map[string]bool) map[string]bool {
	var suffix []string
	for _, w := range strings.Split(form.Name, "-") {
		if !rootWords[w] {
			suffix = append(suffix, w)
		}
	}
	return tokenSet(suffix)
}

// knownBaseLabels are free-text {{Dex/Form|...}} markers that name a
// species' base form using vocabulary with no counterpart anywhere in its
// DB slug -- "Hero of Many Battles" for Zacian/Zamazenta, whose OTHER form
// ("Crowned Sword"/"Crowned Shield") already resolves fine by ordinary
// token overlap. Kept as a small, explicit, reviewable table rather than a
// blanket "nothing matched -> assume base" rule, which would silently
// misattribute species like Sinistea, where an unmatched marker ("Antique
// Form") names a state this database does not model as a separate row at
// all -- there the right outcome is staying unresolved, not a guess.
var knownBaseLabels = map[string]bool{
	"hero of many battles": true, // Zacian/Zamazenta
	"zero form":            true, // Palafin
	"teal mask":            true, // Ogerpon
	"hoopa confined":       true, // Hoopa
	"chest form":           true, // Gimmighoul
	"normal form":          true, // Terapagos
	"normal":               true, // Castform ("Normal" alone). Verified safe: the only
	// "-normal" suffix anywhere in this database is deoxys-normal, and Deoxys
	// has no literal base row (see speciesRootSlug), so this entry never
	// fires for it -- the knownBaseLabels lookup is gated on a literal base
	// existing, which only Castform among "normal"-marker species has.
	"male": true, // Oinkologne (unmarked default sex; verified the only other
	// species with a "-male"/"-female" suffix -- Basculegion, Indeedee,
	// Meowstic -- have no literal base row either, so they resolve their own
	// "Male"/"Female" markers by ordinary token overlap instead, unaffected
	// by this entry.
}

// resolveFormMarker matches a {{Dex/Form|<marker>}} label against the
// species' known forms.
//
// Bulbapedia's marker text has no single convention -- Mega forms spell out
// the full name ("Mega Clefable"), regional forms are generic ("Alolan
// Form"), Rotom appliances give the adjective before the species name
// ("Heat Rotom", the reverse of this project's own FormatName order), and
// fusions sometimes drop the species name entirely ("Dusk Mane" for
// Necrozma). A single fixed template cannot cover all of these, so this
// matches by token overlap against each candidate's species-relative
// suffix (see suffixTokens) instead of any specific naming pattern -- it was
// verified against Mega Charizard X/Y, Alolan/Galarian Form, Gigantamax,
// Zygarde's 10%/50%/Complete Forme, Rotom's five appliances, and Necrozma's
// Dusk Mane/Dawn Wings/Ultra Necrozma, all pulled from live Bulbapedia pages.
//
// An empty marker token set (after removing the species' root and filler
// words like "Form") means the marker names the base species itself, as
// Bulbapedia does when a later dex generation's form list starts over.
// resolveFormMarker returns every pokemon id a marker resolves to -- almost
// always exactly one, but see the tied-full-coverage branch below for the
// genuine exception.
func resolveFormMarker(marker string, speciesID int32, idx speciesIndex) (ids []int32, ok bool) {
	root, hasRoot := idx.root[speciesID]
	if !hasRoot {
		return nil, false
	}
	rootWords := tokenSet(strings.Split(root, "-"))
	forms := idx.forms[speciesID]

	trimmed := strings.ToLower(strings.TrimSpace(marker))
	if knownBaseLabels[trimmed] {
		if base, ok := idx.literalBase[speciesID]; ok {
			return []int32{base.ID}, true
		}
	}
	if target, ok := breedingFormOverrides[knownGapKey{speciesID, marker}]; ok {
		for _, form := range forms {
			if form.Name == target {
				return []int32{form.ID}, true
			}
		}
	}

	markerTokens := tokenSet(tokenize(marker))
	filtered := map[string]bool{}
	for t := range markerTokens {
		if !rootWords[t] {
			filtered[t] = true
		}
	}
	markerTokens = filtered

	if len(markerTokens) == 0 {
		if base, ok := idx.literalBase[speciesID]; ok {
			return []int32{base.ID}, true
		}
		return nil, false
	}

	// matched is how many marker tokens the candidate's suffix accounts for;
	// extra is how many of the candidate's own suffix tokens the marker
	// never mentioned. Ranking on matched alone ties whenever one candidate
	// is a superset of another that also fits -- "Mega Garchomp" scores 1
	// against both garchomp-mega ({mega}, extra 0) and garchomp-mega-z
	// ({mega, z}, extra 1) since neither "z" nor its absence is otherwise
	// weighed. Preferring lower extra as the tiebreak picks the candidate
	// the marker didn't leave anything unexplained about.
	type scored struct {
		id             int32
		matched, extra int
	}
	better := func(a, b scored) bool {
		if a.matched != b.matched {
			return a.matched > b.matched
		}
		return a.extra < b.extra
	}

	var all []scored
	var best, second scored
	for _, form := range forms {
		sfx := suffixTokens(form, rootWords)
		matched := 0
		for t := range markerTokens {
			if sfx[t] {
				matched++
			}
		}
		if matched == 0 {
			continue
		}
		extra := 0
		for t := range sfx {
			if !markerTokens[t] {
				extra++
			}
		}
		cand := scored{form.ID, matched, extra}
		all = append(all, cand)
		if better(cand, best) {
			second = best
			best = cand
		} else if better(cand, second) {
			second = cand
		}
	}

	if best.matched > 0 && better(best, second) {
		return []int32{best.id}, true
	}

	// A genuine tie (equal matched AND equal extra), as with Ogerpon's three
	// "...Mask" forms all scoring 1 against "Teal Mask" alone, is usually
	// unresolved rather than force-fit -- UNLESS every tied candidate fully
	// explains its own suffix using only marker tokens (extra == 0), which
	// only happens when the marker explicitly names each of them. Verified
	// live: Basculin's breeding header "Red-Striped/Blue-Striped Basculin"
	// scores {red,striped} and {blue,striped} both at matched=2/extra=0 --
	// it names both forms outright, unlike Ogerpon's "Mask" naming neither
	// in full (each mask form has extra=1: its own color word is never
	// mentioned), so that case still correctly falls through below.
	if best.matched > 0 && best.extra == 0 {
		var tied []int32
		for _, c := range all {
			if c.matched == best.matched && c.extra == 0 {
				tied = append(tied, c.id)
			}
		}
		if len(tied) >= 2 {
			return tied, true
		}
	}

	// Nothing scored. If the species has exactly one pokemon row at all,
	// the marker -- however it's worded -- can only mean that row: Bulbapedia
	// narrates states (Silvally's 18 memory types, Unown's 28 letters,
	// Sinistea/Polteageist's Phony/Antique) that this database does not
	// model as separate rows, so there is nothing else it could resolve to.
	if len(forms) == 1 {
		return []int32{forms[0].ID}, true
	}

	return nil, false
}

// versionNameToGame maps a Dex/Entry "v=" value to the public.game enum. All
// 39 current enum members are listed even though many core-series games
// never need a lookup miss reported. Side-game/spin-off values Bulbapedia
// also uses on the same pages (Stadium, Colosseum, XD, Pokemon Battle
// Revolution, Champions, and any future one like "Pokopia") are deliberately
// absent -- they are counted as skipped, not unresolved, since this project
// has no game enum member for them and is not meant to (Champions was
// excluded by an explicit decision; the others are non-core-series).
var versionNameToGame = map[string]string{
	"Red": "red", "Blue": "blue", "Yellow": "yellow",
	"Gold": "gold", "Silver": "silver", "Crystal": "crystal",
	"Ruby": "ruby", "Sapphire": "sapphire", "Emerald": "emerald",
	"FireRed": "firered", "LeafGreen": "leafgreen",
	"Diamond": "diamond", "Pearl": "pearl", "Platinum": "platinum",
	"HeartGold": "heartgold", "SoulSilver": "soulsilver",
	"Black": "black", "White": "white", "Black 2": "black-2", "White 2": "white-2",
	"X": "x", "Y": "y",
	"Omega Ruby": "omega-ruby", "Alpha Sapphire": "alpha-sapphire",
	"Sun": "sun", "Moon": "moon", "Ultra Sun": "ultra-sun", "Ultra Moon": "ultra-moon",
	"Let's Go, Pikachu!": "lets-go-pikachu", "Let's Go, Eevee!": "lets-go-eevee",
	"Sword": "sword", "Shield": "shield",
	"Brilliant Diamond": "brilliant-diamond", "Shining Pearl": "shining-pearl",
	"Legends: Arceus": "legends-arceus",
	"Scarlet":         "scarlet", "Violet": "violet",
	"Legends: Z-A": "legends-za",
}

// titleCharReplacements normalises characters PokeAPI's English name uses
// differently from Bulbapedia's actual page titles. Verified live:
// PokeAPI's move 717 name is "Nature\u2019s Madness" (curly right single
// quote, U+2019); Bulbapedia's page is literally "Nature's Madness (move)"
// with a straight apostrophe. Left uncorrected, every move whose name
// contains an apostrophe (also King's Shield, Land's Wrath, Forest's Curse)
// silently resolves to "page not found" instead of a parser bug -- the
// title looks right until compared byte-for-byte.
var titleCharReplacements = strings.NewReplacer("\u2019", "'")

// buildTitle builds a Bulbapedia page title from a PokeAPI English display
// name. DB slugs (will-o-wisp, ho-oh, mr-mime, nidoran-f) don't title-case
// cleanly, so titles are always derived from the upstream display name --
// see fetchPokeAPIName -- never guessed from the slug.
func buildTitle(displayName, suffix string) string {
	return fmt.Sprintf("%s (%s)", titleCharReplacements.Replace(displayName), suffix)
}

// fetchPokeAPIName returns the English display name PokeAPI has for a move
// or species id -- the string a Bulbapedia title is built from (see
// buildTitle). This is what makes titles correct for slugs that don't
// title-case cleanly: "will-o-wisp" -> "Will-O-Wisp", "ho-oh" -> "Ho-Oh",
// "farfetchd" -> "Farfetch'd", "nidoran-f" -> "Nidoran♀", "type-null" ->
// "Type: Null", "porygon-z" -> "Porygon-Z". Returns "" if PokeAPI has no
// English name, which callers must report rather than guess a title for.
func fetchPokeAPIName(cache *diskCache, kind string, id int32) (string, error) {
	resp, err := fetchPokeAPIEntity(cache, kind, id)
	if err != nil {
		return "", err
	}
	for _, n := range resp.Names {
		if n.Language.Name == "en" {
			return n.Name, nil
		}
	}
	return "", nil
}
