package content

import (
	"context"
	"database/sql"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/Levipanic/stereoblog-v2/backend/internal/testfixture"
	_ "modernc.org/sqlite"
)

var testLimits = Limits{MaxBlocks: 60, MaxText: 4000, MaxMediaText: 500}

func TestFixtureLegacyPostsParseWithoutRewriting(t *testing.T) {
	db, err := sql.Open("sqlite", testfixture.V1Database(t))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	rows, err := db.QueryContext(context.Background(), "SELECT id, blocks_json FROM posts ORDER BY id")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var raw string
		if err := rows.Scan(&id, &raw); err != nil {
			t.Fatal(err)
		}
		blocks, err := ParseBlocksJSON(raw)
		if err != nil {
			t.Fatalf("post %d: %v", id, err)
		}
		if id == 1 {
			want := []BlockType{Paragraph, Heading, Quote, Divider, Media, Media, Media, Media, Media}
			if got := blockTypes(blocks); !reflect.DeepEqual(got, want) {
				t.Fatalf("post 1 types = %v, want %v", got, want)
			}
			if blocks[4].MediaKind != Image || blocks[4].Src != "/uploads/fixture-image.jpg" || !blocks[4].Spoiler || blocks[4].Alt != "Описание" {
				t.Fatalf("legacy media changed: %#v", blocks[4])
			}
			encoded, err := MarshalBlocksJSON(blocks)
			if err != nil {
				t.Fatal(err)
			}
			if reparsed, err := ValidateBlocksJSON(encoded, testLimits); err != nil || !reflect.DeepEqual(reparsed, blocks) {
				t.Fatalf("legacy round trip failed: error=%v\nblocks=%#v\nreparsed=%#v", err, blocks, reparsed)
			}
		} else {
			want := []BlockType{Paragraph, Unknown, Unknown}
			if got := blockTypes(blocks); !reflect.DeepEqual(got, want) {
				t.Fatalf("post 2 types = %v, want %v", got, want)
			}
			if len(blocks[1].Raw) == 0 || len(blocks[2].Raw) == 0 {
				t.Fatal("unsupported legacy blocks were not retained for diagnostics")
			}
			if _, err := MarshalBlocksJSON(blocks); err == nil {
				t.Fatal("canonical save silently accepted unsupported legacy blocks")
			}
			publicJSON, err := json.Marshal(blocks)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(publicJSON), "future-block") || strings.Contains(string(publicJSON), "payload") {
				t.Fatal("opaque unknown JSON leaked through public serialization")
			}
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
}

func TestRichInlineGeneratesStablePlainFallback(t *testing.T) {
	raw := `[{
		"type":"paragraph",
		"text":"caller supplied text is ignored",
		"content":[
			{"type":"text","text":" Hello ","marks":[{"type":"bold"}]},
			{"type":"text","text":"world ","marks":[{"type":"italic"},{"type":"link","href":" https://example.com/path "},{"type":"code"}]}
		]
	}]`
	blocks, err := ValidateBlocksJSON(raw, testLimits)
	if err != nil {
		t.Fatal(err)
	}
	if blocks[0].Text != "Hello world" || blocks[0].Content[1].Marks[1].Href != "https://example.com/path" {
		t.Fatalf("rich content was not normalized: %#v", blocks[0])
	}
	encoded, err := MarshalBlocksJSON(blocks)
	if err != nil {
		t.Fatal(err)
	}
	want := `[{"type":"paragraph","text":"Hello world","content":[{"type":"text","text":" Hello ","marks":[{"type":"bold"}]},{"type":"text","text":"world ","marks":[{"type":"italic"},{"type":"link","href":"https://example.com/path"},{"type":"code"}]}]}]`
	if encoded != want {
		t.Fatalf("encoded rich content = %s\nwant = %s", encoded, want)
	}
}

func TestPublicReadFallsBackAndCoercesLegacyHeadingLevel(t *testing.T) {
	raw := `[
		{"type":"paragraph","text":"legacy fallback","content":[{"type":"html","html":"<script>"}]},
		{"type":"heading","level":"2","text":"Legacy heading"},
		{"type":"heading","level":2.0,"text":"Numeric heading"},
		{"type":"heading","level":"2e0","text":"Scientific heading"},
		{"type":"future","html":"<script>alert(1)</script>"}
	]`
	blocks, err := ParseBlocksJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if blocks[0].Type != Paragraph || blocks[0].Text != "legacy fallback" || blocks[0].Content != nil {
		t.Fatalf("invalid rich content did not fall back safely: %#v", blocks[0])
	}
	if blocks[1].Type != Heading || blocks[1].Level != 2 {
		t.Fatalf("legacy string heading level was not accepted: %#v", blocks[1])
	}
	if blocks[2].Type != Heading || blocks[2].Level != 2 || blocks[3].Type != Heading || blocks[3].Level != 2 {
		t.Fatalf("legacy numeric heading levels were not accepted: %#v %#v", blocks[2], blocks[3])
	}
	if blocks[4].Type != Unknown {
		t.Fatalf("unknown public block = %#v", blocks[4])
	}
}

func TestAdminWriteRejectsInvalidRichContentAndLinks(t *testing.T) {
	invalid := []string{
		`[{"type":"future","text":"x"}]`,
		`[{"type":"heading","level":"2","text":"x"}]`,
		`[{"type":"paragraph","text":"fallback","content":null}]`,
		`[{"type":"paragraph","content":[]}]`,
		`[{"type":"paragraph","content":[{"type":"html","text":"x"}]}]`,
		`[{"type":"paragraph","content":[{"type":"text","text":"x","marks":[{"type":"underline"}]}]}]`,
		`[{"type":"paragraph","content":[{"type":"text","text":"x","marks":[{"type":"bold"},{"type":"bold"}]}]}]`,
		`[{"type":"paragraph","content":[{"type":"text","text":"x","marks":[{"type":"bold","href":"https://example.com"}]}]}]`,
		`[{"type":"paragraph","content":[{"type":"text","text":"x","marks":[{"type":"link","href":"javascript:alert(1)"}]}]}]`,
		`[{"type":"paragraph","content":[{"type":"text","text":"x","marks":[{"type":"link","href":"data:text/html,x"}]}]}]`,
		`[{"type":"paragraph","content":[{"type":"text","text":"x","marks":[{"type":"link","href":"https:/missing-host"}]}]}]`,
		`[{"type":"paragraph","content":[{"type":"text","text":"x","marks":[{"type":"link","href":"https://user:pass@example.com"}]}]}]`,
		`[{"type":"paragraph","content":[{"type":"text","text":"x","marks":[{"type":"link","href":"//example.com/x"}]}]}]`,
	}
	for _, raw := range invalid {
		if _, err := ValidateBlocksJSON(raw, testLimits); err == nil {
			t.Errorf("invalid content was accepted: %s", raw)
		}
	}

	validLinks := []string{"https://example.com/x", "http://example.com", "mailto:reader@example.com", "/posts/test", "#comments"}
	for _, href := range validLinks {
		raw := `[{"type":"paragraph","content":[{"type":"text","text":"x","marks":[{"type":"link","href":"` + href + `"}]}]}]`
		if _, err := ValidateBlocksJSON(raw, testLimits); err != nil {
			t.Errorf("safe link %q rejected: %v", href, err)
		}
	}
}

func TestHTMLFieldsAreNormalizedOut(t *testing.T) {
	raw := `[{"type":"paragraph","html":"<script>alert(1)</script>","content":[{"type":"text","text":"<b>plain text</b>","html":"<img src=x onerror=alert(1)>"}]}]`
	blocks, err := ValidateBlocksJSON(raw, testLimits)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := MarshalBlocksJSON(blocks)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(encoded, "script") || strings.Contains(encoded, "img") || !strings.Contains(encoded, `\u003cb\u003eplain text\u003c/b\u003e`) {
		t.Fatalf("HTML fields were retained or text was lost: %s", encoded)
	}
}

func TestMediaSourceValidation(t *testing.T) {
	for _, kind := range []string{"image", "gif", "video", "audio", "file"} {
		raw := `[{"type":"media","mediaKind":"` + kind + `","src":"/uploads/файл name.bin","spoiler":false}]`
		if _, err := ValidateBlocksJSON(raw, testLimits); err != nil {
			t.Errorf("valid %s media rejected: %v", kind, err)
		}
	}
	invalid := []string{
		"https://example.com/x.jpg", "/uploads/", "/uploads/../secret", "/uploads/%2e%2e/secret",
		"/uploads/a%2fb", "/uploads/a%5cb", "/uploads//x", `\uploads\x`, "/uploads/x?download=1", "/uploads/x#fragment", "//host/uploads/x",
		"/uploads/x%00.jpg", "/uploads/x%2500.jpg", "/uploads/a%252fb", "/uploads/a%255cb", "/uploads/%252e%252e%252fsecret",
	}
	for _, src := range invalid {
		raw, _ := json.Marshal([]map[string]any{{"type": "media", "mediaKind": "image", "src": src}})
		if _, err := ValidateBlocksJSON(string(raw), testLimits); err == nil {
			t.Errorf("unsafe media source accepted: %q", src)
		}
	}
}

func TestAdminWriteLimitsAndMalformedDocuments(t *testing.T) {
	if _, err := ValidateBlocksJSON(`[{"type":"paragraph","text":"😀"}]`, Limits{1, 1, 10}); err == nil {
		t.Fatal("v1-compatible UTF-16 text limit was not enforced")
	}
	if _, err := ValidateBlocksJSON(`[{"type":"media","mediaKind":"image","src":"/uploads/x","caption":"абв"}]`, Limits{1, 10, 2}); err == nil {
		t.Fatal("media text limit was not enforced")
	}
	if _, err := ValidateBlocksJSON(`[{},{}]`, Limits{1, 10, 10}); err == nil {
		t.Fatal("block count limit was not enforced")
	}
	for _, raw := range []string{"", "null", `{}`, `1`, `[`, `[]`} {
		if _, err := ValidateBlocksJSON(raw, testLimits); err == nil {
			t.Errorf("invalid admin document accepted: %q", raw)
		}
	}
	if blocks, err := ParseBlocksJSON(`[]`); err != nil || len(blocks) != 0 {
		t.Fatalf("empty public document failed safely: blocks=%v error=%v", blocks, err)
	}
	for _, raw := range []string{"", "null", `{}`, `1`, `[`} {
		if _, err := ParseBlocksJSON(raw); err == nil {
			t.Errorf("invalid public document accepted: %q", raw)
		}
	}
	if _, err := ValidateBlocksJSON(`[{"type":"paragraph","text":"ok"}]`, Limits{}); err == nil {
		t.Fatal("invalid limits were accepted")
	}
}

func TestMarshalRejectsUnsafeConstructedBlocks(t *testing.T) {
	for _, block := range []Block{
		{Type: Unknown, Raw: json.RawMessage(`{"type":"future"}`)},
		{Type: Media, MediaKind: Image, Src: "javascript:alert(1)"},
		{Type: Paragraph, Content: []InlineNode{{Type: "html", Text: "x"}}},
	} {
		if _, err := MarshalBlocksJSON([]Block{block}); err == nil {
			t.Fatalf("unsafe constructed block was accepted: %#v", block)
		}
	}
}

func TestStableJSONForEveryBlockShape(t *testing.T) {
	blocks := []Block{
		{Type: Paragraph, Text: "Paragraph"},
		{Type: Heading, Level: 2, Text: "Heading"},
		{Type: Quote, Text: "Quote"},
		{Type: Divider},
		{Type: Media, MediaKind: Image, Src: "/uploads/image.jpg"},
		{Type: Media, MediaKind: GIF, Src: "/uploads/a.gif", Spoiler: true, Name: " a.gif ", Alt: " alt ", Caption: " caption "},
		{Type: Media, MediaKind: Video, Src: "/uploads/a.mp4"},
		{Type: Media, MediaKind: Audio, Src: "/uploads/a.mp3"},
		{Type: Media, MediaKind: File, Src: "/uploads/a.zip"},
	}
	encoded, err := MarshalBlocksJSON(blocks)
	if err != nil {
		t.Fatal(err)
	}
	want := `[{"type":"paragraph","text":"Paragraph"},{"type":"heading","level":2,"text":"Heading"},{"type":"quote","text":"Quote"},{"type":"divider"},{"type":"media","mediaKind":"image","src":"/uploads/image.jpg","spoiler":false},{"type":"media","mediaKind":"gif","src":"/uploads/a.gif","spoiler":true,"name":"a.gif","alt":"alt","caption":"caption"},{"type":"media","mediaKind":"video","src":"/uploads/a.mp4","spoiler":false},{"type":"media","mediaKind":"audio","src":"/uploads/a.mp3","spoiler":false},{"type":"media","mediaKind":"file","src":"/uploads/a.zip","spoiler":false}]`
	if encoded != want {
		t.Fatalf("stable JSON = %s\nwant = %s", encoded, want)
	}
	public, err := json.Marshal(append(blocks, Block{Type: Unknown, Raw: json.RawMessage(`{"secret":true}`)}))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(public), "secret") || !strings.HasSuffix(string(public), `{"type":"unknown"}]`) {
		t.Fatalf("public JSON contract is unsafe: %s", public)
	}
}

func blockTypes(blocks []Block) []BlockType {
	types := make([]BlockType, len(blocks))
	for index, block := range blocks {
		types[index] = block.Type
	}
	return types
}
