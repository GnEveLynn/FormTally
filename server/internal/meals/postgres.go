package meals

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct{ pool *pgxpool.Pool }

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore { return &PostgresStore{pool: pool} }

func (s *PostgresStore) Create(ctx context.Context, userID string, input CreateInput, now time.Time) (Meal, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Meal{}, err
	}
	defer tx.Rollback(ctx)
	var imageKey *string
	if input.AnalysisID != nil {
		var status string
		var expires time.Time
		err = tx.QueryRow(ctx, `select image_key,status,expires_at from meal_analyses where id=$1 and user_id=$2 for update`, *input.AnalysisID, userID).Scan(&imageKey, &status, &expires)
		if errors.Is(err, pgx.ErrNoRows) {
			return Meal{}, ErrNotFound
		}
		if err != nil {
			return Meal{}, err
		}
		if !expires.After(now) {
			return Meal{}, ErrAnalysisExpired
		}
		if status != "review_required" && status != "failed" {
			return Meal{}, ErrAnalysisState
		}
	}
	occurred, _ := time.Parse(time.RFC3339, input.OccurredAt)
	meal := Meal{ID: "meal_" + rand.Text(), UserID: userID, OccurredAt: occurred, LocalDate: occurred.Format("2006-01-02"), MealType: input.MealType, ImageKey: imageKey, Revision: 1, CreatedAt: now, UpdatedAt: now}
	if _, err = tx.Exec(ctx, `insert into meals(id,user_id,occurred_at,local_date,meal_type,image_key,revision,created_at,updated_at) values($1,$2,$3,$4,$5,$6,1,$7,$7)`, meal.ID, userID, meal.OccurredAt, meal.LocalDate, meal.MealType, imageKey, now); err != nil {
		return Meal{}, err
	}
	meal.Items = make([]Item, len(input.Items))
	for index, item := range input.Items {
		item.Name = strings.TrimSpace(item.Name)
		id := "item_" + rand.Text()
		basis, _ := json.Marshal(item.BasisPer100Grams)
		if item.BasisPer100Grams == nil {
			basis = nil
		}
		origin := item.Origin
		if origin == "" {
			origin = "manual"
		}
		_, err = tx.Exec(ctx, `insert into meal_items(id,meal_id,position,draft_item_id,name,grams,energy_kcal,protein_grams,carb_grams,fat_grams,basis_per_100_grams,origin,confidence,assumption) values($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`, id, meal.ID, index, item.DraftItemID, item.Name, item.Grams, item.Nutrition.EnergyKcal, item.Nutrition.ProteinGrams, item.Nutrition.CarbGrams, item.Nutrition.FatGrams, basis, origin, item.Confidence, item.Assumption)
		if err != nil {
			return Meal{}, err
		}
		item.Origin = origin
		meal.Items[index] = Item{ID: id, ItemInput: item}
	}
	if input.AnalysisID != nil {
		if _, err = tx.Exec(ctx, `update meal_analyses set status='saved',meal_id=$2,revision=revision+1,updated_at=$3 where id=$1`, *input.AnalysisID, meal.ID, now); err != nil {
			return Meal{}, err
		}
	}
	meal.Totals = Sum(meal.Items)
	if err = tx.Commit(ctx); err != nil {
		return Meal{}, err
	}
	return meal, nil
}

func (s *PostgresStore) Get(ctx context.Context, userID, id string) (*Meal, error) {
	meal := &Meal{ID: id, UserID: userID}
	err := s.pool.QueryRow(ctx, `select occurred_at,local_date::text,meal_type,image_key,revision,created_at,updated_at from meals where id=$1 and user_id=$2`, id, userID).Scan(&meal.OccurredAt, &meal.LocalDate, &meal.MealType, &meal.ImageKey, &meal.Revision, &meal.CreatedAt, &meal.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `select id,draft_item_id,name,grams,energy_kcal,protein_grams,carb_grams,fat_grams,basis_per_100_grams,origin,confidence,assumption from meal_items where meal_id=$1 order by position`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var item Item
		var basis []byte
		if err = rows.Scan(&item.ID, &item.DraftItemID, &item.Name, &item.Grams, &item.Nutrition.EnergyKcal, &item.Nutrition.ProteinGrams, &item.Nutrition.CarbGrams, &item.Nutrition.FatGrams, &basis, &item.Origin, &item.Confidence, &item.Assumption); err != nil {
			return nil, err
		}
		if len(basis) > 0 {
			item.BasisPer100Grams = new(Nutrition)
			if err = json.Unmarshal(basis, item.BasisPer100Grams); err != nil {
				return nil, err
			}
		}
		meal.Items = append(meal.Items, item)
	}
	meal.Totals = Sum(meal.Items)
	return meal, rows.Err()
}

func (s *PostgresStore) Update(ctx context.Context, userID, id string, input UpdateInput, now time.Time) (Meal, []string, error) {
	current, err := s.Get(ctx, userID, id)
	if err != nil {
		return Meal{}, nil, err
	}
	if current == nil {
		return Meal{}, nil, ErrNotFound
	}
	if current.Revision != input.ExpectedRevision {
		return Meal{}, nil, ErrRevisionConflict
	}
	oldDate := current.LocalDate
	if input.OccurredAt != nil {
		parsed, e := time.Parse(time.RFC3339, *input.OccurredAt)
		if e != nil || parsed.After(now) {
			return Meal{}, nil, ErrValidation
		}
		current.OccurredAt = parsed
		current.LocalDate = parsed.Format("2006-01-02")
	}
	if input.MealType != nil {
		current.MealType = *input.MealType
	}
	items := currentInputs(current.Items)
	if input.Items != nil {
		items = *input.Items
	}
	if err = ValidateCreate(CreateInput{OccurredAt: current.OccurredAt.Format(time.RFC3339), MealType: current.MealType, Items: items}, now); err != nil {
		return Meal{}, nil, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Meal{}, nil, err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `update meals set occurred_at=$4,local_date=$5,meal_type=$6,revision=revision+1,updated_at=$7 where id=$1 and user_id=$2 and revision=$3`, id, userID, input.ExpectedRevision, current.OccurredAt, current.LocalDate, current.MealType, now)
	if err != nil {
		return Meal{}, nil, err
	}
	if tag.RowsAffected() == 0 {
		return Meal{}, nil, ErrRevisionConflict
	}
	if input.Items != nil {
		if _, err = tx.Exec(ctx, `delete from meal_items where meal_id=$1`, id); err != nil {
			return Meal{}, nil, err
		}
		if err = insertItems(ctx, tx, id, *input.Items); err != nil {
			return Meal{}, nil, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return Meal{}, nil, err
	}
	updated, err := s.Get(ctx, userID, id)
	dates := []string{oldDate}
	if current.LocalDate != oldDate {
		dates = append(dates, current.LocalDate)
	}
	return *updated, dates, err
}

func (s *PostgresStore) RemoveImage(ctx context.Context, userID, id string, revision int, now time.Time) (Meal, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Meal{}, err
	}
	defer tx.Rollback(ctx)
	var key *string
	var current int
	err = tx.QueryRow(ctx, `select image_key,revision from meals where id=$1 and user_id=$2 for update`, id, userID).Scan(&key, &current)
	if errors.Is(err, pgx.ErrNoRows) {
		return Meal{}, ErrNotFound
	}
	if err != nil {
		return Meal{}, err
	}
	if current != revision {
		return Meal{}, ErrRevisionConflict
	}
	if key != nil {
		if err = queueDeletion(ctx, tx, *key, now); err != nil {
			return Meal{}, err
		}
	}
	if _, err = tx.Exec(ctx, `update meals set image_key=null,revision=revision+1,updated_at=$2 where id=$1`, id, now); err != nil {
		return Meal{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Meal{}, err
	}
	meal, err := s.Get(ctx, userID, id)
	return *meal, err
}
func (s *PostgresStore) Delete(ctx context.Context, userID, id string, revision int, now time.Time) (DeleteResult, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return DeleteResult{}, err
	}
	defer tx.Rollback(ctx)
	var key *string
	var date string
	var current int
	err = tx.QueryRow(ctx, `select image_key,local_date::text,revision from meals where id=$1 and user_id=$2 for update`, id, userID).Scan(&key, &date, &current)
	if errors.Is(err, pgx.ErrNoRows) {
		return DeleteResult{}, ErrNotFound
	}
	if err != nil {
		return DeleteResult{}, err
	}
	if current != revision {
		return DeleteResult{}, ErrRevisionConflict
	}
	if key != nil {
		if err = queueDeletion(ctx, tx, *key, now); err != nil {
			return DeleteResult{}, err
		}
	}
	if _, err = tx.Exec(ctx, `delete from meals where id=$1`, id); err != nil {
		return DeleteResult{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return DeleteResult{}, err
	}
	return DeleteResult{DeletedMealID: id, AffectedLocalDates: []string{date}}, nil
}
func insertItems(ctx context.Context, tx pgx.Tx, mealID string, items []ItemInput) error {
	for position, item := range items {
		origin := item.Origin
		if origin == "" {
			origin = "manual"
		}
		basis, _ := json.Marshal(item.BasisPer100Grams)
		if item.BasisPer100Grams == nil {
			basis = nil
		}
		if _, err := tx.Exec(ctx, `insert into meal_items(id,meal_id,position,draft_item_id,name,grams,energy_kcal,protein_grams,carb_grams,fat_grams,basis_per_100_grams,origin,confidence,assumption) values($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`, `item_`+rand.Text(), mealID, position, item.DraftItemID, strings.TrimSpace(item.Name), item.Grams, item.Nutrition.EnergyKcal, item.Nutrition.ProteinGrams, item.Nutrition.CarbGrams, item.Nutrition.FatGrams, basis, origin, item.Confidence, item.Assumption); err != nil {
			return err
		}
	}
	return nil
}
func queueDeletion(ctx context.Context, tx pgx.Tx, key string, now time.Time) error {
	_, err := tx.Exec(ctx, `insert into object_deletions(id,object_key,next_attempt_at,created_at) values($1,$2,$3,$3) on conflict(object_key) do nothing`, `deletion_`+rand.Text(), key, now)
	return err
}
func currentInputs(items []Item) []ItemInput {
	out := make([]ItemInput, len(items))
	for i, item := range items {
		out[i] = item.ItemInput
	}
	return out
}
