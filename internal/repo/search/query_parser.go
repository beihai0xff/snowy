package search

import (
	"strings"
	"unicode"
)

// NewSimpleParser creates the default lightweight query parser.
func NewSimpleParser() QueryParser {
	return &simpleParser{}
}

type simpleParser struct{}

func (p *simpleParser) Parse(raw string) (*ParsedQuery, error) {
	cleaned := strings.TrimSpace(raw)
	keywords := dedupeTokens(strings.FieldsFunc(cleaned, func(r rune) bool {
		return unicode.IsSpace(r) || strings.ContainsRune(",，。！？!?;；:：()（）[]【】'\"", r)
	}))

	entities := make([]string, 0, len(keywords))
	for _, token := range keywords {
		if len([]rune(token)) >= 2 {
			entities = append(entities, token)
		}
	}

	return &ParsedQuery{
		Original: cleaned,
		Keywords: keywords,
		Entities: entities,
		Intent:   resolveIntent(cleaned),
	}, nil
}

func dedupeTokens(tokens []string) []string {
	seen := make(map[string]struct{}, len(tokens))

	result := make([]string, 0, len(tokens))
	for _, token := range tokens {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}

		key := strings.ToLower(token)
		if _, ok := seen[key]; ok {
			continue
		}

		seen[key] = struct{}{}

		result = append(result, token)
	}

	return result
}

func resolveIntent(text string) string {
	lower := strings.ToLower(text)
	switch {
	case strings.Contains(lower, "为什么") || strings.Contains(lower, "why"):
		return IntentReason
	case strings.Contains(lower, "如何") || strings.Contains(lower, "how"):
		return IntentMethod
	case strings.Contains(lower, "定义") || strings.Contains(lower, "是什么") || strings.Contains(lower, "what"):
		return IntentDefinition
	default:
		return IntentExplain
	}
}
