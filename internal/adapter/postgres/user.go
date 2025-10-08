package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/Temutjin2k/ride-hail-system/internal/domain/models"
	"github.com/Temutjin2k/ride-hail-system/pkg/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepo struct {
	db *pgxpool.Pool
}

func NewUserRepo(db *pgxpool.Pool) *UserRepo {
	return &UserRepo{
		db: db,
	}
}

// CreateUser inserts a user row. It expects u.Email, u.Role, and u.PasswordHash to be set.
// u.Status is optional (defaults to 'ACTIVE' in DB); u.Attrs is optional.
func (r *UserRepo) CreateUser(ctx context.Context, u *models.User) (uuid.UUID, error) {
	if u == nil {
		return uuid.UUID{}, errors.New("nil user")
	}

	var attrsJSON []byte
	if u.Attrs != nil {
		var err error
		attrsJSON, err = json.Marshal(u.Attrs)
		if err != nil {
			return uuid.UUID{}, err
		}
	} else {
		attrsJSON = []byte(`{}`)
	}

	const q = `
		INSERT INTO users (email, role, status, password_hash, attrs)
		VALUES ($1, $2, COALESCE(NULLIF($3, ''), 'ACTIVE'), $4, $5::jsonb)
		RETURNING id, created_at, updated_at, status;
	`

	var (
		id     uuid.UUID
		status string
	)

	// pgx.Timestamp is just time.Time in v5, but keep explicit variables for clarity
	err := TxorDB(ctx, r.db).QueryRow(
		ctx, q, u.Email, u.Role, u.Status, u.PasswordHash, string(attrsJSON),
	).Scan(&id, &u.CreatedAt, &u.UpdatedAt, &status)
	if err != nil {
		return uuid.UUID{}, err
	}

	u.ID = id
	if u.Status == "" {
		u.Status = status
	}
	return id, nil
}

// GetUser fetches by email (unique).
func (r *UserRepo) GetUser(ctx context.Context, email string) (*models.User, error) {
	if email == "" {
		return nil, errors.New("email is required")
	}

	const q = `
		SELECT id, created_at, updated_at, email, role, status, password_hash, attrs
		FROM users
		WHERE email = $1;
	`

	var (
		u         models.User
		attrsJSON []byte
	)
	err := TxorDB(ctx, r.db).QueryRow(ctx, q, email).Scan(
		&u.ID,
		&u.CreatedAt,
		&u.UpdatedAt,
		&u.Email,
		&u.Role,
		&u.Status,
		&u.PasswordHash,
		&attrsJSON,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // not found
		}
		return nil, err
	}

	if len(attrsJSON) > 0 {
		_ = json.Unmarshal(attrsJSON, &u.Attrs) // tolerate malformed attrs; optionally handle error
	}
	return &u, nil
}

// GetUserByID fetches by UUID id.
func (r *UserRepo) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	if id.String() == "" {
		return nil, errors.New("id is required")
	}

	const q = `
		SELECT id, created_at, updated_at, email, role, status, password_hash, attrs
		FROM users
		WHERE id = $1;
	`

	var (
		u         models.User
		attrsJSON []byte
	)

	err := TxorDB(ctx, r.db).QueryRow(ctx, q, id).Scan(
		&u.ID,
		&u.CreatedAt,
		&u.UpdatedAt,
		&u.Email,
		&u.Role,
		&u.Status,
		&u.PasswordHash,
		&attrsJSON,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // not found
		}
		return nil, err
	}

	if len(attrsJSON) > 0 {
		_ = json.Unmarshal(attrsJSON, &u.Attrs)
	}
	return &u, nil
}
