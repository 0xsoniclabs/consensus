// Copyright (c) 2025 Fantom Foundation
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at fantom.foundation/bsl11.
//
// Change Date: 2028-4-16
//
// On the date above, in accordance with the Business Source License, use of
// this software will be governed by the GNU Lesser General Public License v3.

package fmtfilter

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

// parseScanfOps extracts the format specifiers (%d, %s) from a template string.
// For example, parseScanfOps("Hello %d World %s") returns "%d%s", nil.
func parseScanfOps(template string) (string, error) {
	readOp := false // Flag to indicate if we're currently expecting a format specifier after a '%'
	ops := ""       // String to accumulate the format specifiers
	for i, ch := range template {
		if readOp { // If the previous character was a '%'
			if ch == 'd' {
				ops += "%d"
			} else if ch == 's' {
				ops += "%s"
			} else if ch == '%' {
				// Handle escaped percent signs (%%) - do nothing, just consume the second '%'
			} else if !unicode.IsLetter(ch) {
				// Allow non-letter characters immediately after the % (e.g. %10d)
				continue
			} else {
				// Invalid format specifier
				return "", fmt.Errorf("unexpected op in position %d for template '%s'", i, template)
			}
		}
		readOp = !readOp && ch == '%' // Toggle readOp when we encounter a '%'
	}

	// Handle the case where a '%' is at the end of the template without a following specifier
	if readOp {
		return "", fmt.Errorf("non-closed %% in template '%s'", template)
	}

	return ops, nil
}

// CompileFilter creates a filter function that transforms strings based on scanf and printf templates.
// It returns a function that takes a string as input and returns a transformed string and an error.
func CompileFilter(scanfTemplate, printfTemplate string) (func(req string) (string, error), error) {
	// Extract the format specifiers from both templates
	ops, err := parseScanfOps(scanfTemplate)
	if err != nil {
		return nil, err
	}
	printfOps, err := parseScanfOps(printfTemplate)
	if err != nil {
		return nil, err
	}

	// Validate that the printf template's operators are a prefix of the scanf template's operators.
	// This ensures that the printf template has access to all the values it needs.
	if !strings.HasPrefix(ops, printfOps) {
		return nil, fmt.Errorf("template ops for scanf don't match scanf ops: '%s' != '%s'", ops, printfOps)
	}

	// Create and return the appropriate filter function based on the extracted format specifiers.
	// This is where the function behaves differently based on how many and which types of template ops are present
	if ops == "" {
		// Case: No format specifiers in either template.  Simple string comparison and replacement.
		return func(req string) (string, error) {
			if req == scanfTemplate {
				return printfTemplate, nil
			}
			return "", errors.New("doesn't match template")
		}, nil
	} else if ops == "%d" {
		// Case: Single integer format specifier.
		return func(req string) (string, error) {
			var v1 int64 // Use int64 to handle larger numbers
			// Use fmt.Sscanf to parse the input according to the scanfTemplate.
			// It reads the input string req, tries to match it against scanfTemplate,
			// and stores the extracted value in the v1 variable (if successful).
			if _, err := fmt.Sscanf(req, scanfTemplate, &v1); err != nil {
				return "", err
			}
			// If parsing succeeds, use fmt.Sprintf to format the output string according to
			// the printfTemplate and the extracted value v1.
			return fmt.Sprintf(printfTemplate, v1), nil
		}, nil
	} else if ops == "%s" {
		// Case: Single string format specifier.
		return func(req string) (string, error) {
			var v1 string
			if _, err := fmt.Sscanf(req, scanfTemplate, &v1); err != nil {
				return "", err
			}
			return fmt.Sprintf(printfTemplate, v1), nil
		}, nil
	} else if ops == "%d%d" {
		// Case: Two integer format specifiers.
		return func(req string) (string, error) {
			var v1 int64
			var v2 int64
			if _, err := fmt.Sscanf(req, scanfTemplate, &v1, &v2); err != nil {
				return "", err
			}
			return fmt.Sprintf(printfTemplate, v1, v2), nil
		}, nil
	} else if ops == "%d%s" {
		// Case: Integer followed by string format specifier.
		return func(req string) (string, error) {
			var v1 int64
			var v2 string
			if _, err := fmt.Sscanf(req, scanfTemplate, &v1, &v2); err != nil {
				return "", err
			}
			return fmt.Sprintf(printfTemplate, v1, v2), nil
		}, nil
	} else if ops == "%s%d" {
		// Case: String followed by integer format specifier.
		return func(req string) (string, error) {
			var v1 string
			var v2 int64
			if _, err := fmt.Sscanf(req, scanfTemplate, &v1, &v2); err != nil {
				return "", err
			}
			return fmt.Sprintf(printfTemplate, v1, v2), nil
		}, nil
	} else if ops == "%s%s" {
		// Case: Two string format specifiers.
		return func(req string) (string, error) {
			var v1 string
			var v2 string
			if _, err := fmt.Sscanf(req, scanfTemplate, &v1, &v2); err != nil {
				return "", err
			}
			return fmt.Sprintf(printfTemplate, v1, v2), nil
		}, nil
	} else {
		// Case: Unsupported combination of format specifiers.
		return nil, errors.New("unknown combination of scanf operations")
	}
}
