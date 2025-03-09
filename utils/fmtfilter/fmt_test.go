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
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseScanfOps_NormalFormatSpecifiers(t *testing.T) {
	ops, err := parseScanfOps("Hello %d World %s")
	require.NoError(t, err)
	require.Equal(t, "%d%s", ops)
}

func TestParseScanfOps_EscapedPercentSigns(t *testing.T) {
	ops, err := parseScanfOps("100%% sure that %d is a number")
	require.NoError(t, err)
	require.Equal(t, "%d", ops)
}

func TestParseScanfOps_NonLetterCharactersBetweenPercentAndSpecifier(t *testing.T) {
	ops, err := parseScanfOps("Text with %123d and %456s")
	require.NoError(t, err)
	require.Equal(t, "%d%s", ops)
}

func TestParseScanfOps_InvalidFormatSpecifier(t *testing.T) {
	_, err := parseScanfOps("Invalid %f format")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unexpected op")
}

func TestParseScanfOps_NonClosedPercentAtEnd(t *testing.T) {
	_, err := parseScanfOps("Incomplete format %")
	require.Error(t, err)
	require.Contains(t, err.Error(), "non-closed %")
}

func TestParseScanfOps_EmptyString(t *testing.T) {
	ops, err := parseScanfOps("")
	require.NoError(t, err)
	require.Equal(t, "", ops)
}

func TestCompileFilter_TemplateOpsMismatch1(t *testing.T) {
	_, err := CompileFilter("%d", "%s")
	require.Error(t, err)
	require.Contains(t, err.Error(), "template ops for scanf don't match scanf ops")
}

func TestCompileFilter_TemplateOpsMismatch2(t *testing.T) {
	_, err := CompileFilter("%d%s", "%s")
	require.Error(t, err)
	require.Contains(t, err.Error(), "template ops for scanf don't match scanf ops")
}

func TestCompileFilter_TemplateOpsMismatch3(t *testing.T) {
	_, err := CompileFilter("dd%d%sdd", "dd%sss")
	require.Error(t, err)
	require.Contains(t, err.Error(), "template ops for scanf don't match scanf ops")
}

func TestCompileFilter_TemplatesNoFormatSpecifiersDifferentStrings(t *testing.T) {
	_, err := CompileFilter("dd%d%sdd", "%%")
	require.NoError(t, err)
}

func TestCompileFilter_ErrorFromParseScanfOps_NonClosedPercent(t *testing.T) {
	_, err := CompileFilter("%", "%s")
	require.Error(t, err)
	require.Contains(t, err.Error(), "non-closed %")
}

func TestCompileFilter_ErrorFromParseScanfOps_NonClosedPercent2(t *testing.T) {
	_, err := CompileFilter("%s", "%5")
	require.Error(t, err)
	require.Contains(t, err.Error(), "non-closed %")
}

func TestCompileFilter_ErrorFromParseScanfOps_UnexpectedOp(t *testing.T) {
	_, err := CompileFilter("%f", "%s")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unexpected op")
}

func TestCompileFilter_EmptyOpsCase(t *testing.T) {
	fn, err := CompileFilter("hello", "world")
	require.NoError(t, err)
	res, err := fn("hello")
	require.NoError(t, err)
	require.Equal(t, "world", res)
	_, err = fn("not hello")
	require.Error(t, err)
}

func TestCompileFilter_PercentDCase(t *testing.T) {
	fn, err := CompileFilter("qw%der", "ty%dui")
	require.NoError(t, err)
	_, err = fn("123")
	require.Error(t, err)
	_, err = fn("ty123ui")
	require.Error(t, err)
	_, err = fn("qw123")
	require.Error(t, err)
	_, err = fn("qw123e")
	require.Error(t, err)
	res, err := fn("qw123er")
	require.NoError(t, err)
	require.Equal(t, "ty123ui", res)
}

func TestCompileFilter_PercentSCase(t *testing.T) {
	fn, err := CompileFilter("text %s", "output %s")
	require.NoError(t, err)
	res, err := fn("text hello")
	require.NoError(t, err)
	require.Equal(t, "output hello", res)
	_, err = fn("invalid input")
	require.Error(t, err)
}

func TestCompileFilter_PercentDPercentDCase(t *testing.T) {
	fn, err := CompileFilter("nums %d and %d", "values %d,%d")
	require.NoError(t, err)
	res, err := fn("nums 123 and 456")
	require.NoError(t, err)
	require.Equal(t, "values 123,456", res)
	_, err = fn("invalid input")
	require.Error(t, err)
}

func TestCompileFilter_PercentDPercentSCase_EdgeCases(t *testing.T) {
	fn, err := CompileFilter("qw%d%2s123%%", "--%d__%s~~%%")
	require.NoError(t, err)
	_, err = fn("qw456AB123")
	require.Error(t, err)
	res, err := fn("qw456AB123%")
	require.NoError(t, err)
	require.Equal(t, "--456__AB~~%", res)
}

func TestCompileFilter_PercentDPercentSCase_EdgeCases2(t *testing.T) {
	fn, err := CompileFilter("qw%d%2s123", "--%d__%s~~")
	require.NoError(t, err)
	_, err = fn("qw456ABC123")
	require.Error(t, err)
	res, err := fn("qw456AB123")
	require.NoError(t, err)
	require.Equal(t, "--456__AB~~", res)
}

func TestCompileFilter_PercentSPercentDCase(t *testing.T) {
	fn, err := CompileFilter("text %s number %d", "result: %s-%d")
	require.NoError(t, err)
	res, err := fn("text hello number 42")
	require.NoError(t, err)
	require.Equal(t, "result: hello-42", res)
	_, err = fn("invalid input")
	require.Error(t, err)
}

func TestCompileFilter_PercentSPercentSCase(t *testing.T) {
	fn, err := CompileFilter("words %s and %s", "%s+%s")
	require.NoError(t, err)
	res, err := fn("words foo and bar")
	require.NoError(t, err)
	require.Equal(t, "foo+bar", res)
	_, err = fn("invalid input")
	require.Error(t, err)
}

func TestCompileFilter_UnsupportedCombinations(t *testing.T) {
	_, err := CompileFilter("%d%d%d", "%d%d%d")
	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown combination of scanf operations")
}
