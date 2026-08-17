package utils

import (
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"strings"
)

// replaceChars replaces characters in a string
// Parameters:
//   - s: the string to replace characters in
//   - o: the characters to replace
//   - r: the replacement character
//
// Example:
//
//	replaceChars("a-b-c", []string{"-"}, "_") => "a_b_c"
//
// Returns:
//   - the string with characters replaced
func replaceChars(s string, o []string, r string) string {
	for _, c := range o {
		s = strings.ReplaceAll(s, c, r)
	}
	return s
}

func ToSnakeCase(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	return replaceChars(s, []string{" ", "-", "."}, "_")
}

func ToPascalCase(s string) string {
	s = Capitalize(s)
	s = strings.ReplaceAll(s, " ", "")
	return s
}

func ToKebabCase(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	return replaceChars(s, []string{" ", "_", "."}, "-")
}

func Capitalize(s string) string {
	s = strings.TrimSpace(s)
	for _, char := range []string{" ", "_", "-", "/"} {
		s = strings.ReplaceAll(s, char, " ")
	}
	s = cases.Title(language.English).String(s)
	return s
}

// splitStringWithQuotes splits a string with quotes
// Parameters:
//   - s: the string to split
//   - q: the quote character
//   - d: the delimiter
//   - r: the replacement
//
// Example:
//
//	splitStringWithQuotes("a,b,c", "'", ",", ", ") => "'a', 'b', 'c'"
//
// Returns:
//   - the split string
func splitStringWithQuotes(s, q, d, r string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, d, q+r+q)
	s = q + s + q
	return s
}

func SplitStringWithSingleQuotes(s string) string {
	return splitStringWithQuotes(s, "'", ",", ", ")
}

func SplitStringWithDoubleQuotes(s string) string {
	return splitStringWithQuotes(s, "\"", ",", "\", \"")
}
