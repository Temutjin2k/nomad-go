package trm

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TxManager interface {
	Do(ctx context.Context, fn func(ctx context.Context) error) error
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
func (m *Manager) Do(ctx context.Context, fn func(ctx context.Context) error) error {
	var tx pgx.Tx
	var err error

	// Get or start a transaction
	tx, ctx, err = m.getTransactionFromContext(ctx)
	if err != nil {
		return err
	}

	// Execute the function within the transaction context
	if err := fn(ctx); err != nil {
		// If error occurs, rollback the transaction
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
			return fmt.Errorf("failed to rollback tx: %w", rollbackErr)
		}
		return err
	}

	// If no error, commit the transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit tx: %w", err)
	}

	return nil
}

// getTransactionFromContext tries to retrieve an existing transaction from the context.
// If no transaction exists, it starts a new one and returns it along with an updated context.
func (m *Manager) getTransactionFromContext(ctx context.Context) (pgx.Tx, context.Context, error) {
	// Check if a transaction already exists in the context
	if tx := ctx.Value(TxKey); tx != nil {
		// If transaction exists, return it
		if tx, ok := tx.(pgx.Tx); ok {
			return tx, ctx, nil
		}
		return nil, ctx, fmt.Errorf("invalid transaction type in context")
	}

	// Check if transaction options are provided in the context
	if opt, ok := ctx.Value(txOptions).(pgx.TxOptions); ok {
		tx, err := m.db.BeginTx(ctx, opt)
		if err != nil {
			return nil, ctx, fmt.Errorf("failed to start new transaction with options: %w", err)
		}
		ctx = context.WithValue(ctx, TxKey, tx)
		return tx, ctx, nil
	}

	tx, err := m.db.Begin(ctx)
	if err != nil {
		return nil, ctx, fmt.Errorf("failed to start new transaction: %w", err)
	}

	// Save the new transaction in the context
	ctx = context.WithValue(ctx, TxKey, tx)

	// Return the new transaction and updated context
	return tx, ctx, nil
}

func WithOptionsCtx(ctx context.Context, opt pgx.TxOptions) context.Context {
	return context.WithValue(ctx, txOptions, opt)
}
