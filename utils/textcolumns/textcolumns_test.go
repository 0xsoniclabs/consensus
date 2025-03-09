package textcolumns

import (
	"strings"
	"testing"
)

func TestTextColumns(t *testing.T) {
	tests := []struct {
		name     string
		inputs   []string
		expected string
	}{
		{
			name:     "Empty input",
			inputs:   []string{},
			expected: "\n",
		},
		{
			name:     "Single empty column",
			inputs:   []string{""},
			expected: "\t\n",
		},
		{
			name:     "Multiple empty columns",
			inputs:   []string{"", "", ""},
			expected: "\t\t\t\n",
		},
		{
			name:   "Single column with one line",
			inputs: []string{"Hello"},
			// Padding with spaces (count=width of column) in the absence of content for this column for this line
			expected: "Hello\t\n     \t\n",
		},
		{
			name:     "Single column with multiple lines",
			inputs:   []string{"Hello\nWorld"},
			expected: "Hello\t\nWorld\t\n     \t\n",
		},
		{
			name:     "Multiple columns with same height",
			inputs:   []string{"Hello\nWorld", "Foo\nBar"},
			expected: "Hello\tFoo\t\nWorld\tBar\t\n     \t   \t\n",
		},
		{
			name:     "Multiple columns with different heights",
			inputs:   []string{"Hello\nWorld\nGoodbye", "Foo\nBar", "Test"},
			expected: "Hello  \tFoo\tTest\t\nWorld  \tBar\t    \t\nGoodbye\t   \t    \t\n       \t   \t    \t\n",
		},
		{
			name:     "Different width columns",
			inputs:   []string{"Short", "This is a longer text"},
			expected: "Short\tThis is a longer text\t\n     \t                     \t\n",
		},
		{
			name:     "Mix of empty and non-empty lines",
			inputs:   []string{"First\n\nThird", "One\nTwo"},
			expected: "First\tOne\t\n     \tTwo\t\nThird\t   \t\n     \t   \t\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TextColumns(tt.inputs...)

			// Normalize line endings for comparison on different platforms
			result = strings.ReplaceAll(result, "\r\n", "\n")
			expected := strings.ReplaceAll(tt.expected, "\r\n", "\n")

			if result != expected {
				t.Errorf("TextColumns() = %q, want %q", result, expected)
			}
		})
	}
}

func TestColumnAlignment(t *testing.T) {
	input1 := "A\nLong line here"
	input2 := "B\nC"

	result := TextColumns(input1, input2)
	lines := strings.Split(strings.TrimSuffix(result, "\n"), "\n")

	if len(lines) != 3 {
		t.Fatalf("Expected 3 lines, got %d", len(lines))
	}

	firstLineTabPos := strings.Index(lines[0], "\t")
	secondLineTabPos := strings.Index(lines[1], "\t")

	if firstLineTabPos != secondLineTabPos {
		t.Errorf("Columns not aligned properly: tab positions are %d and %d",
			firstLineTabPos, secondLineTabPos)
	}
}

func TestEdgeCases(t *testing.T) {
	// Test with line containing only whitespace (line 2)
	result := TextColumns("Line 1\n \nLine 3", "A\nB\nC")
	if !strings.Contains(result, " \tB") {
		t.Errorf("Whitespace line not handled correctly")
	}

	// Test with very long input
	longString := strings.Repeat("x", 1000)
	result = TextColumns(longString)
	if len(result) < 1000 {
		t.Errorf("Long string not handled correctly")
	}
}
