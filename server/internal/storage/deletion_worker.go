package storage

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

type Deletion struct {
	ID, ObjectKey string
	Attempts      int
}
type DeletionRepository interface {
	Next(context.Context, time.Time) (Deletion, bool, error)
	Complete(context.Context, string, time.Time) error
	Retry(context.Context, string, int, time.Time) error
}
type DeletionWorker struct {
	repo    DeletionRepository
	objects Store
	now     func() time.Time
}

func NewDeletionWorker(repo DeletionRepository, objects Store) *DeletionWorker {
	return &DeletionWorker{repo: repo, objects: objects, now: time.Now}
}
func (w *DeletionWorker) RunOnce(ctx context.Context) (bool, error) {
	deletion, ok, err := w.repo.Next(ctx, w.now())
	if err != nil || !ok {
		return ok, err
	}
	if err = w.objects.Delete(ctx, deletion.ObjectKey); err != nil {
		attempts := deletion.Attempts + 1
		delay := time.Duration(1<<min(attempts, 10)) * time.Minute
		if retryErr := w.repo.Retry(ctx, deletion.ID, attempts, w.now().Add(delay)); retryErr != nil {
			return true, retryErr
		}
		return true, nil
	}
	return true, w.repo.Complete(ctx, deletion.ID, w.now())
}

type PostgresDeletionRepository struct{ pool *pgxpool.Pool }

func NewPostgresDeletionRepository(pool *pgxpool.Pool) *PostgresDeletionRepository {
	return &PostgresDeletionRepository{pool: pool}
}
func (r *PostgresDeletionRepository) Next(ctx context.Context, now time.Time) (Deletion, bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Deletion{}, false, err
	}
	defer tx.Rollback(ctx)
	var d Deletion
	err = tx.QueryRow(ctx, `select id,object_key,attempts from object_deletions where completed_at is null and next_attempt_at<=$1 order by next_attempt_at for update skip locked limit 1`, now).Scan(&d.ID, &d.ObjectKey, &d.Attempts)
	if errors.Is(err, pgx.ErrNoRows) {
		return Deletion{}, false, nil
	}
	if err != nil {
		return Deletion{}, false, err
	}
	if _, err = tx.Exec(ctx, `update object_deletions set next_attempt_at=$2 where id=$1`, d.ID, now.Add(time.Minute)); err != nil {
		return Deletion{}, false, err
	}
	return d, true, tx.Commit(ctx)
}
func (r *PostgresDeletionRepository) Complete(ctx context.Context, id string, now time.Time) error {
	_, err := r.pool.Exec(ctx, `update object_deletions set completed_at=$2 where id=$1`, id, now)
	return err
}
func (r *PostgresDeletionRepository) Retry(ctx context.Context, id string, attempts int, next time.Time) error {
	_, err := r.pool.Exec(ctx, `update object_deletions set attempts=$2,next_attempt_at=$3 where id=$1`, id, attempts, next)
	return err
}
