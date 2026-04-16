package query

import "github.com/beihai0xff/snowy/internal/repo/search"

// Parser 查询理解接口 — 将原始文本转为结构化查询。
type Parser interface {
	Parse(raw string) (*search.ParsedQuery, error)
}
