package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Levipanic/stereoblog-v2/backend/internal/testfixture"
)

func TestOpenV1WithoutModification(t *testing.T) {
	path := testfixture.V1Database(t)
	before := fileHash(t, path)

	db, schema, err := Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if schema != SchemaV1 {
		t.Fatalf("schema = %q, want %q", schema, SchemaV1)
	}
	var posts, comments int
	if err := db.QueryRow("SELECT count(*) FROM posts").Scan(&posts); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow("SELECT count(*) FROM comments").Scan(&comments); err != nil {
		t.Fatal(err)
	}
	if posts != 2 || comments != 4 {
		t.Fatalf("unexpected rows: posts=%d comments=%d", posts, comments)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if after := fileHash(t, path); after != before {
		t.Fatal("opening and inspecting v1 changed the database file")
	}
}

func TestOpenFreshDatabase(t *testing.T) {
	for _, name := range []string{"fresh.db", "space % ? #.db"} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), name)
			db, schema, err := Open(context.Background(), path)
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			if schema != SchemaFresh {
				t.Fatalf("schema = %q, want %q", schema, SchemaFresh)
			}
			if _, err := os.Stat(path); err != nil {
				t.Fatalf("fresh database was not created: %v", err)
			}
		})
	}
}

func TestConnectionSettingsApplyToEveryConnection(t *testing.T) {
	db, _, err := Open(context.Background(), filepath.Join(t.TempDir(), "settings.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if db.Stats().MaxOpenConnections != maxOpenConnections {
		t.Fatalf("max open connections = %d", db.Stats().MaxOpenConnections)
	}

	connections := make([]*sql.Conn, 0, maxOpenConnections)
	defer func() {
		for _, connection := range connections {
			connection.Close()
		}
	}()
	for range maxOpenConnections {
		connection, err := db.Conn(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		connections = append(connections, connection)
		var foreignKeys, busyTimeout int
		if err := connection.QueryRowContext(context.Background(), "PRAGMA foreign_keys").Scan(&foreignKeys); err != nil {
			t.Fatal(err)
		}
		if err := connection.QueryRowContext(context.Background(), "PRAGMA busy_timeout").Scan(&busyTimeout); err != nil {
			t.Fatal(err)
		}
		if foreignKeys != 1 || busyTimeout != busyTimeoutMS {
			t.Fatalf("connection settings: foreign_keys=%d busy_timeout=%d", foreignKeys, busyTimeout)
		}
	}

	var journalMode string
	if err := connections[0].QueryRowContext(context.Background(), "PRAGMA journal_mode").Scan(&journalMode); err != nil {
		t.Fatal(err)
	}
	if journalMode == "wal" {
		t.Fatal("WAL must remain disabled until deployment filesystem testing")
	}
}

func TestOpenRejectsUnsupportedSchemaWithoutModification(t *testing.T) {
	path := filepath.Join(t.TempDir(), "unsupported.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("CREATE TABLE posts (id INTEGER PRIMARY KEY)"); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	before := fileHash(t, path)

	opened, _, err := Open(context.Background(), path)
	if opened != nil {
		opened.Close()
		t.Fatal("unsupported database remained open")
	}
	if !errors.Is(err, ErrUnsupportedSchema) {
		t.Fatalf("error = %v, want ErrUnsupportedSchema", err)
	}
	if after := fileHash(t, path); after != before {
		t.Fatal("unsupported schema inspection changed the database file")
	}
}

func TestOpenRejectsSchemaLookalike(t *testing.T) {
	path := filepath.Join(t.TempDir(), "lookalike.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	for table, signature := range v1Signatures {
		var columns []string
		for _, column := range strings.Split(signature, ",") {
			columns = append(columns, `"`+strings.SplitN(column, ":", 2)[0]+`"`)
		}
		statement := `CREATE TABLE "` + table + `" (` + strings.Join(columns, ",") + `)`
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	before := fileHash(t, path)

	opened, _, err := Open(context.Background(), path)
	if opened != nil {
		opened.Close()
		t.Fatal("schema lookalike remained open")
	}
	if !errors.Is(err, ErrUnsupportedSchema) {
		t.Fatalf("error = %v, want ErrUnsupportedSchema", err)
	}
	if after := fileHash(t, path); after != before {
		t.Fatal("schema lookalike inspection changed the database file")
	}
}

func TestOpenRejectsDatabaseWithOnlyAView(t *testing.T) {
	path := filepath.Join(t.TempDir(), "view.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("CREATE VIEW user_view AS SELECT 1 AS value"); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	opened, _, err := Open(context.Background(), path)
	if opened != nil {
		opened.Close()
		t.Fatal("database with a user view remained open")
	}
	if !errors.Is(err, ErrUnsupportedSchema) {
		t.Fatalf("error = %v, want ErrUnsupportedSchema", err)
	}
}

func TestPartialUniqueIndexDoesNotSatisfyConstraint(t *testing.T) {
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec("CREATE TABLE sessions (token_hash TEXT); CREATE UNIQUE INDEX partial_token ON sessions(token_hash) WHERE token_hash <> ''"); err != nil {
		t.Fatal(err)
	}
	columns, err := uniqueColumns(context.Background(), db, "sessions")
	if err != nil {
		t.Fatal(err)
	}
	if len(columns) != 0 {
		t.Fatalf("partial unique index was accepted: %v", columns)
	}
}

func TestOpenRejectsCorruptDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "corrupt.db")
	if err := os.WriteFile(path, []byte("not a sqlite database"), 0o600); err != nil {
		t.Fatal(err)
	}
	db, _, err := Open(context.Background(), path)
	if db != nil {
		db.Close()
		t.Fatal("corrupt database remained open")
	}
	if err == nil {
		t.Fatal("corrupt database was accepted")
	}
	if !errors.Is(err, ErrCorruptDatabase) {
		t.Fatalf("error = %v, want ErrCorruptDatabase", err)
	}
}

func TestQuickCheckRejectsTruncatedV1WithoutModification(t *testing.T) {
	path := testfixture.V1Database(t)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Truncate(path, info.Size()-100); err != nil {
		t.Fatal(err)
	}
	before := fileHash(t, path)

	db, _, err := Open(context.Background(), path)
	if db != nil {
		db.Close()
		t.Fatal("truncated database remained open")
	}
	if !errors.Is(err, ErrCorruptDatabase) {
		t.Fatalf("error = %v, want ErrCorruptDatabase", err)
	}
	if after := fileHash(t, path); after != before {
		t.Fatal("corrupt database inspection changed the database file")
	}
}

func fileHash(t *testing.T, path string) [sha256.Size]byte {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return sha256.Sum256(contents)
}
