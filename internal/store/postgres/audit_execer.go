package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgconn"
)

type auditEventExecer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}
