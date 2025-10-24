package trm

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TxManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
	DoReadOnly(ctx context.Context, fn func(ctx context.Context) error) error
}

// Manager implements a transaction manager using pgx
// It provides methods to execute functions within a transaction context.
type Manager struct {
	db *pgxpool.Pool
}

// New returns a new Transaction Manager
func New(db *pgxpool.Pool) *Manager {
	return &Manager{db: db}
}

// Unique key for TX
type ctxKeyTx struct{}
type ctxTxOptions struct{}

var TxKey = ctxKeyTx{}
var txOptions = ctxTxOptions{}

// Do executes the provided function within a transaction context.
// It supports nesting via savepoints. Panics rollback and are re-panicked.
func (m *Manager) Do(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	var tx pgx.Tx
	tx, ctx, err = m.getTransactionFromContext(ctx)
	if err != nil {
		return err
	}

	// Handle commit/rollback (and panic) once fn returns/exits.
	defer func() {
		if p := recover(); p != nil {
			// Panic path: best-effort rollback, then re-panic to preserve the stack.
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				err = fmt.Errorf("failed to rollback tx after panic: %v", rbErr)
			}
			panic(p)
		}

		if err != nil {
			// Error path: rollback and prefer original error while surfacing rollback failures.
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				err = fmt.Errorf("rollback failed: %v (original error: %w)", rbErr, err)
			}
			return
		}

		// Success path: commit/release savepoint.
		if commitErr := tx.Commit(ctx); commitErr != nil {
			err = fmt.Errorf("failed to commit tx: %w", commitErr)
		}
	}()

	// Execute user code within the transaction context (tx already injected into ctx).
	err = fn(ctx)
	return err
}

// getTransactionFromContext returns the "current layer" tx and an updated ctx:
// - If a tx exists, it opens a SAVEPOINT (nested tx), stores that in ctx, and returns it.
// - If no tx exists, it begins a new tx (honoring any options in ctx), stores it in ctx, and returns it.
func (m *Manager) getTransactionFromContext(ctx context.Context) (pgx.Tx, context.Context, error) {
	// Nested transaction: create a savepoint under the current layer.
	if current, ok := ctx.Value(TxKey).(pgx.Tx); ok && current != nil {
		savepoint, err := current.Begin(ctx)
		if err != nil {
			return nil, ctx, fmt.Errorf("failed to create savepoint: %w", err)
		}
		// IMPORTANT: set the savepoint as the current layer so deeper nesting nests under it.
		ctx = context.WithValue(ctx, TxKey, savepoint)
		return savepoint, ctx, nil
	}

	// Top-level transaction: honor options if present.
	if opt, ok := ctx.Value(txOptions).(pgx.TxOptions); ok {
		tx, err := m.db.BeginTx(ctx, opt)
		if err != nil {
			return nil, ctx, fmt.Errorf("failed to start new transaction with options: %w", err)
		}
		ctx = context.WithValue(ctx, TxKey, tx)
		return tx, ctx, nil
	}

	// No options: start a default transaction.
	tx, err := m.db.Begin(ctx)
	if err != nil {
		return nil, ctx, fmt.Errorf("failed to start new transaction: %w", err)
	}
	ctx = context.WithValue(ctx, TxKey, tx)
	return tx, ctx, nil
}

// DoReadOnly executes the provided function within a read-only transaction context.
// It merges read-only with any existing pgx.TxOptions in the context (keeps isolation, deferrable, etc.).
func (m *Manager) DoReadOnly(ctx context.Context, fn func(ctx context.Context) error) error {
	// Merge with any existing options instead of overwriting.
	if existing, ok := ctx.Value(txOptions).(pgx.TxOptions); ok {
		existing.AccessMode = pgx.ReadOnly
		ctx = WithOptionsCtx(ctx, existing)
	} else {
		ctx = WithOptionsCtx(ctx, pgx.TxOptions{
			AccessMode: pgx.ReadOnly,
		})
	}
	return m.Do(ctx, fn)
}

// WithOptionsCtx stores tx options in the context (used by Do / DoReadOnly).
func WithOptionsCtx(ctx context.Context, opt pgx.TxOptions) context.Context {
	return context.WithValue(ctx, txOptions, opt)
}
