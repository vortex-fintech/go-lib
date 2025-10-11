package postgres

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// IsUniqueViolation — true, если ошибка = уникальный индекс (23505).
func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// Constraint — вытащить код и имя констрейнта из ошибки PG.
func Constraint(err error) (code, constraint string, ok bool) {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code, pgErr.ConstraintName, true
	}
	return "", "", false
}
