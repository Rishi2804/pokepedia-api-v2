package service

import (
	"context"
	"errors"
	"slices"

	"github.com/Rishi2804/pokepedia-api-v2/internal/dto"
	"github.com/Rishi2804/pokepedia-api-v2/internal/pokeenum"
	"github.com/Rishi2804/pokepedia-api-v2/internal/store"
	"github.com/Rishi2804/pokepedia-api-v2/internal/util"
)

// ErrInvalidItemFormat is returned for a format the battle server has no
// item rules for.
var ErrInvalidItemFormat = errors.New("invalid item format")

// itemFormats are the BattleFormatKey values stored in item.legal_formats.
var itemFormats = []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "nationaldex", "legendsza"}

type ItemsService struct {
	q *store.Queries
}

func NewItemsService(q *store.Queries) *ItemsService {
	return &ItemsService{q: q}
}

func (s *ItemsService) GetItemList(ctx context.Context) ([]dto.ItemSnap, error) {
	rows, err := s.q.GetItemsList(ctx)
	if err != nil {
		return nil, err
	}
	response := []dto.ItemSnap{}
	for _, r := range rows {
		response = append(response, dto.ItemSnap{
			ID:       r.ID,
			Slug:     r.Name,
			Name:     r.DisplayName,
			Gen:      r.Gen,
			Category: r.Category,
			Pocket:   r.Pocket,
			Image:    deref(r.Image),
		})
	}
	return response, nil
}

func (s *ItemsService) GetItemDetail(ctx context.Context, id int32) (*dto.ItemDetail, error) {
	i, err := s.q.GetItem(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.buildDetail(ctx, i)
}

func (s *ItemsService) GetItemByName(ctx context.Context, name string) (*dto.ItemDetail, error) {
	i, err := s.q.GetItemByName(ctx, name)
	if err != nil {
		return nil, err
	}
	return s.buildDetail(ctx, i)
}

// GetHoldableItems returns the items the team builder may offer for a battle
// format: holdable and accepted by the battle server in that format.
func (s *ItemsService) GetHoldableItems(ctx context.Context, format string) ([]dto.HoldableItem, error) {
	if !slices.Contains(itemFormats, format) {
		return nil, ErrInvalidItemFormat
	}
	rows, err := s.q.GetHoldableItems(ctx, format)
	if err != nil {
		return nil, err
	}
	response := []dto.HoldableItem{}
	for _, r := range rows {
		response = append(response, dto.HoldableItem{
			ShowdownID:  deref(r.ShowdownID),
			Name:        r.DisplayName,
			Image:       deref(r.Image),
			ShortEffect: deref(r.ShortEffect),
		})
	}
	return response, nil
}

func (s *ItemsService) buildDetail(ctx context.Context, i store.Item) (*dto.ItemDetail, error) {
	descriptionRows, err := s.q.GetItemDescriptions(ctx, i.ID)
	if err != nil {
		return nil, err
	}
	descriptions := toDescriptions(descriptionRows, func(r store.GetItemDescriptionsRow) ([]string, string) {
		return r.Games, r.Text
	})

	holders, err := s.q.GetPokemonHoldingItem(ctx, i.ID)
	if err != nil {
		return nil, err
	}
	pokemon := []dto.PokemonSnap{}
	for _, p := range holders {
		pokemon = append(pokemon, dto.PokemonSnap{
			SpeciesID: p.SpeciesID,
			ID:        p.ID,
			Name:      util.FormatName(p.Name, true),
			Type1:     pokeenum.ToDisplay(p.Type1),
			Type2:     displayPtr(p.Type2),
		})
	}

	var flingEffect *string
	if i.FlingEffect != nil {
		formatted := util.FormatName(*i.FlingEffect, false)
		flingEffect = &formatted
	}

	legalFormats := i.LegalFormats
	if legalFormats == nil {
		legalFormats = []string{}
	}

	return &dto.ItemDetail{
		ID:           i.ID,
		Slug:         i.Name,
		Name:         i.DisplayName,
		Gen:          i.Gen,
		Category:     i.Category,
		Pocket:       i.Pocket,
		Cost:         i.Cost,
		FlingPower:   i.FlingPower,
		FlingEffect:  flingEffect,
		Effect:       deref(i.Effect),
		ShortEffect:  deref(i.ShortEffect),
		Image:        deref(i.Image),
		Holdable:     i.Holdable,
		ShowdownID:   i.ShowdownID,
		LegalFormats: legalFormats,
		Descriptions: descriptions,
		Pokemon:      pokemon,
	}, nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
