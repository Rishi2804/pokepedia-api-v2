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

// reportVerbose is report plus the full detail line for every known gap too
// -- for when you want to double check the suppressed list still matches
// what's actually in the database, e.g. after adding rows for a variant
// that used to be a known gap.
func reportVerbose(rows []descriptionRow, unresolved []unresolvedItem) {
	report(rows, unresolved)
	for _, u := range unresolved {
		if u.Known {
			fmt.Printf("  = %s -- %s\n", u.Message, u.Reason)
		}
	}
}
