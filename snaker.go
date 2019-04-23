package main

import (
	"strings"
	"unicode"
)

// CamelToSnake converts a given string to snake case
func CamelToSnake(s string) string {
	result := ""
	words := make([]string, 0)
	lastPos := 0
	rs := []rune(s)

	for i := 0; i < len(rs); i++ {
		if i > 0 && unicode.IsUpper(rs[i]) {
			words = append(words, s[lastPos:i])
			lastPos = i
		}
	}

	// append the last word
	if s[lastPos:] != "" {
		words = append(words, s[lastPos:])
	}

	for k, word := range words {
		if k > 0 {
			result += "_"
		}

		result += strings.ToLower(word)
	}

	return result
}
