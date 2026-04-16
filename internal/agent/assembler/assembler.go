// Package assembler 定义结果组装接口。
package assembler

import (
	"context"

	"github.com/beihai0xff/snowy/internal/agent"
)

// Assembler 结果组装接口。
// 负责将工具输出和模型输出组装为最终 ChatResponse。
type Assembler interface {
	// Assemble 组装最终响应。
	Assemble(
		ctx context.Context,
		mode agent.Mode,
		toolOutputs map[string]any,
		modelOutput any,
	) (*agent.ChatResponse, error)
}
