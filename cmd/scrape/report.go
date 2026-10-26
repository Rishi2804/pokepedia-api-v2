package main

import (
	"fmt"
	"sort"
	"strings"
)

// report prints one line per (entity, version) pair -- the checkpoint the
// plan's Stage 1 go/no-go and every later stage's dry-run reads.
//
// Unresolved items split in two: known ones (see knowngaps.go) collapse to
// a one-line count so the same ~38 diagnosed gaps don't repeat on every run,
// while anything NOT in that table prints in full -- a silent skip there is
// exactly the failure mode this tool exists to avoid (see resolve.go), so
// only DIAGNOSED gaps get to be quiet.
func report(rows []descriptionRow, unresolved []unresolvedItem) {
	type key struct {
		entity  entity
		version string
	}
	counts := map[key]int{}
	sources := map[key]map[string]bool{}
	for _, r := range rows {
		k := key{r.Entity, r.Version}
		counts[k]++
		if sources[k] == nil {
			sources[k] = map[string]bool{}
		}
		sources[k][r.Source] = true
	}

	var keys []key
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].entity != keys[j].entity {
			return keys[i].entity < keys[j].entity
		}
		return keys[i].version < keys[j].version
	})

	fmt.Printf("built %d description rows:\n", len(rows))
	for _, k := range keys {
		var srcs []string
		for s := range sources[k] {
			srcs = append(srcs, s)
		}
		sort.Strings(srcs)
		fmt.Printf("  %-8s %-16s %6d  (%s)\n", k.entity, k.version, counts[k], strings.Join(srcs, ","))
	}

	printUnresolved(unresolved)
}

// reportVerbose is report plus the full detail line for every known gap too
// -- for when you want to double check the suppressed list still matches
// what's actually in the database, e.g. after adding rows for a variant
// that used to be a known gap.
func reportVerbose(rows []descriptionRow, unresolved []unresolvedItem) {
	report(rows, unresolved)
	printKnownDetail(unresolved)
}

// printUnresolved is the split-and-summarize logic report() and
// reportEggMoves() share: known gaps (see knowngaps.go) collapse to a
// one-line count, everything else prints in full.
func printUnresolved(unresolved []unresolvedItem) {
	var known, unknown []unresolvedItem
	for _, u := range unresolved {
		if u.Known {
			known = append(known, u)
		} else {
			unknown = append(unknown, u)
		}
	}

	fmt.Printf("unresolved: %d (%d known, %d new)\n", len(unresolved), len(known), len(unknown))
	for _, u := range unknown {
		fmt.Printf("  ? %s\n", u.Message)
	}
	if len(known) > 0 {
		fmt.Printf("  known gaps (see cmd/scrape/knowngaps.go): %d -- run with -verbose-known to list\n", len(known))
	}
}

func printKnownDetail(unresolved []unresolvedItem) {
	for _, u := range unresolved {
		if u.Known {
			fmt.Printf("  = %s -- %s\n", u.Message, u.Reason)
		}
	}
}

// reportEggMoves prints one line per version group -- the movedetails
// analogue of report() above. Pokemon count (not just row count) is worth
// showing separately since the headline finding motivating this pass was
// entirely about evolved forms sitting at zero, not about row volume.
func reportEggMoves(rows []moveDetailRow, unresolved []unresolvedItem) {
	type key struct{ version string }
	counts := map[key]int{}
	pokemon := map[key]map[int32]bool{}
	for _, r := range rows {
		k := key{r.Version}
		counts[k]++
		if pokemon[k] == nil {
			pokemon[k] = map[int32]bool{}
		}
		pokemon[k][r.PokemonID] = true
	}

	var keys []key
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].version < keys[j].version })

	fmt.Printf("built %d egg-move rows:\n", len(rows))
	for _, k := range keys {
		fmt.Printf("  %-38s %6d rows  %5d pokemon\n", k.version, counts[k], len(pokemon[k]))
	}

	printUnresolved(unresolved)
}

func reportEggMovesVerbose(rows []moveDetailRow, unresolved []unresolvedItem) {
	reportEggMoves(rows, unresolved)
	printKnownDetail(unresolved)
}

// reportLegends prints one line per (version, method) pair -- the
// movedetails+legendsmovevalues analogue of reportEggMoves above.
func reportLegends(rows []legendsMoveRow, unresolved []unresolvedItem) {
	type key struct{ version, method string }
	counts := map[key]int{}
	pokemon := map[key]map[int32]bool{}
	for _, r := range rows {
		k := key{r.Version, r.Method}
		counts[k]++
		if pokemon[k] == nil {
			pokemon[k] = map[int32]bool{}
		}
		pokemon[k][r.PokemonID] = true
	}

	var keys []key
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].version != keys[j].version {
			return keys[i].version < keys[j].version
		}
		return keys[i].method < keys[j].method
	})

	fmt.Printf("built %d legends learnset rows:\n", len(rows))
	for _, k := range keys {
		fmt.Printf("  %-16s %-10s %6d rows  %5d pokemon\n", k.version, k.method, counts[k], len(pokemon[k]))
	}

	printUnresolved(unresolved)
}

func reportLegendsVerbose(rows []legendsMoveRow, unresolved []unresolvedItem) {
	reportLegends(rows, unresolved)
	printKnownDetail(unresolved)
}

// reportItems prints the item pass summary. Items that need a human look
// (Showdown items with no PokeAPI row, non-holdable Showdown matches, missing
// generations) come through unresolved.
func reportItems(rows []itemRow, unresolved []unresolvedItem) {
	var holdable, withShowdown, descs, heldBy int
	pockets := map[string]int{}
	for _, r := range rows {
		if r.Holdable {
			holdable++
		}
		if r.ShowdownID != nil {
			withShowdown++
		}
		descs += len(r.Descriptions)
		heldBy += len(r.HeldBy)
		pockets[r.Pocket]++
	}

	fmt.Printf("items: %d (holdable+battle-legal: %d, matched to Showdown: %d)\n", len(rows), holdable, withShowdown)
	fmt.Printf("item descriptions: %d, wild held-by rows: %d\n", descs, heldBy)

	names := make([]string, 0, len(pockets))
	for p := range pockets {
		names = append(names, p)
	}
	sort.Strings(names)
	for _, p := range names {
		fmt.Printf("  pocket %-10s %d\n", p, pockets[p])
	}

	printUnresolved(unresolved)
}

// reportNames prints the names pass summary and a few sample rows, so the
// results can be eyeballed before anything is written.
func reportNames(data *nameData, unresolved []unresolvedItem) {
	var withShowdown, battleOnly, withItem, gmax int
	for _, p := range data.Pokemon {
		if p.ShowdownID != nil {
			withShowdown++
		}
		if p.BattleOnlyBaseID != nil {
			battleOnly++
		}
		if p.RequiredItemID != nil {
			withItem++
		}
		if p.Gigantamax {
			gmax++
		}
	}
	matched := func(rows []nameRow) int {
		n := 0
		for _, r := range rows {
			if r.ShowdownID != nil {
				n++
			}
		}
		return n
	}
	fmt.Printf("species:   %d named\n", len(data.Species))
	fmt.Printf("pokemon:   %d named, %d with a Showdown id (%d battle-only, %d with a required item, %d Gmax)\n",
		len(data.Pokemon), withShowdown, battleOnly, withItem, gmax)
	fmt.Printf("moves:     %d named, %d with a Showdown id\n", len(data.Moves), matched(data.Moves))
	fmt.Printf("abilities: %d named, %d with a Showdown id\n", len(data.Abilities), matched(data.Abilities))
	printUnresolved(unresolved)
}
