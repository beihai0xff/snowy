package node

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/beihai0xff/snowy/internal/agent"
)

func TestRegenerateClassifier_NoParent_Defaults_New(t *testing.T) {
	node := NewRegenerateClassifierNode(nil)
	state := &State{Request: &agent.ChatRequest{Message: "随便说点什么"}}

	out, err := node.Run(context.Background(), state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := out.(*State)
	if got.RegenerateAction != RegenerateActionNew {
		t.Fatalf("want new, got %s", got.RegenerateAction)
	}
}

func TestRegenerateClassifier_Keyword_Regenerate(t *testing.T) {
	parent := uuid.New()
	node := NewRegenerateClassifierNode(nil)
	state := &State{Request: &agent.ChatRequest{
		Message:         "换一个天体公转的例子",
		ParentPackageID: &parent,
	}}

	out, _ := node.Run(context.Background(), state)
	got := out.(*State)

	if got.RegenerateAction != RegenerateActionRegenerate {
		t.Fatalf("want regenerate, got %s", got.RegenerateAction)
	}
}

func TestRegenerateClassifier_Keyword_Recompute(t *testing.T) {
	parent := uuid.New()
	node := NewRegenerateClassifierNode(nil)
	state := &State{Request: &agent.ChatRequest{
		Message:         "如果 v0 改成 30 m/s 呢",
		ParentPackageID: &parent,
	}}

	out, _ := node.Run(context.Background(), state)
	got := out.(*State)

	if got.RegenerateAction != RegenerateActionRecompute {
		t.Fatalf("want recompute, got %s", got.RegenerateAction)
	}
}

func TestRegenerateClassifier_DigitTriggersRecompute(t *testing.T) {
	parent := uuid.New()
	node := NewRegenerateClassifierNode(nil)
	state := &State{Request: &agent.ChatRequest{
		Message:         "30 度试试",
		ParentPackageID: &parent,
	}}

	out, _ := node.Run(context.Background(), state)
	got := out.(*State)

	if got.RegenerateAction != RegenerateActionRecompute {
		t.Fatalf("want recompute, got %s", got.RegenerateAction)
	}
}
