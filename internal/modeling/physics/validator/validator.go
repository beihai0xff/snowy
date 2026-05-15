package validator

import (
	"errors"
	"fmt"
	"strings"
)

// CodeValidator 校验生成代码包的安全性与结构完整性。
type CodeValidator interface {
	Validate(bundle map[string]string) error
}

type defaultCodeValidator struct{}

// NewDefaultCodeValidator 创建默认代码校验器。
func NewDefaultCodeValidator() CodeValidator {
	return &defaultCodeValidator{}
}

var forbiddenSnippets = []string{
	"fetch(",
	"xmlhttprequest",
	"websocket(",
	"navigator.sendbeacon",
	"localstorage",
	"sessionstorage",
	"indexeddb",
	"document.cookie",
	"<script src=",
	"import(",
}

func (v *defaultCodeValidator) Validate(bundle map[string]string) error {
	if len(bundle) == 0 {
		return errors.New("code bundle is empty")
	}

	indexHTML, ok := bundle["index.html"]
	if !ok || strings.TrimSpace(indexHTML) == "" {
		return errors.New("index.html is required")
	}

	for name, content := range bundle {
		trimmed := strings.TrimSpace(content)
		if trimmed == "" {
			return fmt.Errorf("%s is empty", name)
		}

		lower := strings.ToLower(trimmed)
		for _, snippet := range forbiddenSnippets {
			if strings.Contains(lower, strings.ToLower(snippet)) {
				return fmt.Errorf("%s contains forbidden snippet %q", name, snippet)
			}
		}
	}

	return nil
}
