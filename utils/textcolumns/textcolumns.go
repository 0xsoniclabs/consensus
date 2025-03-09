// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package textcolumns

import (
	"bufio"
	"strings"
)

// TextColumns joins multiple text blocks side-by-side, aligning them into columns.
// Example Usage:
//
//	names := "Alice\nBob\nCharlie"
//	numbers := "123-456-7890\n987-654-3210\n555-1212"
//	result := TextColumns(names, numbers)
//	// result will be:
//	// Alice   123-456-7890
//	// Bob     987-654-3210
//	// Charlie 555-1212
func TextColumns(texts ...string) string {
	var (
		columns = make([][]string, len(texts)) // columns to store each text
		widths  = make([]int, len(texts))      // widths to store the max widths for each text (column)
	)

	// First, we iterate over each input text.
	for i, text := range texts {
		scanner := bufio.NewScanner(strings.NewReader(text)) // scanner to read text line by line
		for scanner.Scan() {
			line := scanner.Text()                // get current line of text
			columns[i] = append(columns[i], line) // append current line to corresponding text
			// Keep track of the maximum width encountered in each text. This is crucial for alignment.
			if widths[i] < len([]rune(line)) {
				widths[i] = len([]rune(line))
			}
		}
	}

	// Now, we construct the output string.
	var (
		res strings.Builder // Use a strings.Builder for efficient string concatenation.
		j   int             // Denotes a row index j.
	)
	for {
		eof := true // Assume end-of-file until we do not find a line in at least one column.
		// Iterate through each column.
		for i := range columns {
			var s string // string to be written for the current cell
			// Check if the current column has a line at index j
			if len(columns[i]) > j {
				s = columns[i][j]                                     // If yes, retrieve the line.
				s = s + strings.Repeat(" ", widths[i]-len([]rune(s))) // calculate padding to make all lines same width
				eof = false                                           // We found a line, so it's not the end of the file (yet).
			} else {
				s = strings.Repeat(" ", widths[i]) // Padding with spaces (count=width of column) in the absence of content for this column for this line
			}
			res.WriteString(s)    // Write to string builder
			res.WriteString("\t") // Writes a tab to string builder, to split column text by.
		}
		res.WriteString("\n") // Add a newline after each row.
		j++
		if eof {
			break // If we didn't find any lines in any of the columns in this iteration, we're done.
		}
	}

	return res.String()
}
