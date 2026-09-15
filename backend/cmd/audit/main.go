package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"

	database "github.com/Levipanic/stereoblog-v2/backend/internal/storage/sqlite"
)

func main() {
	os.Exit(run(context.Background(), os.Args[1:], os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("audit", flag.ContinueOnError)
	flags.SetOutput(stderr)
	databasePath := flags.String("db", envOr("DB_PATH", "../data/blog.db"), "path to an already migrated SQLite database copy")
	uploadsPath := flags.String("uploads", envOr("UPLOADS_PATH", "../uploads"), "path to its uploads directory")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	report, err := database.Audit(ctx, *databasePath, *uploadsPath)
	if err != nil {
		fmt.Fprintf(stderr, "compatibility audit failed: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "integrity: ok\n")
	fmt.Fprintf(stdout, "migration_version: %d/%d\n", report.MigrationVersion, report.ExpectedVersion)
	fmt.Fprintf(stdout, "posts: %d\ncomments: %d\n", report.Posts, report.Comments)
	fmt.Fprintf(stdout, "post_likes: %d\ncomment_likes: %d\n", report.PostLikes, report.CommentLikes)
	fmt.Fprintf(stdout, "post_like_events: %d\ncomment_like_events: %d\n", report.PostLikeEvents, report.CommentLikeEvents)
	fmt.Fprintf(stdout, "foreign_key_violations: %d\ncross_post_parents: %d\n", report.ForeignKeyViolations, report.CrossPostParents)
	fmt.Fprintf(stdout, "media_references: %d\nmissing_uploads: %d\n", report.MediaReferences, len(report.MissingUploads))
	for _, path := range report.MissingUploads {
		fmt.Fprintf(stdout, "  %s\n", path)
	}
	fmt.Fprintf(stdout, "unparseable_posts: %d\ninvalid_previews: %d\nunknown_blocks: %d\n", len(report.UnparseablePosts), len(report.InvalidPreviewPosts), report.UnknownBlocks)
	if len(report.UnparseablePosts) != 0 {
		fmt.Fprintf(stdout, "  unparseable post IDs: %v\n", report.UnparseablePosts)
	}
	if len(report.InvalidPreviewPosts) != 0 {
		fmt.Fprintf(stdout, "  invalid preview post IDs: %v\n", report.InvalidPreviewPosts)
	}
	if report.Compatible() {
		fmt.Fprintln(stdout, "result: PASS")
		return 0
	}
	fmt.Fprintln(stdout, "result: FAIL")
	return 1
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
