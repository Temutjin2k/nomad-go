package postgres

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// IsForeignKeyViolation проверяет, является ли переданная ошибка нарушением внешнего ключа PostgreSQL (SQLSTATE 23503).
//
// Эта функция безопасно работает с обернутыми ошибками благодаря errors.As.
func IsForeignKeyViolation(err error) bool {
	if err == nil {
		return false
	}

	var pgErr *pgconn.PgError

	// errors.As пытается извлечь конкретный тип *pgconn.PgError из всей цепочки ошибок.
	if errors.As(err, &pgErr) {
		// Код '23503' является стандартным SQLSTATE для Foreign Key Violation.
		return pgErr.SQLState() == "23503"
	}

	return false
}
