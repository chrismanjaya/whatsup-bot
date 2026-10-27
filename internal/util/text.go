package util

import (
	"strings"
	"unicode"
)

// ContainsAnyWord reports whether text contains any keyword as a whole word,
// case-insensitively. Words are split on anything that isn't a letter or digit,
// so "*EXPENSE*" matches "expense" but "listrik" does not match "list".
func ContainsAnyWord(text string, keywords []string) bool {
	words := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})

	for _, word := range words {
		for _, keyword := range keywords {
			if word == strings.ToLower(keyword) {
				return true
			}
		}
	}
	return false
}
