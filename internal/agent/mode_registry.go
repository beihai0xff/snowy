package agent

// ToolKind identifies the primary tool flow for an agent mode.
type ToolKind string

const (
	ToolKindSearch    ToolKind = "search"
	ToolKindPhysics   ToolKind = "physics"
	ToolKindBiology   ToolKind = "biology"
	ToolKindChemistry ToolKind = "chemistry"
)

// ModeHandler defines the orchestration metadata for one user-facing mode.
type ModeHandler struct {
	Mode       Mode
	Domain     string
	Tool       ToolKind
	DemoDomain string
}

var modeHandlers = map[Mode]ModeHandler{
	ModeSearch:    {Mode: ModeSearch, Domain: "search", Tool: ToolKindSearch},
	ModePhysics:   {Mode: ModePhysics, Domain: "physics", Tool: ToolKindPhysics, DemoDomain: "physics"},
	ModeBiology:   {Mode: ModeBiology, Domain: "biology", Tool: ToolKindBiology, DemoDomain: "biology"},
	ModeChemistry: {Mode: ModeChemistry, Domain: "chemistry", Tool: ToolKindChemistry, DemoDomain: "chemistry"},
	ModeAuto:      {Mode: ModeAuto, Domain: "auto", Tool: ToolKindSearch},
}

// HandlerForMode returns the orchestration metadata for a mode.
func HandlerForMode(mode Mode) (ModeHandler, bool) {
	handler, ok := modeHandlers[mode]

	return handler, ok
}

// DemoDomainForMode returns the default modeling domain for demo generation.
func DemoDomainForMode(mode Mode) string {
	handler, ok := HandlerForMode(mode)
	if !ok {
		return ""
	}

	return handler.DemoDomain
}
