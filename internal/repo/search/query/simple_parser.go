package query

import (
	"strings"
	"unicode"

	internalsearch "github.com/beihai0xff/snowy/internal/repo/search"
)

// simpleParser 提供轻量查询理解实现。
type simpleParser struct{}

// NewSimpleParser 创建默认查询解析器。
func NewSimpleParser() Parser {
	return &simpleParser{}
}

func (p *simpleParser) Parse(raw string) (*internalsearch.ParsedQuery, error) {
	cleaned := strings.TrimSpace(raw)
	keywords := dedupeTokens(strings.FieldsFunc(cleaned, func(r rune) bool {
		return unicode.IsSpace(r) || strings.ContainsRune(",，。！？!?;；:：()（）[]【】'\"", r)
	}))

	intent := "explain"
	lower := strings.ToLower(cleaned)
	switch {
	case strings.Contains(lower, "为什么") || strings.Contains(lower, "why"):
		intent = "reason"
	case strings.Contains(lower, "如何") || strings.Contains(lower, "how"):
		intent = "method"
	case strings.Contains(lower, "定义") || strings.Contains(lower, "是什么") || strings.Contains(lower, "what"):
		intent = "definition"
	}

	entities := make([]string, 0, len(keywords))
	for _, token := range keywords {
		if len([]rune(token)) >= 2 {
			entities = append(entities, token)
		}
	}

	return &internalsearch.ParsedQuery{
		Original: cleaned,
		Keywords: keywords,
		Entities: entities,
		Intent:   intent,
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
