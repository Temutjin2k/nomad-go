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
// It starts a new transaction if one does not already exist in the context.
// If the function returns an error, the transaction is rolled back.
// If the function completes successfully, the transaction is committed.
func (m *Manager) Do(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	var tx pgx.Tx
	tx, ctx, err = m.getTransactionFromContext(ctx)
	if err != nil {
		return err // return error if starting or retrieving transaction fails
	}

	// use defer to handle commit/rollback logic
	defer func() {
		if p := recover(); p != nil {
			// if a panic occurred, rollback
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				// log the error from rollback
				fmt.Printf("failed to rollback tx after panic: %v\n", rbErr)
			}
			panic(p) // re-throw panic after rollback
		} else if err != nil {
			// if an error occurred, rollback
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				// log the error from rollback
				err = fmt.Errorf("failed to rollback tx: %v (original error: %w)", rbErr, err)
			}
			// 'err' already contains an error from fn
		} else {
			// if no error and no panic, commit
			if commitErr := tx.Commit(ctx); commitErr != nil {
				err = fmt.Errorf("failed to commit tx: %w", commitErr)
			}
		}
	}()

	// execute the provided function within the transaction context
	err = fn(ctx)

	return err
}

// getTransactionFromContext tries to retrieve an existing transaction from the context.
// If a transaction exists, it creates a SAVEPOINT (nested transaction).
// If no transaction exists, it starts a NEW transaction and adds it to the context.
func (m *Manager) getTransactionFromContext(ctx context.Context) (pgx.Tx, context.Context, error) {
    // 1. Check if a transaction already exists in the context
	if tx, ok := ctx.Value(TxKey).(pgx.Tx); ok {
        // 2. Yes! This means it's a nested call.
        // Create a Savepoint
		savepoint, err := tx.Begin(ctx)
		if err != nil {
			return nil, ctx, fmt.Errorf("failed to create savepoint: %w", err)
		}

        // Return the savepoint (which is also pgx.Tx) and the ORIGINAL ctx.
		return savepoint, ctx, nil
	}

    // 3. No! This means it's a top-level call.
    // Create a NEW transaction, CHECKING OPTIONS.

	if opt, ok := ctx.Value(txOptions).(pgx.TxOptions); ok {
		tx, err := m.db.BeginTx(ctx, opt)
		if err != nil {
			return nil, ctx, fmt.Errorf("failed to start new transaction with options: %w", err)
		}
        // Key point: put the NEW transaction in ctx
		ctx = context.WithValue(ctx, TxKey, tx)
		return tx, ctx, nil
	}

    // No options, just start a transaction
	tx, err := m.db.Begin(ctx)
	if err != nil {
		return nil, ctx, fmt.Errorf("failed to start new transaction: %w", err)
	}

	// Key point: put the NEW transaction in ctx
	ctx = context.WithValue(ctx, TxKey, tx)

	// Return the new transaction and the UPDATED ctx
	return tx, ctx, nil
}

// DoReadOnly executes the provided function within a read-only transaction context.
func (m *Manager) DoReadOnly(ctx context.Context, fn func(ctx context.Context) error) error {
	opts := pgx.TxOptions{
		AccessMode: pgx.ReadOnly,
	}
    
	ctx = WithOptionsCtx(ctx, opts)

    // use panic-safe version of Do
	return m.Do(ctx, fn)
}

func WithOptionsCtx(ctx context.Context, opt pgx.TxOptions) context.Context {
	return context.WithValue(ctx, txOptions, opt)
}