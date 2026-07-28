package store

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Pokemon mirrors the columns actually used by the existing Spring Boot
// service (dex_entries is a real column on this table but is intentionally
// unused — dex entries are fetched separately from the `dexentries` table).
type Pokemon struct {
	ID         int32
	SpeciesID  int32
	Name       string
	Gen        int32
	Type1      string
	Type2      *string // nullable — single-typed Pokemon have no second type
	Weight     float64
	Height     float64
	GenderRate int32
	HP         int32
	Atk        int32
	Def        int32
	SpAtk      int32
	SpDef      int32
	Speed      int32
	BST        int32
	Forms      []string // text[], nullable — pgx maps NULL text[] to a nil slice
}

type PokemonStore struct {
	db *pgxpool.Pool
}

func NewPokemonStore(db *pgxpool.Pool) *PokemonStore {
	return &PokemonStore{db: db}
}

// pokemonColumns is the single source of truth for column order, shared by
// every query below. Keeping this in one place is what prevents the
// "SELECT * returned more columns than Scan() expected" bug we just hit —
// there's only one list to keep in sync with the Scan() calls now.
const pokemonColumns = `
	id, species_id, name, gen, type1, type2, weight, height,
	gender_rate, hp, atk, def, spatk, spdef, speed, bst, forms
`

func (s *PokemonStore) List(ctx context.Context) ([]Pokemon, error) {
	rows, err := s.db.Query(ctx, `
		SELECT `+pokemonColumns+`
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
			&p.ID, &p.SpeciesID, &p.Name, &p.Gen, &p.Type1, &p.Type2,
			&p.Weight, &p.Height, &p.GenderRate, &p.HP, &p.Atk, &p.Def,
			&p.SpAtk, &p.SpDef, &p.Speed, &p.BST, &p.Forms,
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
		SELECT `+pokemonColumns+`
		FROM pokemon
		WHERE id = $1
	`, id).Scan(
		&p.ID, &p.SpeciesID, &p.Name, &p.Gen, &p.Type1, &p.Type2,
		&p.Weight, &p.Height, &p.GenderRate, &p.HP, &p.Atk, &p.Def,
		&p.SpAtk, &p.SpDef, &p.Speed, &p.BST, &p.Forms,
	)
	return p, err
}

// GetByName mirrors the Java repository's getPokemonByName — the
// controller supports lookup by both numeric id and name.
func (s *PokemonStore) GetByName(ctx context.Context, name string) (Pokemon, error) {
	var p Pokemon
	err := s.db.QueryRow(ctx, `
		SELECT `+pokemonColumns+`
		FROM pokemon
		WHERE name = $1
	`, name).Scan(
		&p.ID, &p.SpeciesID, &p.Name, &p.Gen, &p.Type1, &p.Type2,
		&p.Weight, &p.Height, &p.GenderRate, &p.HP, &p.Atk, &p.Def,
		&p.SpAtk, &p.SpDef, &p.Speed, &p.BST, &p.Forms,
	)
	return p, err
}
