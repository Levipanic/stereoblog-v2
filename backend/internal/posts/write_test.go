package posts

import (
	"context"
	"encoding/json"
	"github.com/Levipanic/stereoblog-v2/backend/internal/content"
	"testing"
)

func TestLegacyPostEditPreservesContent(t *testing.T) {
	r := NewRepository(migratedFixture(t))
	ctx := context.Background()
	p, err := r.AdminByID(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	in := WriteInput{Title: "Edited legacy title", Blocks: p.Blocks}
	if err := in.Validate(content.Limits{MaxBlocks: 60, MaxText: 4000, MaxMediaText: 500}); err != nil {
		t.Fatal(err)
	}
	if err := r.Update(ctx, p.ID, in); err != nil {
		t.Fatal(err)
	}
	after, err := r.AdminByID(ctx, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Slug != p.Slug || after.CreatedAt != p.CreatedAt || string(after.Preview) != string(p.Preview) {
		t.Fatal("legacy metadata changed")
	}
	var beforeBlocks, afterBlocks []map[string]any
	json.Unmarshal(p.Blocks, &beforeBlocks)
	json.Unmarshal(after.Blocks, &afterBlocks)
	if len(beforeBlocks) != len(afterBlocks) {
		t.Fatal("blocks lost")
	}
	for i, b := range beforeBlocks {
		for _, key := range []string{"type", "text", "src", "caption", "alt", "name", "spoiler", "mediaKind"} {
			if b[key] != nil && b[key] != "" && b[key] != afterBlocks[i][key] {
				t.Fatalf("block %d %s changed", i, key)
			}
		}
	}
	p, err = r.AdminByID(ctx, 2)
	if err != nil {
		t.Fatal(err)
	}
	in = WriteInput{Title: p.Title, Blocks: p.Blocks}
	if err := in.Validate(content.Limits{MaxBlocks: 60, MaxText: 4000, MaxMediaText: 500}); err == nil {
		t.Fatal("unsupported legacy block silently accepted")
	}
}
