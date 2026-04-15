package user

import (
	"context"
	"fmt"
	"strings"
)

// VerificationCodeChecker 抽象验证码校验端口。
type VerificationCodeChecker interface {
	Verify(ctx context.Context, phone, code string) error
}

type noopVerificationCodeChecker struct{}

// NewNoopVerificationCodeChecker 返回默认的轻量验证码校验器。
func NewNoopVerificationCodeChecker() VerificationCodeChecker {
	return noopVerificationCodeChecker{}
}

func (noopVerificationCodeChecker) Verify(_ context.Context, phone, code string) error {
	if strings.TrimSpace(phone) == "" {
		return fmt.Errorf("phone is empty")
	}
	if strings.TrimSpace(code) == "" {
		return fmt.Errorf("verification code is empty")
	}
	return nil
}
