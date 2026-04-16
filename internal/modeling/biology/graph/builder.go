// Package graph 定义生物结构图/流程图构建接口。
package graph

import "github.com/beihai0xff/snowy/internal/modeling/biology/domain"

// DiagramBuilder 结构图/流程图构建接口。
type DiagramBuilder interface {
	// Build 根据概念和关系生成可渲染的 DiagramSpec。
	Build(concepts []domain.Concept, relations []domain.Relation, title string) (*domain.DiagramSpec, error)
}
