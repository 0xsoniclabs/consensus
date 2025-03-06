// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package utils

import (
	"bufio"
	"strings"
)

// TextColumns join side-by-side.
func TextColumns(texts ...string) string {
	var (
		columns = make([][]string, len(texts))
		widthes = make([]int, len(texts))
	)

	for i, text := range texts {
		scanner := bufio.NewScanner(strings.NewReader(text))
		for scanner.Scan() {
			line := scanner.Text()
			columns[i] = append(columns[i], line)
			if widthes[i] < len([]rune(line)) {
				widthes[i] = len([]rune(line))
			}
		}
	}

	var (
		res strings.Builder
		j   int
	)
	for {
		eof := true
		for i := range columns {
			var s string
			if len(columns[i]) > j {
				s = columns[i][j]
				s = s + strings.Repeat(" ", widthes[i]-len([]rune(s)))
				eof = false
			} else {
				s = strings.Repeat(" ", widthes[i])
			}
			res.WriteString(s)
			res.WriteString("\t")
		}
		res.WriteString("\n")
		j++
		if eof {
			break
		}
	}

	return res.String()
}
