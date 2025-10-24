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
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				fmt.Printf("failed to rollback tx after panic: %v\n", rbErr)
			}
			panic(p)
		}
		if err != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				err = fmt.Errorf("rollback failed: %v (original error: %w)", rbErr, err)
			}
			return
		}
		if commitErr := tx.Commit(ctx); commitErr != nil {
			err = fmt.Errorf("failed to commit tx: %w", commitErr)
		}
	}()

	err = fn(ctx)
	return err
}

// getTransactionFromContext returns the "current layer" tx and an updated ctx:
// - If a tx exists, it opens a SAVEPOINT (nested tx), stores that in ctx, and returns it.
// - If no tx exists, it begins a new tx (honoring any options in ctx), stores it in ctx, and returns it.
func (m *Manager) getTransactionFromContext(ctx context.Context) (pgx.Tx, context.Context, error) {
	// Nested layer → savepoint
	if current, ok := ctx.Value(TxKey).(pgx.Tx); ok && current != nil {
		savepoint, err := current.Begin(ctx)
		if err != nil {
			return nil, ctx, fmt.Errorf("failed to create savepoint: %w", err)
		}
		ctx = context.WithValue(ctx, TxKey, savepoint)
		return savepoint, ctx, nil
	}

	// Top-level → read clean Options first
	switch v := ctx.Value(txOptions).(type) {
	case Options:
		tx, err := m.db.BeginTx(ctx, toPGXOptions(v))
		if err != nil {
			return nil, ctx, fmt.Errorf("failed to start new transaction with options: %w", err)
		}
		ctx = context.WithValue(ctx, TxKey, tx)
		return tx, ctx, nil
	case pgx.TxOptions: // backward-compat if someone still stores pgx.TxOptions directly
		tx, err := m.db.BeginTx(ctx, v)
		if err != nil {
			return nil, ctx, fmt.Errorf("failed to start new transaction with options: %w", err)
		}
		ctx = context.WithValue(ctx, TxKey, tx)
		return tx, ctx, nil
	}

	// No options → default
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
	// Merge into existing trm.Options if present; otherwise set minimal RO.
	switch v := ctx.Value(txOptions).(type) {
	case Options:
		v.AccessMode = AccessModeReadOnly
		ctx = WithOptions(ctx, v)
	case pgx.TxOptions: // if legacy is in play, convert & merge to clean options
		o := fromPGXOptions(v)
		o.AccessMode = AccessModeReadOnly
		ctx = WithOptions(ctx, o)
	default:
		ctx = WithOptions(ctx, Options{AccessMode: AccessModeReadOnly})
	}
	return m.Do(ctx, fn)
}

type AccessMode uint8

const (
	AccessModeReadWrite AccessMode = iota
	AccessModeReadOnly
)

type IsolationLevel uint8

const (
	IsoReadUncommited IsolationLevel = iota
	IsoReadCommitted
	IsoRepeatableRead
	IsoSerializable
)

type DeferrableLevel uint8

const (
	Deferrable DeferrableLevel = iota
	NotDeferrable
)

// Options is DB-agnostic and safe for usecase layer.
type Options struct {
	AccessMode AccessMode
	IsoLevel   IsolationLevel
	Deferrable DeferrableLevel
	// Future additions go here without leaking driver types.
}

// WithOptions stores trm.Options in context (preferred for clean architecture).
func WithOptions(ctx context.Context, opt Options) context.Context {
	return context.WithValue(ctx, txOptions, opt)
}

// WithOptionsCtx keeps backward-compat for existing code that passes pgx.TxOptions.
// It converts pgx options into trm.Options and stores them under the same key.
func WithOptionsCtx(ctx context.Context, opt pgx.TxOptions) context.Context {
	return WithOptions(ctx, fromPGXOptions(opt))
}
func toPGXOptions(o Options) pgx.TxOptions {
	var am pgx.TxAccessMode
	switch o.AccessMode {
	case AccessModeReadOnly:
		am = pgx.ReadOnly
	case AccessModeReadWrite:
		am = pgx.ReadWrite
	default:
		am = pgx.ReadWrite // pgx default
	}

	var iso pgx.TxIsoLevel
	switch o.IsoLevel {
	case IsoReadUncommited:
		iso = pgx.ReadUncommitted
	case IsoReadCommitted:
		iso = pgx.ReadCommitted
	case IsoRepeatableRead:
		iso = pgx.RepeatableRead
	case IsoSerializable:
		iso = pgx.Serializable
	default:
		iso = pgx.ReadCommitted // pgx default
	}

	var df pgx.TxDeferrableMode
	switch o.Deferrable {
	case Deferrable:
		df = pgx.Deferrable
	case NotDeferrable:
		df = pgx.NotDeferrable
	}

	return pgx.TxOptions{
		IsoLevel:       iso,
		AccessMode:     am,
		DeferrableMode: df,
	}
}

func fromPGXOptions(p pgx.TxOptions) Options {
	var am AccessMode
	switch p.AccessMode {
	case pgx.ReadOnly:
		am = AccessModeReadOnly
	case pgx.ReadWrite:
		am = AccessModeReadWrite
	}

	var iso IsolationLevel
	switch p.IsoLevel {
	case pgx.ReadUncommitted:
		iso = IsoReadUncommited
	case pgx.ReadCommitted:
		iso = IsoReadCommitted
	case pgx.RepeatableRead:
		iso = IsoRepeatableRead
	case pgx.Serializable:
		iso = IsoSerializable
	}

	var df DeferrableLevel
	switch p.DeferrableMode {
	case pgx.Deferrable:
		df = Deferrable
	case pgx.NotDeferrable:
		df = NotDeferrable
	}

	return Options{
		AccessMode: am,
		IsoLevel:   iso,
		Deferrable: df,
	}
}
