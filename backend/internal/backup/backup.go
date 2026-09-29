package backup

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Levipanic/stereoblog-v2/backend/internal/media"
	database "github.com/Levipanic/stereoblog-v2/backend/internal/storage/sqlite"
)

type Manifest struct {
	FormatVersion  int    `json:"format_version"`
	CreatedAt      string `json:"created_at"`
	SchemaVersion  int    `json:"schema_version"`
	Posts          int    `json:"posts"`
	Comments       int    `json:"comments"`
	MediaFiles     int    `json:"media_files"`
	DatabaseSHA256 string `json:"database_sha256"`
}
type Archive struct {
	Path      string
	directory string
	Manifest  Manifest
}

func (a *Archive) Close() error { return os.RemoveAll(a.directory) }

func Create(ctx context.Context, db *sql.DB, uploads string) (archive *Archive, err error) {
	dir, err := os.MkdirTemp("", "stereodamage-backup-")
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			os.RemoveAll(dir)
		}
	}()
	snapshot := filepath.Join(dir, "blog.db")
	if err = database.Snapshot(ctx, db, snapshot); err != nil {
		return nil, err
	}
	report, err := database.Audit(ctx, snapshot, uploads)
	if err != nil {
		return nil, err
	}
	if !report.Compatible() {
		return nil, errors.New("snapshot compatibility audit failed; refusing incomplete backup")
	}
	archive = &Archive{Path: filepath.Join(dir, "backup.zip"), directory: dir, Manifest: Manifest{FormatVersion: 1, CreatedAt: time.Now().UTC().Format(time.RFC3339), SchemaVersion: report.MigrationVersion, Posts: report.Posts, Comments: report.Comments}}
	output, err := os.OpenFile(archive.Path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}
	defer output.Close()
	z := zip.NewWriter(output)
	defer z.Close()
	dbFile, err := os.Open(snapshot)
	if err != nil {
		return nil, err
	}
	hash := sha256.New()
	err = addFile(ctx, z, "data/blog.db", io.TeeReader(dbFile, hash))
	dbFile.Close()
	if err != nil {
		return nil, err
	}
	archive.Manifest.DatabaseSHA256 = hex.EncodeToString(hash.Sum(nil))
	root, err := os.OpenRoot(uploads)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if err == nil {
		defer root.Close()
		err = fs.WalkDir(root.FS(), ".", func(name string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			if name == "." {
				return nil
			}
			if strings.HasPrefix(entry.Name(), ".") {
				if entry.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("upload symlink is not portable: %s", name)
			}
			if entry.IsDir() {
				return nil
			}
			file, err := media.OpenRegular(root, name)
			if err != nil {
				return err
			}
			defer file.Close()
			if err := addFile(ctx, z, "uploads/"+name, file); err != nil {
				return err
			}
			archive.Manifest.MediaFiles++
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	manifest, err := z.Create("manifest.json")
	if err != nil {
		return nil, err
	}
	if err = json.NewEncoder(manifest).Encode(archive.Manifest); err != nil {
		return nil, err
	}
	if err = z.Close(); err != nil {
		return nil, err
	}
	if err = output.Sync(); err != nil {
		return nil, err
	}
	if err = output.Close(); err != nil {
		return nil, err
	}
	return archive, nil
}

func addFile(ctx context.Context, z *zip.Writer, name string, r io.Reader) error {
	header := &zip.FileHeader{Name: name, Method: zip.Store}
	header.SetMode(0600)
	w, err := z.CreateHeader(header)
	if err != nil {
		return err
	}
	// ponytail: store media without compression to bound CPU on the VPS; compress only if measured savings justify it.
	_, err = io.Copy(w, contextReader{ctx, r})
	return err
}

type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.r.Read(p)
}
