package main

// knownGapKey identifies one {{Dex/Form|...}} marker that resolveFormMarker
// cannot resolve, by the species it appears on and its exact text.
type knownGapKey struct {
	SpeciesID int32
	Marker    string
}

// knownGaps records every unresolved form marker already diagnosed, and
// why, as of the 2026-08-27 dry run that first showed a clean 38-item list.
// Each entry was checked against this database's actual pokemon rows before
// being added here -- see the two categories below -- so this table is a
// record of confirmed findings, not a blanket silencer. report.go collapses
// these to a one-line summary instead of repeating the same ~38 lines every
// run; anything NOT in this table still prints in full, so a genuinely new
// problem (a parser regression, a page Bulbapedia restructured, ...) stays
// visible rather than getting lost in a wall of known noise.
//
// Category 1 -- no matching pokemon row exists for the named variant at
// all, so there is no id this scraper could write to even if the marker
// resolved (cosmetic-only variants this project never modeled as separate
// rows, plus the cap Pikachus excluded by an earlier explicit decision):
var knownGaps = map[knownGapKey]string{
	{143, "Mossy"}: "Snorlax: no matching pokemon row",

	{670, "Red Flower"}:    "Floette: no per-color pokemon row",
	{670, "Yellow Flower"}: "Floette: no per-color pokemon row",
	{670, "Orange Flower"}: "Floette: no per-color pokemon row",
	{670, "Blue Flower"}:   "Floette: no per-color pokemon row",
	{670, "White Flower"}:  "Floette: no per-color pokemon row",

	{479, "Stereo Rotom"}: "Rotom: no matching pokemon row",

	{869, "Vanilla Cream"}: "Alcremie: no per-flavor pokemon row",
	{869, "Ruby Cream"}:    "Alcremie: no per-flavor pokemon row",
	{869, "Matcha Cream"}:  "Alcremie: no per-flavor pokemon row",
	{869, "Mint Cream"}:    "Alcremie: no per-flavor pokemon row",
	{869, "Lemon Cream"}:   "Alcremie: no per-flavor pokemon row",
	{869, "Salted Cream"}:  "Alcremie: no per-flavor pokemon row",
	{869, "Ruby Swirl"}:    "Alcremie: no per-flavor pokemon row",
	{869, "Caramel Swirl"}: "Alcremie: no per-flavor pokemon row",
	{869, "Rainbow Swirl"}: "Alcremie: no per-flavor pokemon row",

	{25, "Original Cap, Hoenn Cap, Sinnoh Cap, Unova Cap, Kalos Cap, Alola Cap, and Partner Cap"}: "Pikachu: cap forms not modeled (excluded by earlier decision)",
	{25, "Original Cap"}: "Pikachu: cap forms not modeled (excluded by earlier decision)",
	{25, "Hoenn Cap, Sinnoh Cap, Unova Cap, Kalos Cap, and Alola Cap"}: "Pikachu: cap forms not modeled (excluded by earlier decision)",
	{25, "Partner Cap"}: "Pikachu: cap forms not modeled (excluded by earlier decision)",
	{25, "World Cap"}:   "Pikachu: cap forms not modeled (excluded by earlier decision)",
	{25, "Pale"}:        "Pikachu: no matching pokemon row",

	{931, "Green Plumage"}: "Squawkabilly: no matching pokemon row",

	// Category 2 -- genuine ambiguity in Bulbapedia's own text, verified
	// against the live wikitext, not a resolver shortcoming:
	{678, "Mega Meowstic"}: "Meowstic: one shared Legends: Z-A text applies to both meowstic-male-mega and meowstic-female-mega; not a single-id fit",
	{555, "Galarian Form"}: "Darmanitan: ties between darmanitan-galar-standard and darmanitan-galar-zen; Bulbapedia only disambiguates Zen via a separate \"Galarian Form/Zen Mode\" marker",
}
