package mk2rbc

// Starlark expression types we use
type starlarkType int

const (
	starlarkTypeUnknown starlarkType = iota
	starlarkTypeList    starlarkType = iota
	starlarkTypeString  starlarkType = iota
	starlarkTypeInt     starlarkType = iota
	starlarkTypeBool    starlarkType = iota
	starlarkTypeVoid    starlarkType = iota
)

type varClass int

const (
	VarClassConfig varClass = iota
	VarClassSoong  varClass = iota
	VarClassLocal  varClass = iota
)

type variableRegistrar interface {
	NewVariable(name string, varClass varClass, valueType starlarkType)
}

// ScopeBase is a dummy implementation of the mkparser.Scope.
// All our scopes are read-only and resolve only simple variables.
type ScopeBase struct{}

func (s ScopeBase) Set(_, _ string) {
	panic("implement me")
}

func (s ScopeBase) Call(_ string, _ []string) []string {
	panic("implement me")
}

func (s ScopeBase) SetFunc(_ string, _ func([]string) []string) {
	panic("implement me")
}
