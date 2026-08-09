package lyrics

import "testing"

func TestParse(t *testing.T) {
	content := "[ar:Some Artist]\n" +
		"[00:12.34]First line\n" +
		"[00:15.80]Second line\n" +
		"\n" +
		"[01:05.00]Third line"

	lines, err := Parse(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []Line{
		{TimestampMs: 12340, Text: "First line"},
		{TimestampMs: 15800, Text: "Second line"},
		{TimestampMs: 65000, Text: "Third line"},
	}

	if len(lines) != len(want) {
		t.Fatalf("got %d lines, want %d: %+v", len(lines), len(want), lines)
	}
	for i, l := range lines {
		if l != want[i] {
			t.Errorf("line %d: got %+v, want %+v", i, l, want[i])
		}
	}
}

func TestParseMalformedLine(t *testing.T) {
	content := "[00:12.34]Good line\n" +
		"[no closing bracket\n" +
		"[00:20.00]Another good line"

	lines, err := Parse(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2: %+v", len(lines), lines)
	}
}

func TestParseEmptyContent(t *testing.T) {
	lines, err := Parse("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 0 {
		t.Fatalf("got %d lines, want 0", len(lines))
	}
}
