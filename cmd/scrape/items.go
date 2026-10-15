package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// showdownItemsJSON is pokepedia-battle's `npm run dumpitems` output: every
// Showdown item with the battle formats it is legal in. Joining it onto the
// PokeAPI items is what makes the team builder's item picker agree with the
// battle server's validation.
//
//go:embed data/showdown-items.json
var showdownItemsJSON []byte

type showdownItem struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Gen          int      `json:"gen"`
	LegalFormats []string `json:"legalFormats"`
}

func loadShowdownItems() (map[string]showdownItem, error) {
	var list []showdownItem
	if err := json.Unmarshal(showdownItemsJSON, &list); err != nil {
		return nil, fmt.Errorf("decode showdown-items.json: %w", err)
	}
	out := make(map[string]showdownItem, len(list))
	for _, it := range list {
		out[it.ID] = it
	}
	return out, nil
}

// toShowdownID mirrors Showdown's toID: lowercase, alphanumerics only. The
// only non-ASCII letter in item names is the e-acute in "Poké Ball"-style
// names, which Showdown spells without the accent.
func toShowdownID(s string) string {
	s = strings.NewReplacer("é", "e", "É", "e").Replace(s)
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// itemVersionGroupGames maps a PokeAPI version_group slug to the public.game
// values sharing its flavor text. Unlike versionGroupGames (pokeapi.go) this
// covers every group, because the item tables start empty. Colosseum and XD
// have no public.game member and are omitted.
var itemVersionGroupGames = map[string][]string{
	"red-blue":                            {"red", "blue"},
	"yellow":                              {"yellow"},
	"gold-silver":                         {"gold", "silver"},
	"crystal":                             {"crystal"},
	"ruby-sapphire":                       {"ruby", "sapphire"},
	"emerald":                             {"emerald"},
	"firered-leafgreen":                   {"firered", "leafgreen"},
	"diamond-pearl":                       {"diamond", "pearl"},
	"platinum":                            {"platinum"},
	"heartgold-soulsilver":                {"heartgold", "soulsilver"},
	"black-white":                         {"black", "white"},
	"black-2-white-2":                     {"black-2", "white-2"},
	"x-y":                                 {"x", "y"},
	"omega-ruby-alpha-sapphire":           {"omega-ruby", "alpha-sapphire"},
	"sun-moon":                            {"sun", "moon"},
	"ultra-sun-ultra-moon":                {"ultra-sun", "ultra-moon"},
	"lets-go-pikachu-lets-go-eevee":       {"lets-go-pikachu", "lets-go-eevee"},
	"sword-shield":                        {"sword", "shield"},
	"brilliant-diamond-and-shining-pearl": {"brilliant-diamond", "shining-pearl"},
	"legends-arceus":                      {"legends-arceus"},
	"scarlet-violet":                      {"scarlet", "violet"},
	"legends-z-a":                         {"legends-za"},
}

// itemDLCVersionGroups are folded onto their base game, per the convention in
// 000003: DLC is a Pokedex dimension, never a description dimension. They are
// applied after the base groups so a base-game entry always wins.
var itemDLCVersionGroups = map[string][]string{
	"the-isle-of-armor": {"sword", "shield"},
	"the-crown-tundra":  {"sword", "shield"},
	"the-teal-mask":     {"scarlet", "violet"},
	"the-indigo-disk":   {"scarlet", "violet"},
}

// validGames is the public.game enum, for filtering held-by version names
// (PokeAPI's held_by_pokemon also lists Colosseum and XD).
var validGames = func() map[string]bool {
	m := map[string]bool{}
	for _, games := range itemVersionGroupGames {
		for _, g := range games {
			m[g] = true
		}
	}
	return m
}()

// heldItemCategories are the PokeAPI item categories whose members are held
// in battle. PokeAPI's own `holdable` attribute is not usable for this: it is
// set on Leftovers and Choice Scarf but missing on Rocky Helmet, Assault Vest,
// Eviolite, every Mega Stone and every Z-Crystal, and set on Poke Balls that
// nobody holds in battle. The category is what reliably says "held item".
var heldItemCategories = map[string]bool{
	"bad-held-items":   true,
	"choice":           true,
	"effort-training":  true,
	"healing":          true,
	"held-items":       true,
	"in-a-pinch":       true,
	"jewels":           true,
	"medicine":         true, // Sitrus/Lum/Oran berries live here
	"mega-stones":      true,
	"memories":         true,
	"other":            true, // Kee/Maranga/Enigma/Rowap/Jaboca berries
	"picky-healing":    true,
	"plates":           true,
	"plot-advancement": true, // Red/Blue Orb (Primal Reversion)
	"species-specific": true,
	"type-enhancement": true,
	"type-protection":  true,
	"unused":           true, // only the "--held" Z-Crystal variants; see isHoldableItem
	"z-crystals":       true,
}

// isHoldableItem reports whether an item belongs in the team builder picker:
// it has to be in a held-item category. The matching Showdown id is checked
// by the caller, which is what makes it battle-legal.
func isHoldableItem(slug, category string) bool {
	if category == "unused" {
		return strings.HasSuffix(slug, "--held")
	}
	return heldItemCategories[category]
}

// versionGroupGen is the generation a version group belongs to, used to date
// an item whose game_indices are empty (everything new in Legends: Z-A) by its
// earliest flavor text instead. Let's Go counts as 7 and Legends: Arceus as 8,
// matching how the rest of this database numbers them.
var versionGroupGen = map[string]int32{
	"red-blue": 1, "yellow": 1,
	"gold-silver": 2, "crystal": 2,
	"ruby-sapphire": 3, "emerald": 3, "firered-leafgreen": 3,
	"diamond-pearl": 4, "platinum": 4, "heartgold-soulsilver": 4,
	"black-white": 5, "black-2-white-2": 5,
	"x-y": 6, "omega-ruby-alpha-sapphire": 6,
	"sun-moon": 7, "ultra-sun-ultra-moon": 7, "lets-go-pikachu-lets-go-eevee": 7,
	"sword-shield": 8, "brilliant-diamond-and-shining-pearl": 8, "legends-arceus": 8,
	"scarlet-violet": 9, "legends-z-a": 9,
}

var machineID = regexp.MustCompile(`^t[mr]\d+$`)

var romanGenerations = map[string]int32{
	"generation-i": 1, "generation-ii": 2, "generation-iii": 3,
	"generation-iv": 4, "generation-v": 5, "generation-vi": 6,
	"generation-vii": 7, "generation-viii": 8, "generation-ix": 9,
}

type pokeAPINamed struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type pokeAPIItemResp struct {
	ID          int32          `json:"id"`
	Name        string         `json:"name"`
	Cost        int32          `json:"cost"`
	FlingPower  *int32         `json:"fling_power"`
	FlingEffect *pokeAPINamed  `json:"fling_effect"`
	Attributes  []pokeAPINamed `json:"attributes"`
	Category    pokeAPINamed   `json:"category"`
	Names       []pokeAPIName  `json:"names"`
	Sprites     struct {
		Default *string `json:"default"`
	} `json:"sprites"`
	EffectEntries []struct {
		Effect      string       `json:"effect"`
		ShortEffect string       `json:"short_effect"`
		Language    pokeAPINamed `json:"language"`
	} `json:"effect_entries"`
	FlavorTextEntries []struct {
		Text         string       `json:"text"`
		Language     pokeAPINamed `json:"language"`
		VersionGroup pokeAPINamed `json:"version_group"`
	} `json:"flavor_text_entries"`
	GameIndices []struct {
		Generation pokeAPINamed `json:"generation"`
	} `json:"game_indices"`
	HeldByPokemon []struct {
		Pokemon        pokeAPINamed `json:"pokemon"`
		VersionDetails []struct {
			Rarity  int32        `json:"rarity"`
			Version pokeAPINamed `json:"version"`
		} `json:"version_details"`
	} `json:"held_by_pokemon"`
}

type pokeAPIName struct {
	Name     string       `json:"name"`
	Language pokeAPINamed `json:"language"`
}

type pokeAPIItemCategoryResp struct {
	Pocket pokeAPINamed `json:"pocket"`
}

// scrapeItems builds an itemRow for every PokeAPI item outside the machines
// pocket and joins the Showdown dump onto it. Items are fetched by numeric id
// from a cached listing (ids are not contiguous). The only database read is
// the pokemon id set, used to drop held-by rows for pokemon this database
// does not have, since itemheldby has a foreign key to pokemon.
func scrapeItems(ctx context.Context, pool *pgxpool.Pool, limit int, cacheDir string, bulbapedia bool, batchSize int) ([]itemRow, []unresolvedItem, error) {
	cache, err := newDiskCache(cacheDir + "/pokeapi")
	if err != nil {
		return nil, nil, err
	}
	showdown, err := loadShowdownItems()
	if err != nil {
		return nil, nil, err
	}
	pokemon, err := fetchEntities(ctx, pool, pokemonQuery)
	if err != nil {
		return nil, nil, err
	}
	knownPokemon := make(map[int32]bool, len(pokemon))
	for _, p := range pokemon {
		knownPokemon[p.ID] = true
	}

	ids, err := listPokeAPIItemIDs(cache)
	if err != nil {
		return nil, nil, err
	}
	if limit > 0 && len(ids) > limit {
		ids = ids[:limit]
	}

	var rows []itemRow
	var unresolved []unresolvedItem
	usedShowdown := map[string]bool{}
	slugOwner := map[string]int32{}
	matchedShowdown := map[string]bool{}

	for _, id := range ids {
		body, err := fetchPokeAPIRaw(cache, "item", id)
		if err != nil {
			return nil, nil, fmt.Errorf("item %d: %w", id, err)
		}
		var resp pokeAPIItemResp
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, nil, fmt.Errorf("decode item-%d: %w", id, err)
		}

		pocket, err := itemPocket(cache, resp.Category.Name)
		if err != nil {
			return nil, nil, fmt.Errorf("item %d (%s): %w", id, resp.Name, err)
		}
		if pocket == "machines" {
			continue
		}

		// PokeAPI stores each Z-Crystal twice with the same English name: a
		// "--held" item and a "--bag" duplicate in the unused category. Keep
		// the held one only, so the Showdown id can't land on the wrong row.
		if strings.HasSuffix(resp.Name, "--bag") {
			continue
		}

		row := buildItemRow(resp, pocket, knownPokemon)

		// PokeAPI occasionally carries two items under one slug (the Legends
		// Z-A re-add of Roseli Berry). /item/{name} must resolve to exactly one
		// row, and the lower id is the established one, so keep that.
		if first, dup := slugOwner[row.Name]; dup {
			unresolved = append(unresolved, unresolvedItem{
				Message: fmt.Sprintf("item %d (%s): slug already used by item %d; skipped", id, row.Name, first),
				Known:   true,
				Reason:  "PokeAPI duplicate slug; the lower id is kept",
			})
			continue
		}
		slugOwner[row.Name] = id

		sd, ok := matchShowdownItem(resp, showdown)
		if ok && usedShowdown[sd.ID] {
			unresolved = append(unresolved, unresolvedf("item %d (%s): Showdown id %q already taken by another item; left unmatched", id, resp.Name, sd.ID))
			ok = false
		}
		if ok {
			usedShowdown[sd.ID] = true
			matchedShowdown[sd.ID] = true
			sdID := sd.ID
			row.ShowdownID = &sdID
			row.LegalFormats = sd.LegalFormats
			// holdable means it is a held-item category AND Showdown models it.
			// A Showdown item outside those categories (evolution stones,
			// fossils, Poke Balls, inert berries) stays off the picker; the
			// report counts those as known gaps.
			row.Holdable = isHoldableItem(resp.Name, resp.Category.Name)
			if !row.Holdable {
				unresolved = append(unresolved, unresolvedItem{
					Message: fmt.Sprintf("item %d (%s): matches Showdown %q but category %q is not a held-item category", id, resp.Name, sd.ID, resp.Category.Name),
					Known:   true,
					Reason:  "Showdown models it but it isn't held in battle (evolution stones, fossils, Poke Balls, inert berries); kept out of the team builder picker",
				})
			}
		}
		rows = append(rows, row)
	}

	// Showdown items with no PokeAPI row at all. With Bulbapedia enabled they
	// become rows with ids from bulbapediaOnlyItemIDs; a Showdown item missing
	// from that table is reported so the table gets extended deliberately.
	namesTaken := make(map[string]int32, len(rows))
	for _, r := range rows {
		namesTaken[r.Name] = r.ID
	}
	var missing []string
	for id := range showdown {
		// TM/TR items are machines, which this pass skips on purpose.
		if !matchedShowdown[id] && !machineID.MatchString(id) {
			missing = append(missing, id)
		}
	}
	sort.Strings(missing)
	for _, id := range missing {
		itemID, assigned := bulbapediaOnlyItemIDs[id]
		if bulbapedia && assigned && limit == 0 {
			row := newBulbapediaOnlyRow(itemID, showdown[id])
			if clash, taken := namesTaken[row.Name]; taken {
				unresolved = append(unresolved, unresolvedf("showdown item %q: slug %q already used by PokeAPI item %d; add it to bulbapediaOnlyItemSlugs", id, row.Name, clash))
				continue
			}
			rows = append(rows, row)
			continue
		}
		unresolved = append(unresolved, unresolvedf("showdown item %q (%s) has no PokeAPI item", id, showdown[id].Name))
	}

	if bulbapedia {
		u, err := supplementItemsFromBulbapedia(rows, cacheDir, batchSize)
		if err != nil {
			return nil, nil, fmt.Errorf("bulbapedia item supplement: %w", err)
		}
		unresolved = append(unresolved, u...)
	}

	// Reported after the Bulbapedia pass so only items still undated count.
	for _, r := range rows {
		if r.Gen == nil {
			unresolved = append(unresolved, unresolvedf("item %d (%s): no generation from PokeAPI or Bulbapedia", r.ID, r.Name))
		}
	}

	return rows, unresolved, nil
}

// itemDisplayName is the name shown to users: englishItemName, except for
// the renamed Legends: Arceus balls.
func itemDisplayName(resp pokeAPIItemResp) string {
	if name, ok := legendsArceusBallNames[resp.Name]; ok {
		return name
	}
	return englishItemName(resp)
}

// englishItemName is PokeAPI's English display name, the string a Bulbapedia
// title is built from (slugs don't title-case cleanly: "king-s-rock").
func englishItemName(resp pokeAPIItemResp) string {
	for _, n := range resp.Names {
		if n.Language.Name == "en" {
			// PokeAPI writes King’s Rock with a curly apostrophe; Showdown
			// and Bulbapedia use a straight one.
			return titleCharReplacements.Replace(n.Name)
		}
	}
	// Items PokeAPI added without any names yet (Hopo Berry, Legends: Z-A
	// stones) fall back to a title-cased slug, which matches Bulbapedia's
	// title for plain names.
	words := strings.Split(resp.Name, "-")
	for i, w := range words {
		if w != "" {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

// legendsArceusBallNames renames PokeAPI's Legends: Arceus balls. Five of them
// share display names with the mainline balls ("Poke Ball" twice in the list,
// "Great Ball" twice, ...), so all eleven carry an "LA " prefix, the ones with
// unique names too, so the whole family reads and sorts together. The mainline
// versions keep the plain names.
var legendsArceusBallNames = map[string]string{
	"lapoke-ball":    "LA Pok\u00e9 Ball",
	"lagreat-ball":   "LA Great Ball",
	"laultra-ball":   "LA Ultra Ball",
	"laheavy-ball":   "LA Heavy Ball",
	"lastrange-ball": "LA Strange Ball",
	"lafeather-ball": "LA Feather Ball",
	"lawing-ball":    "LA Wing Ball",
	"lajet-ball":     "LA Jet Ball",
	"laleaden-ball":  "LA Leaden Ball",
	"lagigaton-ball": "LA Gigaton Ball",
	"laorigin-ball":  "LA Origin Ball",
}

const legendsArceusPrefix = "LA "

func buildItemRow(resp pokeAPIItemResp, pocket string, knownPokemon map[int32]bool) itemRow {
	row := itemRow{
		ID:           resp.ID,
		Name:         resp.Name,
		Category:     resp.Category.Name,
		Pocket:       pocket,
		Cost:         resp.Cost,
		DisplayName:  itemDisplayName(resp),
		FlingPower:   resp.FlingPower,
		LegalFormats: []string{},
		Source:       "pokeapi",
	}
	if resp.FlingEffect != nil {
		n := resp.FlingEffect.Name
		row.FlingEffect = &n
	}
	for _, e := range resp.EffectEntries {
		if e.Language.Name != "en" {
			continue
		}
		if eff := cleanFlavorText(e.Effect); eff != "" {
			row.Effect = &eff
		}
		if short := cleanFlavorText(e.ShortEffect); short != "" {
			row.ShortEffect = &short
		}
		break
	}
	// The key is a naming convention, set for every item: PokeAPI ships a
	// sprite for only about half of them and fetch-item-images.sh fills more
	// from Bulbapedia, so the file may not exist. See 000015 and 000019.
	key := "items/" + resp.Name + ".png"
	row.Image = &key
	for _, gi := range resp.GameIndices {
		if g, ok := romanGenerations[gi.Generation.Name]; ok && (row.Gen == nil || g < *row.Gen) {
			g := g
			row.Gen = &g
		}
	}
	if row.Gen == nil {
		for _, e := range resp.FlavorTextEntries {
			if g, ok := versionGroupGen[e.VersionGroup.Name]; ok && e.Language.Name == "en" && (row.Gen == nil || g < *row.Gen) {
				g := g
				row.Gen = &g
			}
		}
	}
	row.Descriptions = itemDescriptions(resp)
	row.HeldBy = itemHeldBy(resp, knownPokemon)
	return row
}

// itemDescriptions turns flavor text into one row per (item, game). Base
// version groups are processed before the DLC ones so the DLC copy only fills
// gaps -- (item_id, version) is the primary key.
func itemDescriptions(resp pokeAPIItemResp) []descriptionRow {
	var rows []descriptionRow
	seen := map[string]bool{}
	for _, groups := range []map[string][]string{itemVersionGroupGames, itemDLCVersionGroups} {
		for _, e := range resp.FlavorTextEntries {
			if e.Language.Name != "en" {
				continue
			}
			games, ok := groups[e.VersionGroup.Name]
			if !ok {
				continue
			}
			text := cleanFlavorText(e.Text)
			if text == "" {
				continue
			}
			for _, g := range games {
				if seen[g] {
					continue
				}
				seen[g] = true
				rows = append(rows, descriptionRow{
					Entity: entityItem, ID: resp.ID, Version: g, Text: text, Source: "pokeapi",
				})
			}
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Version < rows[j].Version })
	return rows
}

func itemHeldBy(resp pokeAPIItemResp, knownPokemon map[int32]bool) []itemHeldByRow {
	var rows []itemHeldByRow
	for _, h := range resp.HeldByPokemon {
		pid, ok := idFromPokeAPIURL(h.Pokemon.URL)
		if !ok || !knownPokemon[pid] {
			continue
		}
		for _, v := range h.VersionDetails {
			if !validGames[v.Version.Name] {
				continue
			}
			rows = append(rows, itemHeldByRow{
				ItemID: resp.ID, PokemonID: pid, Version: v.Version.Name, Rarity: v.Rarity,
			})
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].PokemonID != rows[j].PokemonID {
			return rows[i].PokemonID < rows[j].PokemonID
		}
		return rows[i].Version < rows[j].Version
	})
	return rows
}

// matchShowdownItem tries the English display name first ("Choice Scarf"),
// then the slug ("choice-scarf"); both normalise to Showdown's id.
func matchShowdownItem(resp pokeAPIItemResp, showdown map[string]showdownItem) (showdownItem, bool) {
	for _, n := range resp.Names {
		if n.Language.Name == "en" {
			if sd, ok := showdown[toShowdownID(n.Name)]; ok {
				return sd, true
			}
		}
	}
	sd, ok := showdown[toShowdownID(resp.Name)]
	return sd, ok
}

// listPokeAPIItemIDs returns every item id PokeAPI serves, ascending.
func listPokeAPIItemIDs(cache *diskCache) ([]int32, error) {
	const key = "item-list.json"
	body, ok := cache.get(key)
	if !ok {
		var err error
		body, err = httpGet("https://pokeapi.co/api/v2/item?limit=10000")
		if err != nil {
			return nil, err
		}
		if err := cache.put(key, body); err != nil {
			return nil, err
		}
	}
	var list struct {
		Results []pokeAPINamed `json:"results"`
	}
	if err := json.Unmarshal(body, &list); err != nil {
		return nil, fmt.Errorf("decode item list: %w", err)
	}
	var ids []int32
	for _, r := range list.Results {
		id, ok := idFromPokeAPIURL(r.URL)
		if !ok {
			return nil, fmt.Errorf("item list: unparseable url %q", r.URL)
		}
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids, nil
}

// itemPocket resolves an item category slug to its pocket slug via
// /item-category/<name>, cached per category.
func itemPocket(cache *diskCache, category string) (string, error) {
	key := "item-category-" + category + ".json"
	body, ok := cache.get(key)
	if !ok {
		var err error
		body, err = httpGet("https://pokeapi.co/api/v2/item-category/" + category + "/")
		if err != nil {
			return "", err
		}
		if err := cache.put(key, body); err != nil {
			return "", err
		}
	}
	var resp pokeAPIItemCategoryResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("decode item-category %s: %w", category, err)
	}
	return resp.Pocket.Name, nil
}

// idFromPokeAPIURL extracts the trailing numeric id from a PokeAPI resource
// url such as https://pokeapi.co/api/v2/pokemon/6/.
func idFromPokeAPIURL(url string) (int32, bool) {
	parts := strings.Split(strings.Trim(url, "/"), "/")
	if len(parts) == 0 {
		return 0, false
	}
	n, err := strconv.ParseInt(parts[len(parts)-1], 10, 32)
	if err != nil {
		return 0, false
	}
	return int32(n), true
}
