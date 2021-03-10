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
