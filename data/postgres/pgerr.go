package postgres

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// SQLSTATE codes used by this package.
const (
	SQLStateUniqueViolation     = "23505"
	SQLStateForeignKeyViolation = "23503"
	SQLStateNotNullViolation    = "23502"
)

type ConstraintInfo struct {
	Code     string // SQLSTATE (e.g. 23505)
	Name     string // constraint name from PG
	Schema   string // optional: pgErr.SchemaName
	Table    string // optional: pgErr.TableName
	Detail   string // optional: pgErr.Detail
	IsUnique bool
}

func Constraint(err error) (ConstraintInfo, bool) {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return ConstraintInfo{}, false
	}
	info := ConstraintInfo{
		Code:     pgErr.Code,
		Name:     pgErr.ConstraintName,
		Schema:   pgErr.SchemaName,
		Table:    pgErr.TableName,
		Detail:   pgErr.Detail,
		IsUnique: pgErr.Code == SQLStateUniqueViolation,
	}
	return info, true
}

// Narrow helper predicates.
func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == SQLStateUniqueViolation
}
func IsForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == SQLStateForeignKeyViolation
}
