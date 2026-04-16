package policy

import (
	"context"
	"fmt"
	"strings"
)

var defaultUnsafeKeywords = []string{"爆炸", "weapon", "毒品"}

type defaultEngine struct{}

// NewDefaultEngine 创建默认策略引擎。
func NewDefaultEngine() Engine {
	return &defaultEngine{}
}

func (e *defaultEngine) PreCheck(_ context.Context, input any) (*Result, error) {
	text := strings.TrimSpace(fmt.Sprint(input))
	if text == "" || text == "<nil>" {
		return &Result{Passed: false, Reason: "empty input"}, nil
	}

	for _, keyword := range defaultUnsafeKeywords {
		if strings.Contains(strings.ToLower(text), strings.ToLower(keyword)) {
			return &Result{Passed: false, Reason: "unsafe content"}, nil
		}
	}

	return &Result{Passed: true}, nil
}

func (e *defaultEngine) PostCheck(_ context.Context, output any) (*Result, error) {
	text := strings.TrimSpace(fmt.Sprint(output))
	if text == "" || text == "<nil>" {
		return &Result{Passed: false, Reason: "empty output"}, nil
	}

	return &Result{Passed: true}, nil
}
