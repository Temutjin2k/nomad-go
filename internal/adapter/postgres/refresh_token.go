package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RefreshTokenRepo struct {
	db *pgxpool.Pool
}

func NewRefreshTokenRepo(db *pgxpool.Pool) *RefreshTokenRepo {
	return &RefreshTokenRepo{db: db}
}

func (r *RefreshTokenRepo) Save(ctx context.Context, record *models.RefreshTokenRecord) error {
	if record == nil {
		return errors.New("refresh token record is nil")
	}

	const q = `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, revoked, created_at)
		VALUES ($1, $2, $3, $4, false, $5)
		ON CONFLICT (id)
		DO UPDATE SET
			token_hash = EXCLUDED.token_hash,
			expires_at = EXCLUDED.expires_at,
			revoked = false,
			last_used_at = NULL;
	`

	_, err := TxorDB(ctx, r.db).Exec(ctx, q, record.ID, record.UserID, record.TokenHash, record.ExpiresAt, record.CreatedAt)
	return err
}

func (r *RefreshTokenRepo) Get(ctx context.Context, tokenID uuid.UUID) (*models.RefreshTokenRecord, error) {
	const q = `
		SELECT id, user_id, token_hash, expires_at, revoked, created_at, last_used_at
		FROM refresh_tokens
		WHERE id = $1;
	`

	var (
		rec      models.RefreshTokenRecord
		lastUsed sql.NullTime
	)

	err := TxorDB(ctx, r.db).QueryRow(ctx, q, tokenID).Scan(
		&rec.ID,
		&rec.UserID,
		&rec.TokenHash,
		&rec.ExpiresAt,
		&rec.Revoked,
		&rec.CreatedAt,
		&lastUsed,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if lastUsed.Valid {
		ts := lastUsed.Time.UTC()
		rec.LastUsed = &ts
	}

	return &rec, nil
}

func (r *RefreshTokenRepo) MarkUsed(ctx context.Context, tokenID uuid.UUID) error {
	const q = `
		UPDATE refresh_tokens
		SET revoked = true,
		    last_used_at = $2
		WHERE id = $1;
	`

	_, err := TxorDB(ctx, r.db).Exec(ctx, q, tokenID, time.Now().UTC())
	return err
}
