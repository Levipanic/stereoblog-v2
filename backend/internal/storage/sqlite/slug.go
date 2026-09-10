package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"unicode"
)

func Slugify(title string) string {
	var slug strings.Builder
	separator := false
	hasLetterOrDigit := false
	for _, character := range title {
		switch {
		case unicode.IsLetter(character) || unicode.IsDigit(character):
			if separator && hasLetterOrDigit {
				slug.WriteByte('-')
			}
			slug.WriteRune(unicode.ToLower(character))
			separator = false
			hasLetterOrDigit = true
		case unicode.IsMark(character) && hasLetterOrDigit && !separator:
			slug.WriteRune(character)
		case hasLetterOrDigit:
			separator = true
		}
	}
	if !hasLetterOrDigit {
		return "post"
	}
	return slug.String()
}

func addPostSlugs(ctx context.Context, conn *sql.Conn) error {
	if _, err := conn.ExecContext(ctx, "ALTER TABLE posts ADD COLUMN slug TEXT"); err != nil {
		return err
	}
	rows, err := conn.QueryContext(ctx, "SELECT id, title FROM posts ORDER BY id")
	if err != nil {
		return err
	}
	type post struct {
		id    int64
		title string
	}
	var posts []post
	for rows.Next() {
		var value post
		if err := rows.Scan(&value.id, &value.title); err != nil {
			rows.Close()
			return err
		}
		posts = append(posts, value)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}

	used := make(map[string]bool, len(posts))
	for _, post := range posts {
		base := Slugify(post.title)
		slug := base
		if used[slug] {
			suffixBase := fmt.Sprintf("%s-%d", base, post.id)
			slug = suffixBase
			for suffix := 2; used[slug]; suffix++ {
				slug = fmt.Sprintf("%s-%d", suffixBase, suffix)
			}
		}
		if _, err := conn.ExecContext(ctx, "UPDATE posts SET slug = ? WHERE id = ?", slug, post.id); err != nil {
			return err
		}
		used[slug] = true
	}
	_, err = conn.ExecContext(ctx, "CREATE UNIQUE INDEX idx_posts_slug ON posts(slug)")
	return err
}
