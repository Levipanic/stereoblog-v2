package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"slices"
	"strings"

	sqliteDriver "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

const (
	maxOpenConnections = 4
	busyTimeoutMS      = 5000
)

type Schema string

const (
	SchemaFresh Schema = "fresh"
	SchemaV1    Schema = "v1"
)

var (
	ErrUnsupportedSchema = errors.New("unsupported SQLite schema")
	ErrCorruptDatabase   = errors.New("corrupt SQLite database")
)

var v1Signatures = map[string]string{
	"posts":                  "id:INTEGER:0:-:1,title:TEXT:1:-:0,blocks_json:TEXT:1:-:0,likes_count:INTEGER:1:0:0,created_at:TEXT:1:datetime('now'):0,preview_media:TEXT:0:NULL:0",
	"comments":               "id:INTEGER:0:-:1,post_id:INTEGER:1:-:0,parent_id:INTEGER:0:-:0,name:TEXT:0:-:0,content:TEXT:1:-:0,created_at:TEXT:1:datetime('now'):0,likes_count:INTEGER:1:0:0,status:TEXT:1:'visible':0,moderation_reason:TEXT:0:-:0,text_hash:TEXT:0:-:0,text_fingerprint:TEXT:0:-:0",
	"like_events":            "id:INTEGER:0:-:1,post_id:INTEGER:1:-:0,ip_hash:TEXT:1:-:0,created_at:TEXT:1:datetime('now'):0",
	"comment_like_events":    "id:INTEGER:0:-:1,comment_id:INTEGER:1:-:0,ip_hash:TEXT:1:-:0,created_at:TEXT:1:datetime('now'):0",
	"comment_attempts":       "id:INTEGER:0:-:1,ip_hash:TEXT:1:-:0,post_id:INTEGER:0:-:0,status:TEXT:1:-:0,reason:TEXT:0:-:0,content:TEXT:0:-:0,text_hash:TEXT:0:-:0,fingerprint:TEXT:0:-:0,created_at:TEXT:1:datetime('now'):0",
	"comment_mutes":          "id:INTEGER:0:-:1,ip_hash:TEXT:1:-:0,reason:TEXT:0:-:0,muted_until:TEXT:1:-:0,mute_count:INTEGER:1:1:0,created_at:TEXT:1:datetime('now'):0",
	"comment_challenge_uses": "token_hash:TEXT:0:-:1,post_id:INTEGER:1:-:0,used_count:INTEGER:1:0:0,first_used_at:TEXT:1:datetime('now'):0,last_used_at:TEXT:1:datetime('now'):0",
	"admin_sessions":         "id:INTEGER:0:-:1,token_hash:TEXT:1:-:0,created_at:TEXT:1:datetime('now'):0,expires_at:TEXT:1:-:0",
}

var requiredForeignKeys = map[string][]string{
	"comments":            {"post_id->posts.id CASCADE"},
	"like_events":         {"post_id->posts.id CASCADE"},
	"comment_like_events": {"comment_id->comments.id CASCADE"},
}

var requiredUniqueColumns = map[string][]string{
	"admin_sessions": {"token_hash"},
	"comment_mutes":  {"ip_hash"},
}

func Open(ctx context.Context, path string) (*sql.DB, Schema, error) {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return nil, "", fmt.Errorf("resolve database path: %w", err)
	}
	dsn := url.URL{Scheme: "file", Path: filepath.ToSlash(absolutePath)}
	query := dsn.Query()
	query.Set("_foreign_keys", "on")
	query.Set("_busy_timeout", fmt.Sprint(busyTimeoutMS))
	dsn.RawQuery = query.Encode()

	db, err := sql.Open("sqlite", dsn.String())
	if err != nil {
		return nil, "", fmt.Errorf("open database: %w", err)
	}
	db.SetMaxOpenConns(maxOpenConnections)
	db.SetMaxIdleConns(maxOpenConnections)
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, "", databaseError("open database", err)
	}

	schema, err := Inspect(ctx, db)
	if err != nil {
		db.Close()
		return nil, "", err
	}
	return db, schema, nil
}

func Inspect(ctx context.Context, db *sql.DB) (Schema, error) {
	if err := quickCheck(ctx, db); err != nil {
		return "", err
	}
	rows, err := db.QueryContext(ctx, "SELECT type, name FROM sqlite_schema WHERE type IN ('table', 'view', 'trigger') AND name NOT LIKE 'sqlite_%' ORDER BY type, name")
	if err != nil {
		return "", fmt.Errorf("inspect database schema: %w", err)
	}
	var tables []string
	objects := 0
	for rows.Next() {
		var objectType, name string
		if err := rows.Scan(&objectType, &name); err != nil {
			rows.Close()
			return "", fmt.Errorf("inspect database schema: %w", err)
		}
		objects++
		if objectType == "table" {
			tables = append(tables, name)
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return "", fmt.Errorf("inspect database schema: %w", err)
	}
	if err := rows.Close(); err != nil {
		return "", fmt.Errorf("inspect database schema: %w", err)
	}
	if objects == 0 {
		return SchemaFresh, nil
	}
	if len(tables) == 0 {
		return "", fmt.Errorf("%w: no recognized tables", ErrUnsupportedSchema)
	}

	var incompatible []string
	for table, signature := range v1Signatures {
		if !slices.Contains(tables, table) {
			incompatible = append(incompatible, "table "+table)
			continue
		}
		actual, err := columnSignature(ctx, db, table)
		if err != nil {
			return "", err
		}
		if actual != signature && !strings.HasPrefix(actual, signature+",") {
			incompatible = append(incompatible, "table "+table+" columns")
		}
		foreignKeys, err := foreignKeys(ctx, db, table)
		if err != nil {
			return "", err
		}
		for _, required := range requiredForeignKeys[table] {
			if !slices.Contains(foreignKeys, required) {
				incompatible = append(incompatible, "table "+table+" foreign key")
			}
		}
		if table == "comments" {
			for _, foreignKey := range foreignKeys {
				if strings.HasPrefix(foreignKey, "parent_id->") && foreignKey != "parent_id->comments.id CASCADE" {
					incompatible = append(incompatible, "table comments parent foreign key")
				}
			}
		}
		if required := requiredUniqueColumns[table]; len(required) != 0 {
			unique, err := uniqueColumns(ctx, db, table)
			if err != nil {
				return "", err
			}
			for _, columns := range required {
				if !slices.Contains(unique, columns) {
					incompatible = append(incompatible, "table "+table+" unique constraint")
				}
			}
		}
	}
	for _, table := range []string{"posts", "comments", "like_events", "comment_like_events", "comment_attempts", "comment_mutes", "admin_sessions"} {
		var ddl string
		if err := db.QueryRowContext(ctx, "SELECT sql FROM sqlite_schema WHERE type = 'table' AND name = ?", table).Scan(&ddl); err != nil {
			continue
		}
		if !strings.Contains(strings.ToUpper(ddl), "AUTOINCREMENT") {
			incompatible = append(incompatible, "table "+table+" autoincrement")
		}
	}
	if len(incompatible) != 0 {
		slices.Sort(incompatible)
		return "", fmt.Errorf("%w: incompatible %s", ErrUnsupportedSchema, strings.Join(incompatible, ", "))
	}
	return SchemaV1, nil
}

func quickCheck(ctx context.Context, db *sql.DB) error {
	var result string
	if err := db.QueryRowContext(ctx, "PRAGMA quick_check(1)").Scan(&result); err != nil {
		return databaseError("check database", err)
	}
	if result != "ok" {
		return fmt.Errorf("%w: quick check failed", ErrCorruptDatabase)
	}
	return nil
}

func columnSignature(ctx context.Context, db *sql.DB, table string) (string, error) {
	rows, err := db.QueryContext(ctx, "SELECT name, [type], [notnull], dflt_value, pk FROM pragma_table_info(?) ORDER BY cid", table)
	if err != nil {
		return "", fmt.Errorf("inspect table %s: %w", table, err)
	}
	defer rows.Close()
	var columns []string
	for rows.Next() {
		var name, columnType string
		var notNull, primaryKey int
		var defaultValue sql.NullString
		if err := rows.Scan(&name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			return "", fmt.Errorf("inspect table %s: %w", table, err)
		}
		value := "-"
		if defaultValue.Valid {
			value = defaultValue.String
		}
		columns = append(columns, fmt.Sprintf("%s:%s:%d:%s:%d", name, columnType, notNull, value, primaryKey))
	}
	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("inspect table %s: %w", table, err)
	}
	return strings.Join(columns, ","), nil
}

func foreignKeys(ctx context.Context, db *sql.DB, table string) ([]string, error) {
	rows, err := db.QueryContext(ctx, "SELECT [from], [table], [to], on_delete FROM pragma_foreign_key_list(?)", table)
	if err != nil {
		return nil, fmt.Errorf("inspect table %s foreign keys: %w", table, err)
	}
	defer rows.Close()
	var keys []string
	for rows.Next() {
		var from, targetTable, to, onDelete string
		if err := rows.Scan(&from, &targetTable, &to, &onDelete); err != nil {
			return nil, fmt.Errorf("inspect table %s foreign keys: %w", table, err)
		}
		keys = append(keys, fmt.Sprintf("%s->%s.%s %s", from, targetTable, to, onDelete))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("inspect table %s foreign keys: %w", table, err)
	}
	return keys, nil
}

func uniqueColumns(ctx context.Context, db *sql.DB, table string) ([]string, error) {
	rows, err := db.QueryContext(ctx, "SELECT name FROM pragma_index_list(?) WHERE [unique] = 1 AND partial = 0", table)
	if err != nil {
		return nil, fmt.Errorf("inspect table %s unique constraints: %w", table, err)
	}
	var indexes []string
	for rows.Next() {
		var index string
		if err := rows.Scan(&index); err != nil {
			rows.Close()
			return nil, fmt.Errorf("inspect table %s unique constraints: %w", table, err)
		}
		indexes = append(indexes, index)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("inspect table %s unique constraints: %w", table, err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("inspect table %s unique constraints: %w", table, err)
	}

	var result []string
	for _, index := range indexes {
		indexRows, err := db.QueryContext(ctx, "SELECT name FROM pragma_index_info(?) ORDER BY seqno", index)
		if err != nil {
			return nil, fmt.Errorf("inspect table %s unique constraints: %w", table, err)
		}
		var columns []string
		for indexRows.Next() {
			var column string
			if err := indexRows.Scan(&column); err != nil {
				indexRows.Close()
				return nil, fmt.Errorf("inspect table %s unique constraints: %w", table, err)
			}
			columns = append(columns, column)
		}
		if err := indexRows.Err(); err != nil {
			indexRows.Close()
			return nil, fmt.Errorf("inspect table %s unique constraints: %w", table, err)
		}
		if err := indexRows.Close(); err != nil {
			return nil, fmt.Errorf("inspect table %s unique constraints: %w", table, err)
		}
		result = append(result, strings.Join(columns, ","))
	}
	return result, nil
}

func databaseError(action string, err error) error {
	var sqliteErr *sqliteDriver.Error
	if errors.As(err, &sqliteErr) {
		code := sqliteErr.Code() & 0xff
		if code == sqlite3.SQLITE_CORRUPT || code == sqlite3.SQLITE_NOTADB {
			return fmt.Errorf("%s: %w: %v", action, ErrCorruptDatabase, err)
		}
	}
	return fmt.Errorf("%s: %w", action, err)
}
