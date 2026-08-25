package util

import "testing"

func TestSlugify(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"already lowercase", "pikachu", "pikachu"},
		{"mixed case and whitespace", "  PIKACHU  ", "pikachu"},
		{"multi-word, no reordering", "Mega Venusaur", "mega-venusaur"},
		{"regional adjective contracts to db stem", "Alolan Raichu", "alola-raichu"},
		{"galarian contracts", "Galarian Farfetchd", "galar-farfetchd"},
		{"apostrophe stripped, matches real db slug", "King's Shield", "kings-shield"},
		{"an exact slug passes through unchanged", "raichu-alola", "raichu-alola"},
		{"defensive punctuation stripping", "pika(chu)!", "pikachu"},
		{"empty input", "", ""},
		{"whitespace only", "   ", ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Slugify(c.input); got != c.want {
				t.Errorf("Slugify(%q) = %q, want %q", c.input, got, c.want)
			}
		})
	}
}
