package store

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Pokemon struct {
	ID         int32
	Name       string
	Gen        int32
	Type1      string
	Type2      *string // nullable — not every Pokemon has a second type
	Weight     float64
	Height     float64
	GenderRate float64
	HP         int32
	Atk        int32
	Def        int32
	SpAtk      int32
	SpDef      int32
	Speed      int32
	BST        int32
	SpeciesID  int32
}

type PokemonStore struct {
	db *pgxpool.Pool
}

func NewPokemonStore(db *pgxpool.Pool) *PokemonStore {
	return &PokemonStore{db: db}
}

func (s *PokemonStore) List(ctx context.Context) ([]Pokemon, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, name, gen, type1, type2, weight, height, gender_rate,
		       hp, atk, def, spatk, spdef, speed, bst, species_id
		FROM pokemon
		ORDER BY id
		LIMIT 50
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Pokemon
	for rows.Next() {
		var p Pokemon
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Gen, &p.Type1, &p.Type2, &p.Weight, &p.Height,
			&p.GenderRate, &p.HP, &p.Atk, &p.Def, &p.SpAtk, &p.SpDef,
			&p.Speed, &p.BST, &p.SpeciesID,
		); err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

func (s *PokemonStore) Get(ctx context.Context, id int32) (Pokemon, error) {
	var p Pokemon
	err := s.db.QueryRow(ctx, `
		SELECT *
		FROM pokemon
		WHERE id = $1
	`, id).Scan(
		&p.ID, &p.Name, &p.Gen, &p.Type1, &p.Type2, &p.Weight, &p.Height,
		&p.GenderRate, &p.HP, &p.Atk, &p.Def, &p.SpAtk, &p.SpDef,
		&p.Speed, &p.BST, &p.SpeciesID,
	)
	return p, err
}
