package agent

import "testing"

func TestHandlerForModeCoversKnownModes(t *testing.T) {
	cases := []struct {
		mode       Mode
		tool       ToolKind
		demoDomain string
	}{
		{mode: ModeSearch, tool: ToolKindSearch},
		{mode: ModePhysics, tool: ToolKindPhysics, demoDomain: "physics"},
		{mode: ModeBiology, tool: ToolKindBiology, demoDomain: "biology"},
		{mode: ModeChemistry, tool: ToolKindChemistry, demoDomain: "chemistry"},
		{mode: ModeAuto, tool: ToolKindSearch},
	}

	for _, tc := range cases {
		handler, ok := HandlerForMode(tc.mode)
		if !ok {
			t.Fatalf("missing handler for mode %s", tc.mode)
		}

		if handler.Tool != tc.tool {
			t.Fatalf("mode %s tool = %s, want %s", tc.mode, handler.Tool, tc.tool)
		}

		if got := DemoDomainForMode(tc.mode); got != tc.demoDomain {
			t.Fatalf("mode %s demo domain = %q, want %q", tc.mode, got, tc.demoDomain)
		}
	}
}
