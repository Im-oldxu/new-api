package model

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/ClickHouse/clickhouse-go/v2/lib/proto"
	"github.com/Im-oldxu/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// 数据库驱动错误消息可能内联数据值，日志只保留错误码。
func TestSanitizeDBErrorStripsDriverMessage(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		want   string
		leaked string
	}{
		{
			name:   "mysql duplicate entry",
			err:    &mysql.MySQLError{Number: 1062, Message: "Duplicate entry 'secret-value' for key 'users.idx'"},
			want:   "mysql error 1062",
			leaked: "secret-value",
		},
		{
			name:   "postgres unique violation",
			err:    &pgconn.PgError{Code: "23505", Message: "duplicate key value", Detail: "Key (k)=(secret-value) already exists."},
			want:   "postgres error SQLSTATE 23505",
			leaked: "secret-value",
		},
		{
			name:   "postgres pooler prepared statement conflict gets remediation hint",
			err:    &pgconn.PgError{Severity: "FATAL", Code: "08P01", Message: "prepared statement name is already in use: stmt_secret-value"},
			want:   "postgres error SQLSTATE 08P01: prepared statement conflict with a transaction-pooling proxy (PgBouncer/Neon/Supabase); other clients sharing this database must disable prepared statements, or upgrade PgBouncer to >=1.21 with max_prepared_statements enabled",
			leaked: "secret-value",
		},
		{
			name:   "clickhouse exception",
			err:    &proto.Exception{Code: 241, Message: "Memory limit exceeded while processing 'secret-value'"},
			want:   "clickhouse error 241",
			leaked: "secret-value",
		},
		{
			name:   "wrapped driver error",
			err:    fmt.Errorf("exec failed: %w", &mysql.MySQLError{Number: 1064, Message: "syntax error near 'secret-value'"}),
			want:   "mysql error 1064",
			leaked: "secret-value",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := sanitizeDBError(tc.err)
			require.Error(t, got)
			assert.Equal(t, tc.want, got.Error())
			assert.NotContains(t, got.Error(), tc.leaked)
		})
	}
}

func TestSanitizeDBErrorSQLiteDriver(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	execErr := db.Exec("INSERT INTO missing_table (k) VALUES (?)", "secret-value").Error
	require.Error(t, execErr)

	got := sanitizeDBError(execErr)
	assert.Regexp(t, `^sqlite error \d+$`, got.Error())
	assert.NotContains(t, got.Error(), "secret-value")
}

func TestSanitizeDBErrorKeepsNonDriverErrors(t *testing.T) {
	err := fmt.Errorf("dial tcp 127.0.0.1:3306: connect: connection refused")
	assert.Equal(t, err, sanitizeDBError(err))
}

// 保护契约:经 gorm 真实链路,错误日志同时满足 SQL 参数化、驱动错误脱敏、
// 调用点归因到业务代码，调试模式也不输出数据值。
func TestGormLoggerEndToEndSanitizedOutput(t *testing.T) {
	previousDebug := common.DebugEnabled
	t.Cleanup(func() { common.DebugEnabled = previousDebug })

	for _, debug := range []bool{false, true} {
		t.Run(fmt.Sprintf("debug=%t", debug), func(t *testing.T) {
			common.DebugEnabled = debug
			var output bytes.Buffer
			db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: newGormLogger(&output)})
			require.NoError(t, err)
			if debug {
				db = db.Debug()
			}
			require.Error(t, db.Exec("SELECT * FROM missing_table WHERE k = ?", "secret-value").Error)
			assert.Contains(t, output.String(), "k = ?")
			assert.Contains(t, output.String(), "sqlite error")
			assert.Contains(t, output.String(), "gorm_logger_test.go")
			assert.NotContains(t, output.String(), "no such table")
			assert.NotContains(t, output.String(), "secret-value")
		})
	}
}
