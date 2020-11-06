package mk2star

import (
	mkparser "android/soong/androidmk/parser"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/scanner"
)

const (
	baseUri                  = "//build/make/target/product:product_config"
	baseName                 = "rblf"
	configFunctionName       = baseName + ".prodconf"
	printvarsFunctionName    = baseName + ".printvars"
	warningFunctionName      = baseName + ".warning"
	ifdefVarFunctionName     = baseName + ".is_defined"
	soongVarFunctionName     = baseName + ".soong_var"
	fileExistsFunctionName   = baseName + ".file_exists"
	fileWildcardFunctionName = baseName + ".file_wildcard_exists"
)

type ErrorMonitorCB interface {
	NewError(s string, node mkparser.Node, args ...interface{})
}

type emptyErrorMonitor struct{}

func (_ emptyErrorMonitor) NewError(_ string, _ mkparser.Node, _ ...interface{}) {}

var StarlarkSuffix = ".star"
var ErrorMonitor ErrorMonitorCB = emptyErrorMonitor{}
var listSeparatorRex = regexp.MustCompile(", *")

// Derives module name for a given file. It is base name
// (file name without suffix), with some characters replaced to make it a Starlark identifier
func moduleNameForFile(mkFile string) string {
	base := strings.TrimSuffix(filepath.Base(mkFile), filepath.Ext(mkFile))
	// TODO(asmundak): what else can be in the product file names?
	return strings.ReplaceAll(base, "-", "_")
}

// If MakeString is a single variable reference, returns it
func singleVar(v *mkparser.MakeString) (*mkparser.MakeString, bool) {
	if len(v.Strings) == 2 && strings.TrimSpace(v.Strings[0]) == "" && strings.TrimSpace(v.Strings[1]) == "" {
		return v.Variables[0].Name, true
	}
	return nil, false

}

// Extracts variable name and returns true if given MakeString is a simple variable reference
func simpleVariableRef(v *mkparser.MakeString) (string, bool) {
	vRef, ok := singleVar(v)
	if ok && len(vRef.Strings) == 1 && !strings.Contains(vRef.Strings[0], " ") {
		return vRef.Strings[0], true
	}
	return "", false
}

func list2MakeStrings(argList string) []*mkparser.MakeString {
	var res []*mkparser.MakeString
	for _, arg := range listSeparatorRex.Split(strings.TrimSpace(argList), -1) {
		res = append(res, mkparser.SimpleMakeString(arg, 0))
	}
	return res
}

func simpleFunctionCall(v *mkparser.MakeString) (string, []*mkparser.MakeString, bool) {
	vRef, ok := singleVar(v)
	if !ok {
		return "", nil, false
	}

	var args []*mkparser.MakeString
	// We are going to handle the following vRef layouts:
	if len(vRef.Strings) > 2 {
		return "", nil, false
	}
	fields := strings.SplitN(vRef.Strings[0], " ", 2)
	if len(fields) > 1 {
		argString := strings.TrimSpace(strings.TrimPrefix(vRef.Strings[0], fields[0]))
		if argString != "" {
			args = list2MakeStrings(strings.TrimRight(argString, " ,"))
		}
	}
	if len(vRef.Strings) > 1 {
		args = append(args, &mkparser.MakeString{Strings: []string{"", ""}, Variables: vRef.Variables})
		args = append(args, list2MakeStrings(strings.TrimLeft(vRef.Strings[1], ", "))...)
	}
	return fields[0], args, true
}

// Starlark output generation context
type generationContext struct {
	buf               strings.Builder
	starScript        *StarScript
	initDone          bool
	varLastAssignment map[string]*assignmentNode
	indentLevel       int
}

func NewGenerateContext(ss *StarScript) *generationContext {
	return &generationContext{
		starScript:        ss,
		varLastAssignment: make(map[string]*assignmentNode),
	}
}

// emit returns generated script
func (gctx *generationContext) emit() string {
	ss := gctx.starScript
	for asgn := ss.varAssignmentFirst; asgn != nil; asgn = asgn.next {
		if _, ok := gctx.varLastAssignment[asgn.name]; !ok {
			gctx.varLastAssignment[asgn.name] = asgn
		}
	}
	initNeeded := true
	for _, node := range ss.nodes {
		// emit initialization after the initial comment block
		if initNeeded {
			if _, ok := node.(*commentNode); !ok {
				gctx.emitInit()
				initNeeded = false
			}
		}
		node.emit(gctx)
	}
	gctx.emitInit()
	gctx.emitModuleDeclaration()
	if ss.hasErrors {
		gctx.writef("\n%s(%q)\n", warningFunctionName, ss.mkFile)
	}
	if ss.emitPrint {
		gctx.writeln(fmt.Sprintf("%s(%s)", printvarsFunctionName, ss.moduleName))
	}
	return gctx.buf.String()
}

func (gctx *generationContext) emitInit() {
	if gctx.initDone {
		return
	}
	gctx.writef("load(%q, %q)\n", baseUri+StarlarkSuffix, baseName)
	gctx.writeln("_vars = dict()")
	gctx.initDone = true
}

func (gctx *generationContext) emitConfigVariableArguments() {
	gctx.writeln("    **_vars,")
	return
}

func (gctx *generationContext) emitSubConfigsArgument() {
	gctx.write("    [")
	sep := ""
	for sc := gctx.starScript.subConfigFirst; sc != nil; sc = sc.next {
		gctx.write(sep, sc.moduleLocalName)
		sep = ", "
	}
	gctx.writeln("],")
}

func (gctx *generationContext) emitModuleDeclaration() {
	gctx.writef("%s = %s(\n", gctx.starScript.moduleName, configFunctionName)
	gctx.writeln(`    "`, gctx.starScript.moduleName, `",`)
	gctx.emitSubConfigsArgument()
	gctx.emitConfigVariableArguments()
	gctx.writeln(")")
}

func (gctx *generationContext) emitPass() {
	gctx.writeIndent()
	gctx.writeln("pass")
}

func (gctx *generationContext) write(ss ...string) {
	for _, s := range ss {
		gctx.buf.WriteString(s)
	}
}

func (gctx *generationContext) writef(format string, args ...interface{}) {
	gctx.write(fmt.Sprintf(format, args...))
}

func (gctx *generationContext) writeIndent() {
	gctx.write("                                                            "[0 : 2*gctx.indentLevel])
}

func (gctx *generationContext) writeln(ss ...string) {
	gctx.write(ss...)
	gctx.write("\n")
}

type VariableRegistrar interface {
	NewVariable(name, flavor string)
}

type knownVariables struct {
	vars map[string]struct {
		class  string
		flavor string
	}
}

func (pcv *knownVariables) NewVariable(name, class, flavor string) {
	var v struct{ class, flavor string }
	var ok bool
	if v, ok = pcv.vars[name]; ok {
		// Conflict resolution:
		//    * If flavor does not change, first one wins
		//    * Any type trumps unknown type
		//    * List should match list
		//    * specific type ('str', 'bool', etc.) trumps generic 'item'
		newFlavor := v.flavor
		mismatch := false
		switch flavor {
		case v.flavor:
			return
		case "item":
			if v.flavor == "list" {
				mismatch = true
			} else {
				// Specific item type trumps generic
				newFlavor = flavor
			}
		case "list":
			if v.flavor == "" {
				newFlavor = flavor
			} else {
				mismatch = flavor != "list"
			}
		case "":

		default:
			newFlavor = flavor
		}
		if mismatch {
			fmt.Fprintf(os.Stderr, "cannot redefine %s as %s/%s (already defined as %s/%s)\n",
				name, class, flavor, v.class, v.flavor)
			return
		} else if v.flavor == newFlavor {
			return
		}
		v.flavor = newFlavor
	} else {
		v = struct{ class, flavor string }{class, flavor}
	}
	pcv.vars[name] = struct{ class, flavor string }{class, flavor}
}

// All known product variables.
var KnownVariables = knownVariables{make(map[string]struct{ class, flavor string })}

type starlarkNode interface {
	emit(ctx *generationContext)
}

type nodeReceiver interface {
	newNode(node starlarkNode)
}

// Types used to keep processed makefile data:
type commentNode struct {
	text string
}

func (c *commentNode) emit(gctx *generationContext) {
	gctx.writeIndent()
	gctx.writeln(c.text)
}

type literalNode struct {
	text string
}

func (l *literalNode) emit(gctx *generationContext) {
	gctx.write(l.text)
}

type subConfigNode struct {
	path            string
	originalPath    string
	moduleName      string
	moduleLocalName string
	next            *subConfigNode
	loadAlways      bool
}

func (sc *subConfigNode) emit(gctx *generationContext) {
	ss := gctx.starScript
	if sc.loadAlways {
		gctx.writef(`load("%s", `, ss.path2BazelRef(sc.path))
	} else {
		gctx.writef(`load("%s|%s", `, ss.path2BazelRef(sc.path), sc.moduleName)
	}
	if sc.moduleLocalName != sc.moduleName {
		gctx.write(sc.moduleLocalName, ` = `)
	}
	gctx.writeln(`"`, sc.moduleName, `")`)
}

type assignmentNode struct {
	name   string
	value  *mkparser.MakeString
	flavor string
	next   *assignmentNode
}

func (asgn *assignmentNode) emit(gctx *generationContext) {
	desc, ok := KnownVariables.vars[asgn.name]
	if !ok {
		panic(fmt.Errorf("unknown variable %s", asgn.name))
	}
	// TODO: handle ?= assignments
	value := strings.TrimSpace(asgn.value.Dump())
	notFirst := asgn != gctx.varLastAssignment[asgn.name]
	gctx.writeIndent()
	switch desc.class {
	case "product":
		gctx.write(asgn.referenceName())
		switch desc.flavor {
		case "str", "item":
			if asgn.flavor == "+=" && notFirst {
				gctx.writef(" += \" %s\"\n", value)
			} else {
				gctx.writef(" = \"%s\"\n", value)
			}
		case "list":
			if asgn.flavor == "+=" && notFirst {
				gctx.write(` += `)
			} else {
				gctx.write(` = `)
			}
			items := strings.Fields(value)
			if len(items) == 1 {
				gctx.writef("[%q]\n", items[0])
			} else {
				gctx.writeln("[")
				for _, item := range items {
					gctx.writeIndent()
					gctx.writef("    %q,\n", item)
				}
				gctx.writeIndent()
				gctx.writeln("]")
			}
		default:
			panic(fmt.Errorf("variable %s has unexpected type %s", asgn.name, desc.flavor))
		}
	case "soong":
		gctx.writef("%s = None  ## TODO(soong) %q\n", asgn.referenceName(), value)
		gctx.starScript.hasErrors = true
	default:
		panic(fmt.Errorf("variable %s has unexpected class %s", asgn.name, desc.class))
	}
}

func (asgn assignmentNode) referenceName() string {
	return fmt.Sprintf("_vars[%q]", asgn.name)
}

// Various conditions gating the execution of a switchCase nodes
type gateBad struct {
	which     string // if/elif
	checkText string
}

func (cc *gateBad) emit(gctx *generationContext) {
	// TODO(asmundak): flagging error this late smells.
	gctx.writeIndent()
	gctx.writef("# MK2STAR ERROR: cannot check the following condition:\n")
	gctx.writeIndent()
	gctx.writef("# %s\n", cc.checkText)
	gctx.writeIndent()
	gctx.writef("%s True:\n", cc.which)
	gctx.starScript.hasErrors = true
}

type gateElse struct{}

func (br *gateElse) emit(gctx *generationContext) {
	gctx.writeIndent()
	gctx.writeln("else:")
}

type gateIfdef struct {
	variableName string
	negate       bool
	first        bool
}

func (br *gateIfdef) emit(gctx *generationContext) {
	gctx.writeIndent()
	if br.first {
		gctx.write("if ")
	} else {
		gctx.write("elif ")
	}
	if br.negate {
		gctx.write("not ")
	}
	gctx.writef("%s(%q):\n", ifdefVarFunctionName, br.variableName)
}

type gateSoongVarEqValue struct {
	name   string
	value  string
	negate bool
	first  bool
}

func (g *gateSoongVarEqValue) emit(gctx *generationContext) {
	gctx.writeIndent()
	if g.first {
		gctx.write("if ")
	} else {
		gctx.write("elif ")
	}
	gctx.writef("%s(%q)", soongVarFunctionName, g.name)
	if g.negate {
		gctx.write(" != ")
	} else {
		gctx.write(" == ")
	}
	gctx.writef("%q:\n", g.value)
}

type gateVariableOneOf struct {
	name   string
	values []string
	negate bool
	first  bool
}

func (g *gateVariableOneOf) emit(gctx *generationContext) {
	gctx.writeIndent()
	if g.first {
		gctx.write("if ")
	} else {
		gctx.write("elif ")
	}
	gctx.writef("%s(%q)", soongVarFunctionName, g.name)
	if g.negate {
		gctx.write(" not in ")
	} else {
		gctx.write(" in ")
	}
	gctx.writef("[\"%s\"]:\n", strings.Join(g.values, "\", \""))
}

type gateCallFunction struct {
	call   string
	negate bool
	first  bool
}

func (g *gateCallFunction) emit(gctx *generationContext) {
	gctx.writeIndent()
	ifElseIf := "if"
	if !g.first {
		ifElseIf = "elif"
	}
	if g.negate {
		gctx.writef("%s not %s:\n", ifElseIf, g.call)
	} else {
		gctx.writef("%s %s:\n", ifElseIf, g.call)
	}
}

// switchCase represents as single if/elseif/else branch. All the necessary
// info about flavor (if/elseif/else) is supposed to be kept in `gate`.
type switchCase struct {
	gate  starlarkNode
	nodes []starlarkNode
}

func (cb *switchCase) newNode(node starlarkNode) {
	cb.nodes = append(cb.nodes, node)
}

func (cb *switchCase) emit(gctx *generationContext) {
	cb.gate.emit(gctx)
	gctx.indentLevel++
	hasStatements := false
	if len(cb.nodes) > 0 {
		for _, node := range cb.nodes {
			if _, ok := node.(*commentNode); !ok {
				hasStatements = true
			}
			node.emit(gctx)
		}
	}
	if !hasStatements {
		gctx.emitPass()
	}
	gctx.indentLevel--
}

// A single complete if ... elseif ... else ... endif sequences
type switchNode struct {
	ssCases      []*switchCase
	functionName string
}

func (ssw *switchNode) newNode(node starlarkNode) {
	switch br := node.(type) {
	case *switchCase:
		ssw.ssCases = append(ssw.ssCases, br)
	default:
		panic(fmt.Errorf("expected switchCase node, got %t", br))
	}
}

func (ssw *switchNode) emit(gctx *generationContext) {
	if ssw.functionName != "" {
		gctx.writef("def %s():\n", ssw.functionName)
		gctx.indentLevel++
	}
	for _, ssCase := range ssw.ssCases {
		ssCase.emit(gctx)
	}
	if len(ssw.ssCases) == 0 {
		gctx.emitPass()
	}
	if ssw.functionName != "" {
		gctx.indentLevel--
		gctx.writef("%s()\n\n", ssw.functionName)
	}
}

type StarScript struct {
	mkFile                                string
	moduleName                            string
	mkPos                                 scanner.Position
	nodes                                 []starlarkNode
	subConfigFirst, subConfigLast         *subConfigNode
	varAssignmentFirst, varAssignmentLast *assignmentNode
	hasErrors                             bool
	emitPrint                             bool
}

func (ss *StarScript) newNode(node starlarkNode) {
	ss.nodes = append(ss.nodes, node)
}

type parseContext struct {
	starScript       *StarScript
	nodes            []mkparser.Node
	currentNodeIndex int
	ifNestLevel      int
	moduleNameCount  map[string]int
	ifCount          int
}

func newParseContext(ss *StarScript, nodes []mkparser.Node) *parseContext {
	return &parseContext{
		starScript:       ss,
		nodes:            nodes,
		currentNodeIndex: 0,
		ifNestLevel:      0,
		moduleNameCount:  make(map[string]int),
	}
}

func (ctx *parseContext) hasNode() bool {
	return ctx.currentNodeIndex < len(ctx.nodes)
}

func (ctx *parseContext) getNode() mkparser.Node {
	if !ctx.hasNode() {
		return nil
	}
	node := ctx.nodes[ctx.currentNodeIndex]
	ctx.currentNodeIndex++
	return node
}

func (ctx *parseContext) backNode() {
	if ctx.currentNodeIndex <= 0 {
		panic("Cannot back off")
	}
	ctx.currentNodeIndex--
}

const (
	macroMaybeAddToProductCopyFiles = "add-to-product-copy-files-if-exists"
	macroFindCopySubdirFiles        = "find-copy-subdir-files"
)

type evalChecker struct {
	valid     bool
	functions string
}

func newEvalChecker() *evalChecker {
	return &evalChecker{valid: true}
}

func (e *evalChecker) Get(name string) string {
	return name
}

func (e *evalChecker) Set(_, _ string) {
}

func (e *evalChecker) Call(name string, _ []string) []string {
	/* TODO(asmundak): first make assignmentNode handle expressions
	if name == macroFindCopySubdirFiles || name == macroMaybeAddToProductCopyFiles {
		return []string{name}
	}
	*/
	e.valid = false
	if e.functions != "" {
		e.functions += ":" + name
	} else {
		e.functions = name
	}
	return []string{name}
}

func (e *evalChecker) SetFunc(_ string, _ func([]string) []string) {
}

func (ctx *parseContext) handleAssignment(a *mkparser.Assignment, receiver nodeReceiver) {
	ss := ctx.starScript
	// Handle only simple variables
	if len(a.Name.Strings) > 1 {
		ctx.errorf(a, receiver, "Only simple variables are handled")
		return
	}
	name := a.Name.Strings[0]
	if _, ok := KnownVariables.vars[name]; !ok {
		ctx.errorf(a, receiver, "%s is not listed as product configuration variable", name)
		return
	}

	// Evaluate RHS. We don't care about the variables
	// TODO(asmundak): eval variables
	// but we do care about function calls
	evalChecker := newEvalChecker()
	_ = a.Value.Value(evalChecker)
	if !evalChecker.valid {
		ctx.errorf(a, receiver, "Calls in RHS: %s", evalChecker.functions)
		return
	}
	asgn := &assignmentNode{name: name, value: a.Value, flavor: a.Type}
	receiver.newNode(asgn)
	if ss.varAssignmentLast != nil {
		ss.varAssignmentLast.next = asgn
	}
	ss.varAssignmentLast = asgn
	if ss.varAssignmentFirst == nil {
		ss.varAssignmentFirst = asgn
	}
}

var simpleFunctionMapping = map[string]string{
	"info ":    baseName + ".mkinfo",
	"warning ": baseName + ".mkwarning",
	"error ":   baseName + ".mkerror",
}

func (ctx *parseContext) handleVariable(v *mkparser.Variable, receiver nodeReceiver) {
	// The result of parsing
	//   $(call inherit-product, foo)
	// is a variable (its first token is 'call inherit-product ')
	// We are mostly interested in those, but we also handle
	//   $(info xxx)
	//   $(warning xxx)
	//   $(error xxx)
	name := v.Name
	first := strings.TrimSpace(name.Strings[0])
	// Handle simple ones first
	for prefix, f := range simpleFunctionMapping {
		if strings.HasPrefix(first, prefix) {
			text := v.Dump()
			text = strings.TrimSuffix(strings.TrimPrefix(text, "$("), ")")
			text = fmt.Sprintf("%s(%q, %q)\n", f,
				ctx.starScript.mkFile,
				strings.TrimSpace(strings.TrimPrefix(text, prefix)))
			receiver.newNode(&literalNode{text})
			return
		}
	}
	loadAlways := true
	const callLoadAlways = "call inherit-product,"
	const callLoadIf = "call inherit-product-if-exists,"
	if strings.HasPrefix(first, callLoadAlways) {
		first = strings.TrimSpace(strings.TrimPrefix(first, callLoadAlways))
	} else if strings.HasPrefix(first, callLoadIf) {
		first = strings.TrimSpace(strings.TrimPrefix(first, callLoadIf))
		loadAlways = false
	} else {
		if strings.HasPrefix(first, "call ") {
			ctx.errorf(v, receiver,
				"cannot convert calling '%s'",
				strings.TrimSpace(strings.SplitN(strings.TrimPrefix(first, "call "), ",", 2)[0]))
		} else {
			ctx.errorf(v, receiver, "unsupported variable %s", first)
		}
		return
	}
	var path string
	if len(name.Strings) == 1 {
		// Constant path
		path = first
	} else if first != "" || len(name.Strings) != 2 || len(name.Variables) != 1 {
		ctx.errorf(v, receiver, "unsupported line", v.Dump())
		return
	} else {
		// TODO(asmundak): call name.Value(scope) instead of looking for specific variables.
		varName := name.Variables[0].Name
		if len(varName.Strings) == 1 {
			if varName.Strings[0] == "SRC_TARGET_DIR" {
				path = filepath.Join("build", "make", "target", strings.TrimSpace(name.Strings[1]))
			} else if varName.Strings[0] == "LOCAL_PATH" {
				path = filepath.Join(filepath.Dir(ctx.starScript.mkFile), strings.TrimSpace(name.Strings[1]))
			} else {
				ctx.errorf(v, receiver, "unsupported variable reference %s", name.Dump())
				return
			}
		} else {
			ctx.errorf(v, receiver, "unsupported variable reference %s", name.Dump())
			return
		}
	}
	suffix := filepath.Ext(path)
	starPath := strings.TrimSuffix(path, suffix) + StarlarkSuffix
	moduleName := moduleNameForFile(path)
	moduleLocalName := "_" + moduleName
	n, found := ctx.moduleNameCount[moduleName]
	if found {
		moduleLocalName += fmt.Sprintf("%d", n)
	}
	ctx.moduleNameCount[moduleName] = n + 1
	sc := &subConfigNode{
		path:            starPath,
		originalPath:    path,
		moduleName:      moduleName,
		moduleLocalName: moduleLocalName,
		loadAlways:      loadAlways,
	}
	ss := ctx.starScript
	receiver.newNode(sc)
	if ss.subConfigLast != nil {
		ss.subConfigLast.next = sc
	}
	ss.subConfigLast = sc
	if ss.subConfigFirst == nil {
		ss.subConfigFirst = sc
	}
}

func (ctx *parseContext) handleDefine(directive *mkparser.Directive, receiver nodeReceiver) {
	tokens := strings.Fields(directive.Args.Strings[0])
	ctx.errorf(directive, receiver, "define is not supported: %s", tokens[0])
}

func (ctx *parseContext) handleIfBlock(ifDirective *mkparser.Directive, receiver nodeReceiver) {
	ssSwitch := &switchNode{}
	if ctx.ifNestLevel == 0 {
		ssSwitch.functionName = "_maybe"
		if ctx.ifCount > 0 {
			ssSwitch.functionName += fmt.Sprintf("%d", ctx.ifCount)
		}
		ctx.ifCount++
	}
	err := ctx.processBranch(ifDirective, ssSwitch)
	if err != nil {
		return
	}

	for ctx.hasNode() {
		node := ctx.getNode()
		switch x := node.(type) {
		case *mkparser.Directive:
			switch x.Name {
			case "else", "elifdef", "elifndef", "elifeq", "elifneq":
				err := ctx.processBranch(x, ssSwitch)
				if err != nil {
					return
				}
			case "endif":
				receiver.newNode(ssSwitch)
				return
			default:
				ctx.errorf(node, receiver, "unexpected directive %s", x.Name)
			}
		default:
			ctx.errorf(ifDirective, receiver, "unexpected statement")
		}
	}
	ctx.errorf(ifDirective, receiver, "no matching endif", ifDirective.Dump())
}

// processBranch processes a single branch (if/elseif/else) until the next directive
// on the same level.
func (ctx *parseContext) processBranch(check *mkparser.Directive, receiver nodeReceiver) error {
	block := switchCase{gate: ctx.parseCondition(check)}
	ctx.ifNestLevel++
	for ctx.hasNode() {
		node := ctx.getNode()
		if ctx.handleSimpleStatement(node, &block) {
			continue
		}
		switch d := node.(type) {
		case *mkparser.Directive:
			switch d.Name {
			case "else", "elifdef", "elifndef", "elifeq", "elifneq", "endif":
				receiver.newNode(&block)
				ctx.backNode()
				ctx.ifNestLevel--
				return nil
			case "ifdef", "ifndef", "ifeq", "ifneq":
				ctx.handleIfBlock(d, &block)
			default:
				return fmt.Errorf("unexpected directive %s", d.Name)
			}
		default:
			// TODO(asmundak): should it just register bas statement and keep going?
			return fmt.Errorf("unexpected statement %s", node.Dump())
		}
	}
	ctx.ifNestLevel--
	return fmt.Errorf("no matching endif for %s", check.Dump())
}

func newGateIfdef(check *mkparser.Directive, negate, first bool) starlarkNode {
	if check.Args.Const() {
		return &gateIfdef{variableName: check.Args.Strings[0], negate: negate, first: first}
	} else {
		return &gateBad{which: "if", checkText: check.Dump()}
	}
}

func (ctx *parseContext) parseCondition(check *mkparser.Directive) starlarkNode {
	switch check.Name {
	case "ifdef":
		return newGateIfdef(check, false, true)
	case "ifndef":
		return newGateIfdef(check, true, true)
	case "elifdef":
		return newGateIfdef(check, false, false)
	case "elifndef":
		return newGateIfdef(check, true, false)
	case "ifeq", "ifneq", "elifeq", "elifneq":
		return ctx.parseCompare(check)
	case "else":
		if check.Args.Empty() {
			return &gateElse{}
		} else {
			return &gateBad{which: "elif", checkText: check.Dump()}
		}
	default:
		panic(fmt.Errorf("%s: unknown directive: %s", ctx.starScript.mkFile, check.Dump()))
	}
}

func (ctx *parseContext) newBadGate(which string, text string, args ...interface{}) starlarkNode {
	message := fmt.Sprintf(text, args...)
	ErrorMonitor.NewError(text, nil, args)
	return &gateBad{which: which, checkText: message}
}

func (ctx *parseContext) parseCompare(cond *mkparser.Directive) starlarkNode {
	args := cond.Args.Split(",")
	if len(args) != 2 {
		return ctx.newBadGate(cond.Name, "ifeq/ifneq len(args) != 2 %s", cond.Dump())
	}

	// Handle only the case when one of the operands is constant. Save it in valueArg, save the
	// other operand in varArg, after trimming the enclosing parentheses.
	var valueArg string
	var varArg *mkparser.MakeString
	if args[0].Const() {
		// Don't forget to remove parens!
		valueArg = strings.TrimSpace(strings.TrimLeft(args[0].Strings[0], "("))
		varArg = args[1]
		if n := len(varArg.Strings); n > 1 {
			varArg.Strings[n-1] = strings.TrimRight(varArg.Strings[n-1], ") ")
		}
	} else if args[1].Const() {
		valueArg = strings.TrimSpace(strings.TrimRight(args[1].Strings[0], ")"))
		varArg = args[0]
		varArg.Strings[0] = strings.TrimLeft(varArg.Strings[0], "( ")
	} else {
		return ctx.newBadGate(cond.Name, "not constant (%s,%s)",
			args[0].Dump(), args[1].Dump())
	}

	// The non-constant argument that we can handle can be:
	if gate, ok := ctx.parseCompareToVar(cond, valueArg, varArg); ok {
		return gate
	} else if gate, ok := ctx.parseCompareToFunctionResult(cond, valueArg, varArg); ok {
		return gate
	} else {
		return ctx.newBadGate("if", "expression too complex: %s", varArg.Dump())
	}
}

func (ctx *parseContext) parseCompareToVar(cond *mkparser.Directive,
	value string, varArg *mkparser.MakeString) (starlarkNode, bool) {

	name, ok := simpleVariableRef(varArg)
	if !ok {
		return nil, false
	}
	// $(VAR) case
	varInfo, ok := KnownVariables.vars[name]
	if !ok {
		return ctx.newBadGate(cond.Name, "unknown var %s", name), true
	}
	negate, first := negateAndFirstModifiers(cond)
	if varInfo.class == "soong" {
		return &gateSoongVarEqValue{name: name, value: value, negate: negate, first: first}, true
	}
	// TODO(asmundak): handle product config variable
	return ctx.newBadGate(cond.Name, "unknown var class %s", varInfo.class), true
}

func (ctx *parseContext) parseCompareToFunctionResult(cond *mkparser.Directive, value string,
	varArg *mkparser.MakeString) (starlarkNode, bool) {
	funcName, args, ok := simpleFunctionCall(varArg)
	if !ok {
		return nil, false
	}

	negate, first := negateAndFirstModifiers(cond)
	// $(call FUNC, arg1, arg2)
	switch funcName {
	case "filter", "filter-out":
		// We handle only empty filter result, i.e.
		//   ifeq/ifneq (,$(filter arg1, arg2))
		if value != "" {
			return ctx.newBadGate(cond.Name, "filter non-empty arg: %s", value), true
		}
		// Either first or second arguments should be const, and the
		// non-const one should be simple variable reference
		if args[0].Const() {
			name, ok := simpleVariableRef(args[1])
			if !ok {
				return ctx.newBadGate(cond.Name, "filter source is too complex: %s", args[1].Dump()), true
			}
			// if[eq|neq] (,$([filter|filter-out] value1 value2 ..., $(VAR))
			if funcName == "filter" {
				negate = !negate
			}
			return &gateVariableOneOf{name: name, values: strings.Fields(args[0].Dump()), negate: negate, first: first}, true
		} else if args[1].Const() {
			// ifeq/ifneq (,$(filter $(VAR), value1 value2 ...)
			name, ok := simpleVariableRef(args[0])
			if !ok {
				return ctx.newBadGate(cond.Name, "filter is too complex: %s", args[0].Dump()), true
			}
			return &gateVariableOneOf{name: name, values: strings.Fields(args[1].Dump()), negate: negate, first: first}, true
		}
		return ctx.newBadGate(cond.Name, "filter source too complex: %s", args[1].Dump()), true
	case "wildcard":
		if args[0].Const() {
			if value != "" {
				return ctx.newBadGate(cond.Name, "wildcard non-empty arg: %s", value), true
			}
			path := args[0].Strings[0]
			var callFunc string
			if strings.ContainsAny(path, "*?{[") && !strings.Contains(path, "\\") {
				callFunc = fileWildcardFunctionName
			} else {
				callFunc = fileExistsFunctionName
			}
			return &gateCallFunction{
				call:   fmt.Sprintf("%s(%q)", callFunc, path),
				negate: !negate,
				first:  first}, true
		}
		return ctx.newBadGate(cond.Name, "Unexpected wildcard argument %s", varArg.Dump()), true
	default:
		return ctx.newBadGate(cond.Name, "Unknown func in ifeq: %s", funcName), true
	}
}

func negateAndFirstModifiers(cond *mkparser.Directive) (bool, bool) {
	switch cond.Name {
	case "ifeq":
		return false, true
	case "ifneq":
		return true, true
	case "elifeq":
		return false, false
	case "elifneq":
		return true, false
	default:
		panic(fmt.Errorf("unexpected condition directive %s", cond.Dump()))
	}
}

// Handles the statement whose treatment is the same in all contexts: comment,
// assignment, variable (which is a macro call in reality) and all constructs that
// do not handle in any context ('define directive and any unrecognized stuff).
// Return true if we handled it.
func (ctx *parseContext) handleSimpleStatement(node mkparser.Node, receiver nodeReceiver) bool {
	handled := true
	switch x := node.(type) {
	case *mkparser.Comment:
		ctx.insertComment("#"+x.Comment, receiver)
	case *mkparser.Assignment:
		ctx.handleAssignment(x, receiver)
	case *mkparser.Variable:
		ctx.handleVariable(x, receiver)
	case *mkparser.Directive:
		switch x.Name {
		case "define":
			ctx.handleDefine(x, receiver)
		default:
			handled = false
		}
	default:
		ctx.errorf(x, receiver, "unsupported line", receiver)
	}
	return handled
}

func (ctx *parseContext) insertComment(s string, receiver nodeReceiver) {
	receiver.newNode(&commentNode{strings.TrimSpace(s)})
}

func (ctx *parseContext) carryAsComment(failedNode mkparser.Node, receiver nodeReceiver) {
	for _, line := range strings.Split(failedNode.Dump(), "\n") {
		ctx.insertComment("# "+line, receiver)
	}
}

// records that the given node failed to be converted and includes an explanatory message
func (ctx *parseContext) errorf(failedNode mkparser.Node, receiver nodeReceiver,
	message string, args ...interface{}) {
	ErrorMonitor.NewError(message, failedNode, args...)
	message = fmt.Sprintf(message, args...)
	ctx.insertComment(fmt.Sprintf("# MK2STAR TRANSLATION ERROR: %s", message), receiver)
	ctx.carryAsComment(failedNode, receiver)
	ctx.starScript.hasErrors = true
}

func (ss *StarScript) path2BazelRef(path string) string {
	relPath, err := filepath.Rel(filepath.Dir(ss.mkFile), path)
	if err != nil || relPath[0] == '.' {
		return "//" + filepath.Dir(path) + ":" + filepath.Base(path)
	}
	return ":" + filepath.Base(path)
}

func (ss *StarScript) String() string {
	return NewGenerateContext(ss).emit()
}

func (ss *StarScript) SubConfigFiles() []string {
	var subs []string
	for sc := ss.subConfigFirst; sc != nil; sc = sc.next {
		subs = append(subs, sc.originalPath)
	}
	return subs
}

func (ss *StarScript) HasErrors() bool {
	return ss.hasErrors
}

func Convert(mkFile string, reader io.Reader, emitPrint bool) (*StarScript, error) {
	if reader == nil {
		mkContents, err := ioutil.ReadFile(mkFile)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewBuffer(mkContents)
	}
	parser := mkparser.NewParser(mkFile, reader)
	nodes, errs := parser.Parse()
	if len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintln(os.Stderr, "ERROR:", e)
		}
		return nil, fmt.Errorf("bad makefile %s", mkFile)
	}
	starScript := &StarScript{
		moduleName: moduleNameForFile(mkFile),
		mkFile:     mkFile,
		emitPrint:  emitPrint,
	}
	ctx := newParseContext(starScript, nodes)
	for ctx.hasNode() {
		node := ctx.getNode()
		if ctx.handleSimpleStatement(node, starScript) {
			continue
		}
		switch x := node.(type) {
		case *mkparser.Directive:
			switch x.Name {
			case "ifeq", "ifneq", "ifdef", "ifndef":
				ctx.handleIfBlock(x, starScript)
			default:
				ctx.errorf(x, starScript, "unsupported directive "+x.Name)
			}
		default:
			ctx.errorf(x, starScript, "unsupported line")
		}
	}
	return starScript, nil
}
