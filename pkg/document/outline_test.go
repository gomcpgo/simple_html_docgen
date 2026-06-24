package document

import "testing"

func TestParseHeadings_OrderAndLevels(t *testing.T) {
	html := `<html><body>
		<h1>Title</h1>
		<p>intro</p>
		<h2>Section A</h2>
		<h3>Sub A.1</h3>
		<h2>Section B</h2>
	</body></html>`

	got := ParseHeadings(html)
	want := []Heading{
		{Level: 1, Text: "Title"},
		{Level: 2, Text: "Section A"},
		{Level: 3, Text: "Sub A.1"},
		{Level: 2, Text: "Section B"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d headings, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("heading %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestParseHeadings_StripsNestedTagsAndAttributes(t *testing.T) {
	html := `<h1 id="x" class="y">Hello <span>World</span></h1>`
	got := ParseHeadings(html)
	if len(got) != 1 {
		t.Fatalf("got %d headings, want 1", len(got))
	}
	if got[0].Level != 1 || got[0].Text != "Hello World" {
		t.Errorf("got %+v, want {1 Hello World}", got[0])
	}
}

func TestParseHeadings_None(t *testing.T) {
	if got := ParseHeadings("<p>no headings here</p>"); len(got) != 0 {
		t.Errorf("expected no headings, got %+v", got)
	}
}
