package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const bulbapediaAPI = "https://bulbapedia.bulbagarden.net/w/api.php"

// scrapeBulbapedia backfills: (1) every per-form Pokemon dex entry, for
// every game -- the only source, since PokeAPI's pokemon-form has no
// flavor_text_entries field and pokemon-species returns only the base
// form's text -- and (2) Legends: Z-A move descriptions, since PokeAPI has
// none for that version group yet. It returns both the scraped rows and a
// report of titles/markers it could not resolve, which callers must surface
// rather than silently drop.
func scrapeBulbapedia(ctx context.Context, pool *pgxpool.Pool, types map[string]bool, limit, batchSize int, cacheDir string) ([]descriptionRow, []unresolvedItem, error) {
	pageCache, err := newDiskCache(cacheDir + "/bulbapedia")
	if err != nil {
		return nil, nil, err
	}
	// Same cache directory pokeapi.go writes to, so a move/species id
	// already fetched for its flavor text or by an earlier scrape run is
	// reused here instead of re-fetched under a different key.
	pokeCache, err := newDiskCache(cacheDir + "/pokeapi")
	if err != nil {
		return nil, nil, err
	}

	var rows []descriptionRow
	var unresolved []unresolvedItem

	if types["pokemon"] {
		r, u, err := scrapeBulbapediaPokemon(ctx, pool, pageCache, pokeCache, batchSize, limit)
		if err != nil {
			return nil, nil, err
		}
		rows = append(rows, r...)
		unresolved = append(unresolved, u...)
	}

	if types["move"] {
		r, u, err := scrapeBulbapediaMoves(ctx, pool, pageCache, pokeCache, batchSize, limit)
		if err != nil {
			return nil, nil, err
		}
		rows = append(rows, r...)
		unresolved = append(unresolved, u...)
	}

	return rows, unresolved, nil
}

// scrapeBulbapediaPokemon fetches one page per SPECIES, not per form: a
// form's dex entries live on its species' own page, disambiguated only by
// an in-page {{Dex/Form|...}} marker (see resolveFormMarker) -- fetching
// per-form would just re-fetch and re-parse the same page once per sibling.
func scrapeBulbapediaPokemon(ctx context.Context, pool *pgxpool.Pool, pageCache, pokeCache *diskCache, batchSize, limit int) ([]descriptionRow, []unresolvedItem, error) {
	pokemonRows, err := fetchPokemonRows(ctx, pool)
	if err != nil {
		return nil, nil, err
	}
	idx := buildSpeciesIndex(pokemonRows)

	var speciesIDs []int32
	for sid := range idx.forms {
		speciesIDs = append(speciesIDs, sid)
	}
	sort.Slice(speciesIDs, func(i, j int) bool { return speciesIDs[i] < speciesIDs[j] })
	if limit > 0 && len(speciesIDs) > limit {
		speciesIDs = speciesIDs[:limit]
	}

	var unresolved []unresolvedItem
	titleToSpecies := map[string]int32{}
	var titles []string
	for _, sid := range speciesIDs {
		name, err := fetchPokeAPIName(pokeCache, "pokemon-species", sid)
		if err != nil {
			return nil, nil, fmt.Errorf("species %d name: %w", sid, err)
		}
		if name == "" {
			unresolved = append(unresolved, unresolvedf("species %d: no English PokeAPI name", sid))
			continue
		}
		title := buildTitle(name, "Pokémon")
		titleToSpecies[title] = sid
		titles = append(titles, title)
	}

	pages, err := fetchBulbapediaBatch(pageCache, titles, batchSize)
	if err != nil {
		return nil, nil, err
	}

	var rows []descriptionRow
	for title, sid := range titleToSpecies {
		page := pages[title]
		if page.Missing {
			unresolved = append(unresolved, unresolvedf("page not found: %s (species %d)", title, sid))
			continue
		}
		sec, ok := pokedexEntriesSection(page.Content)
		if !ok {
			unresolved = append(unresolved, unresolvedf("no Pokédex entries section: %s", title))
			continue
		}
		r, u := parsePokemonDexEntries(sec, sid, idx, title)
		rows = append(rows, r...)
		unresolved = append(unresolved, u...)
	}
	return rows, unresolved, nil
}

func scrapeBulbapediaMoves(ctx context.Context, pool *pgxpool.Pool, pageCache, pokeCache *diskCache, batchSize, limit int) ([]descriptionRow, []unresolvedItem, error) {
	moves, err := fetchEntities(ctx, pool, movesQuery)
	if err != nil {
		return nil, nil, err
	}
	moves = capEntities(moves, limit)

	var unresolved []unresolvedItem
	titleToID := map[string]int32{}
	var titles []string
	for _, m := range moves {
		name, err := fetchPokeAPIName(pokeCache, "move", m.ID)
		if err != nil {
			return nil, nil, fmt.Errorf("move %d name: %w", m.ID, err)
		}
		if name == "" {
			unresolved = append(unresolved, unresolvedf("move %d (%s): no English PokeAPI name", m.ID, m.Name))
			continue
		}
		title := buildTitle(name, "move")
		titleToID[title] = m.ID
		titles = append(titles, title)
	}

	pages, err := fetchBulbapediaBatch(pageCache, titles, batchSize)
	if err != nil {
		return nil, nil, err
	}

	var rows []descriptionRow
	for title, id := range titleToID {
		page := pages[title]
		if page.Missing {
			unresolved = append(unresolved, unresolvedf("page not found: %s (move %d)", title, id))
			continue
		}
		text, ok := parseMoveZADescription(page.Content)
		if !ok {
			continue // legitimately not in Legends: Z-A -- not an error
		}
		rows = append(rows, descriptionRow{
			Entity: entityMove, ID: id, Version: "legends-za", Text: text,
			Source: "bulbapedia", SourceTitle: title,
		})
	}
	return rows, unresolved, nil
}

// bulbaPage is what gets cached per requested title: either the page's
// wikitext, or Missing=true if MediaWiki reported no such page (after
// following redirects/normalization) -- cached too, so a known-missing
// title isn't re-queried on every run.
type bulbaPage struct {
	Content string `json:"content"`
	Missing bool   `json:"missing"`
}

// fetchBulbapediaBatch resolves a list of page titles to their wikitext,
// serving from cache where possible and batching the rest into MediaWiki
// requests of at most batchSize titles (MediaWiki allows up to 50; smaller
// batches keep individual responses in the 1-2MB range since pages like
// "Tackle (move)" run over 100KB).
func fetchBulbapediaBatch(cache *diskCache, titles []string, batchSize int) (map[string]bulbaPage, error) {
	result := map[string]bulbaPage{}
	var need []string
	for _, t := range titles {
		if b, ok := cache.get(cacheKeyForTitle(t)); ok {
			var p bulbaPage
			if err := json.Unmarshal(b, &p); err == nil {
				result[t] = p
				continue
			}
		}
		need = append(need, t)
	}

	for i := 0; i < len(need); i += batchSize {
		end := i + batchSize
		if end > len(need) {
			end = len(need)
		}
		chunk := need[i:end]

		pages, err := queryBulbapedia(chunk)
		if err != nil {
			return nil, fmt.Errorf("bulbapedia batch %d-%d: %w", i, end, err)
		}
		for _, t := range chunk {
			p := pages[t] // zero value (Missing: false, Content: "") if somehow absent
			result[t] = p
			if b, err := json.Marshal(p); err == nil {
				_ = cache.put(cacheKeyForTitle(t), b)
			}
		}

		if end < len(need) {
			time.Sleep(500 * time.Millisecond)
		}
	}
	return result, nil
}

func cacheKeyForTitle(title string) string {
	return "page-" + title + ".json"
}

type mediaWikiRedirect struct{ From, To string }

type mediaWikiPage struct {
	Title     string `json:"title"`
	Missing   bool   `json:"missing"`
	Revisions []struct {
		Slots struct {
			Main struct {
				Content string `json:"content"`
			} `json:"main"`
		} `json:"slots"`
	} `json:"revisions"`
}

type mediaWikiResponse struct {
	Query struct {
		Normalized []mediaWikiRedirect `json:"normalized"`
		Redirects  []mediaWikiRedirect `json:"redirects"`
		Pages      []mediaWikiPage     `json:"pages"`
	} `json:"query"`
	Error *struct {
		Code string `json:"code"`
		Info string `json:"info"`
	} `json:"error"`
}

// queryBulbapedia fetches wikitext for up to len(titles) pages in a single
// MediaWiki request (action=query&prop=revisions&rvslots=main, batched --
// verified live: 5 large move pages in one request, no warnings) and maps
// each REQUESTED title back through any normalization/redirect chain to the
// page MediaWiki actually returned.
func queryBulbapedia(titles []string) (map[string]bulbaPage, error) {
	v := url.Values{}
	v.Set("action", "query")
	v.Set("prop", "revisions")
	v.Set("rvprop", "content")
	v.Set("rvslots", "main")
	v.Set("format", "json")
	v.Set("formatversion", "2")
	v.Set("redirects", "1")
	v.Set("maxlag", "5")
	v.Set("titles", strings.Join(titles, "|"))

	body, err := mediaWikiGet(bulbapediaAPI + "?" + v.Encode())
	if err != nil {
		return nil, err
	}

	var resp mediaWikiResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("decode mediawiki response: %w", err)
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("mediawiki error %s: %s", resp.Error.Code, resp.Error.Info)
	}

	byFinalTitle := map[string]bulbaPage{}
	for _, p := range resp.Query.Pages {
		bp := bulbaPage{Missing: p.Missing}
		if len(p.Revisions) > 0 {
			bp.Content = p.Revisions[0].Slots.Main.Content
		}
		byFinalTitle[p.Title] = bp
	}

	out := map[string]bulbaPage{}
	for _, requested := range titles {
		final := requested
		// Chained normalize -> redirect, capped defensively; real chains on
		// this wiki are at most one hop of each.
		for i := 0; i < 5; i++ {
			changed := false
			for _, n := range resp.Query.Normalized {
				if n.From == final {
					final = n.To
					changed = true
				}
			}
			for _, r := range resp.Query.Redirects {
				if r.From == final {
					final = r.To
					changed = true
				}
			}
			if !changed {
				break
			}
		}
		if p, ok := byFinalTitle[final]; ok {
			out[requested] = p
		} else {
			out[requested] = bulbaPage{Missing: true}
		}
	}
	return out, nil
}

// mediaWikiGet issues one GET with a real User-Agent (MediaWiki policy) and
// honours 429/503 with the server's Retry-After header, falling back to
// exponential backoff if none is given -- the old scraper's ~905 concurrent
// unthrottled requests are exactly what this exists to never repeat.
func mediaWikiGet(rawURL string) ([]byte, error) {
	delay := 2 * time.Second
	for attempt := 0; attempt < 6; attempt++ {
		req, err := http.NewRequest(http.MethodGet, rawURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", userAgent)

		resp, err := httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return nil, readErr
		}

		if resp.StatusCode == http.StatusOK {
			return body, nil
		}
		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable {
			wait := delay
			if ra := resp.Header.Get("Retry-After"); ra != "" {
				if secs, err := strconv.Atoi(ra); err == nil {
					wait = time.Duration(secs) * time.Second
				}
			}
			time.Sleep(wait)
			delay *= 2
			continue
		}
		return nil, fmt.Errorf("GET %s: status %d: %s", rawURL, resp.StatusCode, truncate(string(body), 200))
	}
	return nil, fmt.Errorf("exceeded retries: %s", rawURL)
}

// pokedexEntriesHeadingRe tolerates "Pokédex" appearing as a plain word or
// wikilinked -- Greninja's page (and others with an anime-Pokedex subsection
// elsewhere confusing a naive search) heads its game dex-entries section
// "===[[Pokédex]] entries===" rather than "===Pokédex entries===".
var pokedexEntriesHeadingRe = regexp.MustCompile(`(?ims)^===\s*(?:\[\[Pokédex(?:\|[^\]]*)?\]\]|Pokédex)\s+entries\s*===\s*$`)
var nextHeadingAnyLevelRe = regexp.MustCompile(`(?m)^={2,3}[^=]`)

// pokedexEntriesSection extracts the "===Pokédex entries===" subsection
// (level 3, unlike the level-2 sections section() handles) up to the next
// heading of level 2 or 3.
func pokedexEntriesSection(wikitext string) (string, bool) {
	loc := pokedexEntriesHeadingRe.FindStringIndex(wikitext)
	if loc == nil {
		return "", false
	}
	rest := wikitext[loc[1]:]
	next := nextHeadingAnyLevelRe.FindStringIndex(rest)
	if next == nil {
		return rest, true
	}
	return rest[:next[0]], true
}

// dexEntryTemplateNames enumerates Dex/EntryN up to a generous N (the
// highest observed on Bulbapedia's Pokemon pages is Dex/Entry3, for a
// three-version combined entry like Diamond/Pearl/Platinum) plus Dex/Form,
// so findTemplates can walk both families in one linear, document-ordered
// pass.
func dexEntryTemplateNames() []string {
	names := []string{"Dex/Form"}
	for i := 1; i <= 6; i++ {
		names = append(names, fmt.Sprintf("Dex/Entry%d", i))
	}
	return names
}

// parsePokemonDexEntries walks a species' "Pokédex entries" section in
// document order, tracking which form is "current" as {{Dex/Form|...}}
// markers reassign it (see resolveFormMarker) and attributing every
// {{Dex/EntryN|...}} that follows to that form. This positional walk is
// mandatory: entries for different forms share the exact same v= value with
// no per-entry form field (Mega Clefable's Legends: Z-A entry and base
// Clefable's Legends: Z-A entry are both just "v=Legends: Z-A"), so keying
// on v= alone would silently merge a Mega's entry onto its base form.
func parsePokemonDexEntries(section string, speciesID int32, idx speciesIndex, pageTitle string) ([]descriptionRow, []unresolvedItem) {
	// Species with no literal base row (Deoxys, Giratina, Morpeko, Darmanitan,
	// ...) always carry an explicit {{Dex/Form|...}} marker before their
	// very first dex entry -- verified live on Zacian and Morpeko -- so
	// starting unresolved (-1, "skip until a marker resolves") rather than
	// guessing an id is safe, not lossy.
	currentID := int32(-1)
	if base, ok := idx.literalBase[speciesID]; ok {
		currentID = base.ID
	}
	var rows []descriptionRow
	var unresolved []unresolvedItem

	for _, occ := range findTemplates(section, dexEntryTemplateNames()...) {
		if occ.Name == "Dex/Form" {
			marker := strings.TrimSpace(occ.Body)
			if id, ok := resolveFormMarker(marker, speciesID, idx); ok {
				currentID = id
			} else if reason, known := knownGaps[knownGapKey{speciesID, marker}]; known {
				unresolved = append(unresolved, unresolvedItem{
					Message: fmt.Sprintf("form marker %q on %s (species %d)", marker, pageTitle, speciesID),
					Known:   true, Reason: reason,
				})
				currentID = -1
			} else {
				unresolved = append(unresolved, unresolvedf(
					"form marker %q on %s (species %d): no unambiguous match", marker, pageTitle, speciesID))
				currentID = -1 // skip entries until the next marker resolves
			}
			continue
		}

		if currentID < 0 {
			continue
		}
		args := namedArgs(splitTopLevel(occ.Body))
		entryText, hasEntry := args["entry"]
		if !hasEntry {
			continue
		}
		cleaned := cleanWikitext(entryText)

		for _, vkey := range []string{"v", "v2", "v3", "v4", "v5"} {
			vraw, ok := args[vkey]
			if !ok {
				continue
			}
			game, ok := versionNameToGame[strings.TrimSpace(vraw)]
			if !ok {
				continue // known non-core-series value (Stadium, Colosseum, Champions, ...) -- not an error
			}
			rows = append(rows, descriptionRow{
				Entity: entityPokemon, ID: currentID, Version: game, Text: cleaned,
				Source: "bulbapedia", SourceTitle: pageTitle,
			})
		}
	}
	return rows, unresolved
}

// parseMoveZADescription returns the movedescentry text tagged
// {{gameabbrev9|ZA}} under a move page's "Description" section, if any --
// most moves have one, but Dragon Dance did not in testing, presumably
// because it isn't in Legends: Z-A.
func parseMoveZADescription(wikitext string) (string, bool) {
	sec, ok := section(wikitext, "Description")
	if !ok {
		return "", false
	}
	for _, occ := range findTemplates(sec, "movedescentry") {
		parts := splitTopLevel(occ.Body)
		if len(parts) < 2 {
			continue
		}
		gamesField, text := parts[0], parts[1]
		for _, code := range gameAbbrevCodes(gamesField) {
			if code == "ZA" {
				return cleanWikitext(text), true
			}
		}
	}
	return "", false
}
