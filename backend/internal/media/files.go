package media

import (
	"errors"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io/fs"
	"os"
	"path"
	"strings"
	"unicode"

	"github.com/gabriel-vasile/mimetype"
)

var ErrInvalid = errors.New("invalid or unsupported media file")

type Upload struct {
	URL          string `json:"url"`
	OriginalName string `json:"original_name"`
	StoredName   string `json:"stored_name"`
	MediaKind    string `json:"media_kind"`
	Width        int    `json:"width,omitempty"`
	Height       int    `json:"height,omitempty"`
}

func ValidName(name string) bool {
	return name != "" && name != "." && name != ".." && len(name) <= 255 && !strings.ContainsAny(name, "/\\") && !strings.HasPrefix(name, ".") && strings.IndexFunc(name, unicode.IsControl) < 0
}

func Inspect(file *os.File, name string) (kind string, width, height int, err error) {
	if !ValidName(name) {
		return "", 0, 0, ErrInvalid
	}
	ext := strings.ToLower(path.Ext(name))
	if ext == ".svg" {
		return "", 0, 0, ErrInvalid
	}
	if _, err = file.Seek(0, 0); err != nil {
		return
	}
	typ, e := mimetype.DetectReader(file)
	if e != nil {
		err = e
		return
	}
	kind = "file"
	var allowed []string
	switch ext {
	case ".jpg", ".jpeg":
		kind = "image"
		allowed = []string{"image/jpeg"}
	case ".png":
		kind = "image"
		allowed = []string{"image/png"}
	case ".webp":
		kind = "image"
		allowed = []string{"image/webp"}
	case ".gif":
		kind = "gif"
		allowed = []string{"image/gif"}
	case ".mp4":
		kind = "video"
		allowed = []string{"video/mp4"}
	case ".mov":
		kind = "video"
		allowed = []string{"video/quicktime"}
	case ".webm":
		kind = "video"
		allowed = []string{"video/webm"}
	case ".mp3":
		kind = "audio"
		allowed = []string{"audio/mpeg"}
	case ".wav":
		kind = "audio"
		allowed = []string{"audio/wav", "audio/x-wav"}
	case ".ogg":
		kind = "audio"
		allowed = []string{"audio/ogg", "application/ogg"}
	case ".m4a":
		kind = "audio"
		allowed = []string{"audio/mp4", "audio/x-m4a"}
	}
	if typ.Is("image/svg+xml") || typ.Is("text/html") || typ.Is("application/xhtml+xml") {
		err = ErrInvalid
		return
	}
	if len(allowed) > 0 {
		matched := false
		for _, mime := range allowed {
			matched = matched || typ.Is(mime)
		}
		if !matched {
			err = ErrInvalid
			return
		}
	}
	if ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif" {
		if _, err = file.Seek(0, 0); err != nil {
			return
		}
		var cfg image.Config
		cfg, _, err = image.DecodeConfig(file)
		if err != nil || cfg.Width <= 0 || cfg.Height <= 0 {
			err = ErrInvalid
			return
		}
		width, height = cfg.Width, cfg.Height
	}
	return
}

func SafeExtension(name string) string {
	ext := strings.ToLower(path.Ext(name))
	if len(ext) < 2 || len(ext) > 11 {
		return ".bin"
	}
	for _, r := range ext[1:] {
		if r < 'a' || r > 'z' {
			if r < '0' || r > '9' {
				return ".bin"
			}
		}
	}
	return ext
}

// OpenRegular rejects hidden paths and symlinks as well as escaping the storage root.
func OpenRegular(root *os.Root, name string) (*os.File, error) {
	if !fs.ValidPath(name) || strings.Contains(name, "\\") {
		return nil, fs.ErrNotExist
	}
	parts := strings.Split(name, "/")
	for i, part := range parts {
		if !ValidName(part) {
			return nil, fs.ErrNotExist
		}
		info, err := root.Lstat(strings.Join(parts[:i+1], "/"))
		if err != nil {
			return nil, err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, fs.ErrNotExist
		}
	}
	f, err := root.Open(name)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		f.Close()
		return nil, fs.ErrNotExist
	}
	return f, nil
}

// InlineType allowlists passive raster/audio/video types; all other files download.
func InlineType(name string) string {
	switch strings.ToLower(path.Ext(name)) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".svg":
		return "image/svg+xml"
	case ".mp4":
		return "video/mp4"
	case ".mov":
		return "video/quicktime"
	case ".webm":
		return "video/webm"
	case ".mp3":
		return "audio/mpeg"
	case ".wav":
		return "audio/wav"
	case ".ogg":
		return "audio/ogg"
	case ".m4a":
		return "audio/mp4"
	}
	return ""
}
