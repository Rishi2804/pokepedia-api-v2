package service

import (
	"context"
	"errors"
	"slices"
	"testing"
)

// The format check runs before any query, so a nil-backed service is enough.
func TestGetHoldableItemsRejectsUnknownFormat(t *testing.T) {
	s := &ItemsService{}
	for _, format := range []string{"", "10", "0", "gen9", "NationalDex", "9; DROP TABLE item"} {
		if _, err := s.GetHoldableItems(context.Background(), format); !errors.Is(err, ErrInvalidItemFormat) {
			t.Errorf("format %q: err = %v, want ErrInvalidItemFormat", format, err)
		}
	}
}

// itemFormats must stay in sync with the BattleFormatKey values the scraper
// writes into item.legal_formats (pokepedia-battle's scripts/dump-items.ts).
func TestItemFormatsCoverEveryBattleFormat(t *testing.T) {
	for _, want := range []string{"2", "3", "4", "5", "6", "7", "8", "9", "nationaldex", "legendsza"} {
		if !slices.Contains(itemFormats, want) {
			t.Errorf("itemFormats is missing %q", want)
		}
	}
}
