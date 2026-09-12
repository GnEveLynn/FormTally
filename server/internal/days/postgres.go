package days

import (
	"context"
	"github.com/GnEveLynn/FormTally/server/internal/meals"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct{ pool *pgxpool.Pool }

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore { return &PostgresStore{pool: pool} }
func (s *PostgresStore) Timezone(ctx context.Context, userID string) (string, error) {
	var timezone string
	err := s.pool.QueryRow(ctx, `select timezone from profiles where user_id=$1`, userID).Scan(&timezone)
	return timezone, err
}
func (s *PostgresStore) Meals(ctx context.Context, userID, date string) ([]meals.Meal, error) {
	rows, err := s.pool.Query(ctx, `select m.id,m.occurred_at,m.meal_type,coalesce(sum(i.energy_kcal),0)::int,coalesce(sum(i.protein_grams),0),coalesce(sum(i.carb_grams),0),coalesce(sum(i.fat_grams),0) from meals m join meal_items i on i.meal_id=m.id where m.user_id=$1 and m.local_date=$2 group by m.id order by m.occurred_at`, userID, date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []meals.Meal{}
	for rows.Next() {
		var meal meals.Meal
		meal.UserID = userID
		meal.LocalDate = date
		if err = rows.Scan(&meal.ID, &meal.OccurredAt, &meal.MealType, &meal.Totals.EnergyKcal, &meal.Totals.ProteinGrams, &meal.Totals.CarbGrams, &meal.Totals.FatGrams); err != nil {
			return nil, err
		}
		result = append(result, meal)
	}
	return result, rows.Err()
}
func (s *PostgresStore) History(ctx context.Context, userID, month string) ([]HistoryDay, error) {
	rows, err := s.pool.Query(ctx, `select m.local_date::text,count(distinct m.id)::int,coalesce(sum(i.energy_kcal),0)::int from meals m join meal_items i on i.meal_id=m.id where m.user_id=$1 and to_char(m.local_date,'YYYY-MM')=$2 group by m.local_date order by m.local_date`, userID, month)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []HistoryDay{}
	for rows.Next() {
		var day HistoryDay
		if err = rows.Scan(&day.LocalDate, &day.MealCount, &day.EnergyKcal); err != nil {
			return nil, err
		}
		day.Status = "recorded"
		result = append(result, day)
	}
	return result, rows.Err()
}
