package transform

import (
	"strings"
	"unicode"
)

// Apply applies a named transformation to the given value
func Apply(value, transformType string) string {
	switch transformType {
	case "upper":
		return strings.ToUpper(value)
	case "lower":
		return strings.ToLower(value)
	case "title":
		return strings.Title(strings.ToLower(value)) //nolint:staticcheck
	case "trim":
		return strings.TrimSpace(value)
	case "ltrim":
		return strings.TrimLeftFunc(value, unicode.IsSpace)
	case "rtrim":
		return strings.TrimRightFunc(value, unicode.IsSpace)
	case "snake":
		return toSnakeCase(value)
	case "camel":
		return toCamelCase(value)
	case "pascal":
		return toPascalCase(value)
	default:
		return value
	}
}

func toSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if unicode.IsUpper(r) && i > 0 {
			result = append(result, '_')
		}
		result = append(result, unicode.ToLower(r))
	}
	// Replace spaces/hyphens with underscore
	str := string(result)
	str = strings.ReplaceAll(str, " ", "_")
	str = strings.ReplaceAll(str, "-", "_")
	return str
}

func toCamelCase(s string) string {
	words := splitWords(s)
	if len(words) == 0 {
		return s
	}
	result := strings.ToLower(words[0])
	for _, w := range words[1:] {
		if len(w) > 0 {
			result += strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
		}
	}
	return result
}

func toPascalCase(s string) string {
	words := splitWords(s)
	var result string
	for _, w := range words {
		if len(w) > 0 {
			result += strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
		}
	}
	return result
}

func splitWords(s string) []string {
	// Split on spaces, underscores, hyphens, and camelCase transitions
	var words []string
	var word strings.Builder
	for i, r := range s {
		if r == ' ' || r == '_' || r == '-' {
			if word.Len() > 0 {
				words = append(words, word.String())
				word.Reset()
			}
		} else if unicode.IsUpper(r) && i > 0 && word.Len() > 0 {
			words = append(words, word.String())
			word.Reset()
			word.WriteRune(r)
		} else {
			word.WriteRune(r)
		}
	}
	if word.Len() > 0 {
		words = append(words, word.String())
	}
	return words
}
