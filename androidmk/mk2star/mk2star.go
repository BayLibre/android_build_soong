// Convert makefile containing device configuration to Starlark file
// The conversion can handle the following constructs in a makefile:
//   * comments
//   * simple variable assignments
//   * $(call init-product,<file>)
//   * $(call inherit-product-if-exists
//   * if directives
// All other constructs are carried over to the output starlark file as comments.
//
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
	"strconv"
	"strings"
	"text/scanner"
)

const (
	baseUri  = "//build/make/target/product:product_config"
	baseName = "rblf"

	configFunctionName    = baseName + ".prodconf"
	printvarsFunctionName = baseName + ".printvars"
	warningFunctionName   = baseName + ".warning"
	ifdefVarFunctionName  = baseName + ".is_defined"
	soongVarFunctionName  = baseName + ".soong_var"

	targetBoardPlatformVarName = "TARGET_BOARD_PLATFORM"
	targetProductVarName       = "TARGET_PRODUCT"
)

const (
	callLoadAlways        = "inherit-product"
	callLoadIf            = "inherit-product-if-exists"
	rtFileExists          = "$file_exists"
	rtWildcardExists      = "$wildcard_exists"
	isBoardPlatform       = "is-board-platform"
	isBoardPlatformInList = "is-board-platform-in-list"
	isProductInList       = "is-product-in-list"
	isVendorBoardPlatform = "is-vendor-board-platform"
)

var knownFunctions = map[string]string{
	rtFileExists:                          baseName + ".file_exists",
	rtWildcardExists:                      baseName + ".file_wildcard_exists",
	"add-to-product-copy-files-if-exists": baseName + ".copy_if_exists",
	"addprefix":                           baseName + ".addprefix",
	"addsuffix":                           baseName + ".addsuffix",
	"enforce-product-packages-exist":      baseName + ".enforce_product_packages_exist",
	"error":                               baseName + ".mkerror",
	"findstring":                          "!findstring",
	"find-copy-subdir-files":              baseName + ".find_and_copy",
	"filter":                              "!filter",
	"filter-out":                          "!filter-out",
	"info":                                baseName + ".mkinfo",
	callLoadAlways:                        "!inherit-product",
	callLoadIf:                            "!inherit-product-if-exists",
	isBoardPlatform:                       "!is-board-platform",
	isBoardPlatformInList:                 "!is-board-platform-in-list",
	isProductInList:                       "!is-product-in-list",
	isVendorBoardPlatform:                 "!is-vendor-board-platform",
	"produce_copy_files":                  baseName + ".produce_copy_files",
	"require-artifacts-in-path":           baseName + ".require_artifacts_in_path",
	"strip":                               baseName + ".mkstrip",
	"warning":                             baseName + ".mkwarning",
	"wildcard":                            "!wildcard",
}

var builtinFuncRex = regexp.MustCompile(
	"^(addprefix|addsuffix|abspath|and|basename|call|dir|error|eval" +
		"|flavor|foreach|file|filter|filter-out|findstring|firstword|guile" +
		"|if|info|join|lastword|notdir|or|origin|patsubst|realpath" +
		"|shell|sort|strip|subst|suffix|value|warning|word|wordlist|words" +
		"|wildcard)")

type ErrorMonitorCB interface {
	NewError(s string, node mkparser.Node, args ...interface{})
}

type emptyErrorMonitor struct{}

func (_ emptyErrorMonitor) NewError(_ string, _ mkparser.Node, _ ...interface{}) {}

var StarlarkSuffix = ".star"
var ErrorMonitor ErrorMonitorCB = emptyErrorMonitor{}

// Derives module name for a given file. It is base name
// (file name without suffix), with some characters replaced to make it a Starlark identifier
func moduleNameForFile(mkFile string) string {
	base := strings.TrimSuffix(filepath.Base(mkFile), filepath.Ext(mkFile))
	// TODO(asmundak): what else can be in the product file names?
	return strings.ReplaceAll(base, "-", "_")
}

func cloneMakeString(mkString *mkparser.MakeString) *mkparser.MakeString {
	r := &mkparser.MakeString{StringPos: mkString.StringPos}
	r.Strings = append(r.Strings, mkString.Strings...)
	r.Variables = append(r.Variables, mkString.Variables...)
	return r
}

func isMakeControlFunc(s string) bool {
	return s == "error" || s == "warning" || s == "info"
}

// If MakeString is a single variable reference, returns it
func singleVar(v *mkparser.MakeString) (*mkparser.MakeString, bool) {
	if len(v.Strings) == 2 && strings.TrimSpace(v.Strings[0]) == "" && strings.TrimSpace(v.Strings[1]) == "" {
		return v.Variables[0].Name, true
	}
	return nil, false

}

// Extracts variable name and returns true if given MakeString is a simple variable reference

// Starlark output generation context
type generationContext struct {
	buf               strings.Builder
	starScript        *StarScript
	initDone          bool
	varLastAssignment map[string]*assignmentNode
	indentLevel       int
	inAssignment      bool
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
		gctx.writeln()

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
	gctx.write("pass")
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

type starlarkExpr interface {
	starlarkNode
	eval(valueMap map[string]starlarkExpr) (expr starlarkExpr, same bool)
}

type nodeReceiver interface {
	newNode(node starlarkNode)
}

// Types used to keep processed makefile data:
type commentNode struct {
	text string
}

func (c *commentNode) emit(gctx *generationContext) {
	chunks := strings.Split(c.text, "\\\n")
	gctx.writeIndent()
	gctx.write(chunks[0])
	for _, chunk := range chunks[1:] {
		gctx.writeln()
		gctx.writeIndent()
		gctx.write("#", chunk)
	}
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
	// TODO(asmundak): next line is gratuitous. load works on on top level
	gctx.writeIndent()
	if sc.loadAlways {
		gctx.writef(`load("%s", `, ss.path2BazelRef(sc.path))
	} else {
		gctx.writef(`load("%s|%s", `, ss.path2BazelRef(sc.path), sc.moduleName)
	}
	if sc.moduleLocalName != sc.moduleName {
		gctx.write(sc.moduleLocalName, ` = `)
	}
	gctx.write(`"`, sc.moduleName, `")`)
}

type assignmentNode struct {
	name    string
	value   starlarkExpr
	mkValue *mkparser.MakeString
	flavor  string
	next    *assignmentNode
}

func (asgn *assignmentNode) emit(gctx *generationContext) {
	desc, ok := KnownVariables.vars[asgn.name]
	if !ok {
		panic(fmt.Errorf("unknown variable %s", asgn.name))
	}
	// TODO: handle ?= assignments
	gctx.inAssignment = true
	notFirst := asgn != gctx.varLastAssignment[asgn.name]
	gctx.writeIndent()
	switch desc.class {
	case "product":
		gctx.write(asgn.referenceName())
		if asgn.flavor == "+=" && notFirst {
			gctx.write(" += ")
		} else {
			gctx.write(" = ")
		}
		asgn.value.emit(gctx)
	case "soong":
		gctx.writef("%s = None  ## TODO(soong) %q", asgn.referenceName(), asgn.mkValue.Dump())
		gctx.starScript.hasErrors = true
	default:
		panic(fmt.Errorf("variable %s has unexpected class %s", asgn.name, desc.class))
	}
	gctx.inAssignment = false
}

func (asgn assignmentNode) referenceName() string {
	return fmt.Sprintf("_vars[%q]", asgn.name)
}

type ifNode struct {
	isElif bool // true if this is 'elif' statement
	expr   starlarkExpr
}

func (in *ifNode) emit(gctx *generationContext) {
	ifElif := "if "
	if in.isElif {
		ifElif = "elif "
	}

	gctx.writeIndent()
	if bad, ok := in.expr.(*badExpr); ok {
		gctx.writeln("# MK2STAR ERROR converting:")
		gctx.writeIndent()
		gctx.writef("#   %s\n", bad.node.Dump())
		gctx.writeIndent()
		gctx.writef("# %s\n", bad.message)
		gctx.writeIndent()
		gctx.writef("%sFalse:\n", ifElif)
		return
	}
	gctx.write(ifElif)
	in.expr.emit(gctx)
	gctx.writeln(":")
}

type stringLiteralExpr struct {
	literal string
}

func (s *stringLiteralExpr) eval(_ map[string]starlarkExpr) (starlarkExpr, bool) {
	return s, true
}

func (s *stringLiteralExpr) emit(gctx *generationContext) {
	gctx.writef("%q", s.literal)
}

type intLiteralExpr struct {
	literal int
}

func (s *intLiteralExpr) eval(_ map[string]starlarkExpr) (starlarkExpr, bool) {
	return s, true
}

func (s *intLiteralExpr) emit(gctx *generationContext) {
	gctx.writef("%d", s.literal)
}

type interpolateExpr struct {
	chunks []string
	args   []starlarkExpr
}

func (xi *interpolateExpr) emit(gctx *generationContext) {
	if len(xi.chunks) != len(xi.args)+1 {
		panic(fmt.Errorf("#chunks(%d) != #args(%d)+1", len(xi.chunks), len(xi.args)))
	}
	format := strings.ReplaceAll(xi.chunks[0], "%", "%%")
	for _, chunk := range xi.chunks[1:] {
		format += "%s" + strings.ReplaceAll(chunk, "%", "%%")
	}
	gctx.writef("%q", format)
	sep := " % ("
	for _, arg := range xi.args {
		gctx.write(sep)
		sep = " ,"
		arg.emit(gctx)
	}
	gctx.write(")")
}

func (xi *interpolateExpr) eval(valueMap map[string]starlarkExpr) (starlarkExpr, bool) {
	same := true
	newChunks := []string{xi.chunks[0]}
	var newArgs []starlarkExpr
	for i, arg := range xi.args {
		newArg, sameArg := arg.eval(valueMap)
		same = same && sameArg
		switch x := newArg.(type) {
		case *stringLiteralExpr:
			newChunks[len(newChunks)-1] += x.literal + xi.chunks[i+1]
			same = false
			continue
		case *intLiteralExpr:
			newChunks[len(newChunks)-1] += strconv.Itoa(x.literal) + xi.chunks[i+1]
			same = false
			continue
		default:
			newChunks = append(newChunks, xi.chunks[i+1])
			newArgs = append(newArgs, newArg)
		}
	}
	if same {
		return xi, true
	}
	if len(newChunks) == 1 {
		return &stringLiteralExpr{newChunks[0]}, false
	}
	return &interpolateExpr{chunks: newChunks, args: newArgs}, false
}

type variableRefExpr struct {
	name string
}

func (v *variableRefExpr) eval(valueMap map[string]starlarkExpr) (starlarkExpr, bool) {
	if x, ok := valueMap[v.name]; ok {
		return x, false
	}
	return v, true
}

func (v *variableRefExpr) emit(gctx *generationContext) {
	switch cl := KnownVariables.vars[v.name].class; cl {
	case "soong":
		gctx.writef("%s(%q)", soongVarFunctionName, v.name)
	case "product":
		gctx.writef("_vars[%q]", v.name)
	default:
		panic(fmt.Errorf("implement referencing '%s' class for variable %s", cl, v.name))
	}
}

// returns an expr referencing given variable or badExpr
func (ctx *parseContext) newVariableRef(node mkparser.Node, name string) (starlarkExpr, bool) {
	if _, found := ctx.knownMkVars[name]; found {
		return &variableRefExpr{name}, true
	}
	varInfo, ok := KnownVariables.vars[name]
	if !ok {
		return ctx.newBadExpr(node, "unknown variable %s", name), false
	}
	switch varInfo.class {
	case "soong", "product":
		return &variableRefExpr{name}, true
	default:
		// TODO(asmundak): handle product config variable
		return ctx.newBadExpr(node, "unknown variable class %s", varInfo.class), false
	}
}

type notExpr struct {
	expr starlarkExpr
}

func (n *notExpr) eval(valueMap map[string]starlarkExpr) (starlarkExpr, bool) {
	if x, same := n.expr.eval(valueMap); !same {
		return &notExpr{expr: x}, false
	}
	return n, true
}

func (n *notExpr) emit(ctx *generationContext) {
	ctx.write("not ")
	n.expr.emit(ctx)
}

type eqExpr struct {
	left, right starlarkExpr
	isEq        bool // if false, it's !=
}

func (eq *eqExpr) eval(valueMap map[string]starlarkExpr) (expr starlarkExpr, same bool) {
	xLeft, sameLeft := eq.left.eval(valueMap)
	xRight, sameRight := eq.right.eval(valueMap)
	if sameLeft && sameRight {
		return eq, true
	}
	return &eqExpr{left: xLeft, right: xRight, isEq: eq.isEq}, false
}

func (eq *eqExpr) emit(gctx *generationContext) {
	eq.left.emit(gctx)
	if eq.isEq {
		gctx.write(" == ")
	} else {
		gctx.write(" != ")
	}
	eq.right.emit(gctx)
}

type variableDefinedExpr struct {
	name string
}

func (v *variableDefinedExpr) eval(_ map[string]starlarkExpr) (expr starlarkExpr, same bool) {
	return v, true

}

func (v *variableDefinedExpr) emit(gctx *generationContext) {
	gctx.writef("%s(%q)", ifdefVarFunctionName, v.name)
}

type listExpr struct {
	items []starlarkExpr
}

func (l *listExpr) eval(valueMap map[string]starlarkExpr) (expr starlarkExpr, same bool) {
	newItems := make([]starlarkExpr, len(l.items))
	same = true
	for i, item := range l.items {
		var sameItem bool
		newItems[i], sameItem = item.eval(valueMap)
		same = same && sameItem
	}
	if same {
		return l, true
	}
	return &listExpr{newItems}, false
}

func (l *listExpr) emit(gctx *generationContext) {
	if !gctx.inAssignment || len(l.items) < 2 {
		gctx.write("[")
		sep := ""
		for _, item := range l.items {
			gctx.write(sep)
			item.emit(gctx)
			sep = ", "
		}
		gctx.write("]")
		return
	}

	gctx.writeln("[")
	gctx.indentLevel += 2
	for _, item := range l.items {
		gctx.writeIndent()
		item.emit(gctx)
		gctx.writeln(",")
	}
	gctx.indentLevel -= 2
	gctx.writeIndent()
	gctx.write("]")
}

func newStringListExpr(items []string) starlarkExpr {
	v := listExpr{}
	for _, item := range items {
		v.items = append(v.items, &stringLiteralExpr{item})
	}
	return &v
}

type concatExpr struct {
	items []starlarkExpr
}

func (c *concatExpr) emit(gctx *generationContext) {
	c.items[0].emit(gctx)
	if len(c.items) == 1 {
		return
	}
	if !gctx.inAssignment {
		for _, item := range c.items[1:] {
			gctx.write(" + ")
			item.emit(gctx)
		}
		return
	}
	gctx.indentLevel += 2
	for _, item := range c.items[1:] {
		gctx.writeln(" +")
		gctx.writeIndent()
		item.emit(gctx)
	}
	gctx.indentLevel -= 2
}

func (c *concatExpr) eval(valueMap map[string]starlarkExpr) (expr starlarkExpr, same bool) {
	same = true
	xConcat := &concatExpr{items: make([]starlarkExpr, len(c.items))}
	for i, item := range c.items {
		var sameItem bool
		xConcat.items[i], sameItem = item.eval(valueMap)
		same = same && sameItem
	}
	if same {
		return c, true
	}
	return xConcat, false
}

type inExpr struct {
	expr  starlarkExpr
	list  starlarkExpr
	isNot bool
}

func (i *inExpr) eval(valueMap map[string]starlarkExpr) (expr starlarkExpr, same bool) {
	res := &inExpr{isNot: i.isNot}
	var sameExpr, sameList bool
	res.expr, sameExpr = i.expr.eval(valueMap)
	res.list, sameList = i.list.eval(valueMap)
	if sameExpr && sameList {
		return i, true
	}
	return res, false
}

func (i *inExpr) emit(gctx *generationContext) {
	i.expr.emit(gctx)
	if i.isNot {
		gctx.write(" not in ")
	} else {
		gctx.write(" in ")
	}
	i.list.emit(gctx)
}

type callExpr struct {
	object starlarkExpr // nil if static call
	name   string
	args   []starlarkExpr
}

func (c *callExpr) eval(valueMap map[string]starlarkExpr) (expr starlarkExpr, same bool) {
	newCallExpr := &callExpr{name: c.name, args: make([]starlarkExpr, len(c.args))}
	if c.object != nil {
		newCallExpr.object, same = c.object.eval(valueMap)
	} else {
		same = true
	}
	for i, args := range c.args {
		var s bool
		newCallExpr.args[i], s = args.eval(valueMap)
		same = same && s
	}
	if same {
		return c, true
	}
	return newCallExpr, false
}

func (c *callExpr) emit(gctx *generationContext) {

	if c.object != nil {
		c.object.emit(gctx)
		gctx.write(".", c.name, "(")
	} else {
		runtimeName, found := knownFunctions[c.name]
		if !found {
			panic(fmt.Errorf("callExpr with unknown function %q", c.name))
		}
		if runtimeName[0] == '!' {
			panic(fmt.Errorf("callExpr for %q should not be there", c.name))
		}
		gctx.write(runtimeName, "(")
	}
	sep := ""
	for _, arg := range c.args {
		gctx.write(sep)
		arg.emit(gctx)
		sep = ", "
	}
	gctx.write(")")
}

type badExpr struct {
	node    mkparser.Node
	message string
}

func (b *badExpr) eval(_ map[string]starlarkExpr) (expr starlarkExpr, same bool) {
	return b, true
}

func (b *badExpr) emit(_ *generationContext) {
	panic("implement me")
}

// parses $(...), returning an expression
func (ctx *parseContext) parseReference(node mkparser.Node, ref *mkparser.MakeString) starlarkExpr {
	ref.TrimLeftSpaces()
	ref.TrimRightSpaces()
	refDump := ref.Dump()

	// Handle only the case where the first (or only) word is constant
	words := ref.SplitN(" ", 2)
	if !words[0].Const() {
		return ctx.newBadExpr(node, "reference is too complex: %s", refDump)
	}

	// If it is a single word, it can be a simple variable
	// reference or a function call
	if len(words) == 1 {
		if isMakeControlFunc(refDump) {
			return &callExpr{name: refDump, args: []starlarkExpr{&stringLiteralExpr{""}}}
		}
		v, _ := ctx.newVariableRef(node, refDump)
		return v
	}

	expr := &callExpr{name: words[0].Dump()}
	args := words[1]
	args.TrimLeftSpaces()
	// So called "make control functions" need special treatment as everything
	// after the name is a single text argument
	if isMakeControlFunc(expr.name) {
		x := ctx.parseMakeString(node, args)
		if xBad, ok := x.(*badExpr); ok {
			return xBad
		}
		expr.args = []starlarkExpr{x}
		return expr
	}
	if expr.name == "call" {
		words = args.SplitN(",", 2)
		if words[0].Empty() || !words[0].Const() {
			return ctx.newBadExpr(nil, "cannot handle %s", refDump)
		}
		expr.name = words[0].Dump()
		if len(words) < 2 {
			return expr
		}
		args = words[1]
	}
	if _, found := knownFunctions[expr.name]; !found {
		return ctx.newBadExpr(node, "cannot handle invoking %s", expr.name)
	}
	for _, arg := range args.Split(",") {
		arg.TrimLeftSpaces()
		arg.TrimRightSpaces()
		x := ctx.parseMakeString(node, arg)
		if xBad, ok := x.(*badExpr); ok {
			return xBad
		}
		expr.args = append(expr.args, x)
	}
	return expr
}

func (ctx *parseContext) parseMakeString(node mkparser.Node, mk *mkparser.MakeString) starlarkExpr {
	if mk.Const() {
		return &stringLiteralExpr{mk.Dump()}
	}
	if len(mk.Variables) == 1 && mk.Strings[0] == "" && mk.Strings[1] == "" {
		mkRef := mk.Variables[0].Name
		return ctx.parseReference(node, mkRef)
	}
	xInterp := &interpolateExpr{args: make([]starlarkExpr, len(mk.Variables))}
	for i, ref := range mk.Variables {
		arg := ctx.parseReference(node, ref.Name)
		if x, ok := arg.(*badExpr); ok {
			return x
		}
		xInterp.args[i] = arg
	}
	xInterp.chunks = append(xInterp.chunks, mk.Strings...)
	return xInterp
}

type elseNode struct{}

func (br *elseNode) emit(gctx *generationContext) {
	gctx.writeIndent()
	gctx.writeln("else:")
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
	emitNode := func(node starlarkNode) {
		if _, ok := node.(*commentNode); !ok {
			hasStatements = true
		}
		node.emit(gctx)
	}
	if len(cb.nodes) > 0 {
		emitNode(cb.nodes[0])
		for _, node := range cb.nodes[1:] {
			gctx.writeln()
			emitNode(node)
		}
		if !hasStatements {
			gctx.writeln()
			gctx.emitPass()
		}
	} else {
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

	if len(ssw.ssCases) == 0 {
		gctx.emitPass()
	} else {
		ssw.ssCases[0].emit(gctx)
		for _, ssCase := range ssw.ssCases[1:] {
			gctx.writeln()
			ssCase.emit(gctx)
		}
	}
	if ssw.functionName != "" {
		gctx.indentLevel--
		gctx.writef("\n%s()\n", ssw.functionName)
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
	topDir                                string
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
	fatalError       error
	knownMkVars      map[string]starlarkExpr
}

func newParseContext(ss *StarScript, nodes []mkparser.Node) *parseContext {
	return &parseContext{
		starScript:       ss,
		nodes:            nodes,
		currentNodeIndex: 0,
		ifNestLevel:      0,
		moduleNameCount:  make(map[string]int),
		knownMkVars: map[string]starlarkExpr{
			"SRC_TARGET_DIR": &stringLiteralExpr{filepath.Join("build", "make", "target")},
			"LOCAL_PATH":     &stringLiteralExpr{filepath.Dir(ss.mkFile)},
			"TOPDIR":         &stringLiteralExpr{ss.topDir},
			// TODO(asmundak): to process internal config files, we need the following variables:
			//    BOARD_CONFIG_VENDOR_PATH
			//    TARGET_VENDOR
			//    target_base_product
			//
		},
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

func (ctx *parseContext) handleAssignment(a *mkparser.Assignment, receiver nodeReceiver) {
	ss := ctx.starScript
	// Handle only simple variables
	if len(a.Name.Strings) > 1 {
		ctx.errorf(a, receiver, "Only simple variables are handled")
		return
	}
	name := a.Name.Strings[0]
	varInfo, ok := KnownVariables.vars[name]
	if !ok {
		ctx.errorf(a, receiver, "unknown variable %s", name)
		return
	}
	asgn := &assignmentNode{name: name, mkValue: a.Value, flavor: a.Type}
	if varInfo.flavor == "list" {
		items := a.Value.Words()
		// A function call in RHS is supposed to return a list, all other item expressions return
		// individual elements.
		xConcat := &concatExpr{}
		var xItemList *listExpr
		for _, item := range items {
			switch x := ctx.parseMakeString(a, item).(type) {
			case *badExpr:
				ctx.wrapBadExpr(x, receiver)
				return
			case *callExpr:
				if xItemList != nil {
					xConcat.items = append(xConcat.items, xItemList)
					xItemList = nil
				}
				xConcat.items = append(xConcat.items, x)
			default:
				if xItemList == nil {
					xItemList = &listExpr{[]starlarkExpr{x}}
				} else {
					xItemList.items = append(xItemList.items, x)
				}
			}
		}
		if xItemList != nil {
			xConcat.items = append(xConcat.items, xItemList)
		}
		switch len(xConcat.items) {
		case 0:
			asgn.value = &listExpr{}
		case 1:
			asgn.value = xConcat.items[0]
		default:
			asgn.value = xConcat
		}
	} else {
		asgn.value = ctx.parseMakeString(a, a.Value)
		if xBad, ok := asgn.value.(*badExpr); ok {
			ctx.wrapBadExpr(xBad, receiver)
			return
		}
	}

	// TODO(asmundak): move evaluation to a separate pass
	asgn.value, _ = asgn.value.eval(ctx.knownMkVars)

	receiver.newNode(asgn)
	if ss.varAssignmentLast != nil {
		ss.varAssignmentLast.next = asgn
	}
	ss.varAssignmentLast = asgn
	if ss.varAssignmentFirst == nil {
		ss.varAssignmentFirst = asgn
	}
}

func (ctx *parseContext) handleLoadConfig(v mkparser.Node, receiver nodeReceiver,
	pathExpr starlarkExpr, loadAlways bool) {
	var path string
	x, _ := pathExpr.eval(ctx.knownMkVars)
	s, ok := x.(*stringLiteralExpr)
	if !ok {
		ctx.errorf(v, receiver, "inherit-product/include argument is too complex")
		return
	}

	path = s.literal
	// Finally, figure out the loaded module path and name and create a node for it
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

	// chain subconfigs
	if ss.subConfigLast != nil {
		ss.subConfigLast.next = sc
	}
	ss.subConfigLast = sc
	if ss.subConfigFirst == nil {
		ss.subConfigFirst = sc
	}
}

func (ctx *parseContext) handleVariable(v *mkparser.Variable, receiver nodeReceiver) {
	// The result of parsing
	//   $(call inherit-product, foo)
	// is a variable (its first token is 'call inherit-product ')
	// We are mostly interested in those, but we also handle
	//   $(info xxx)
	//   $(warning xxx)
	//   $(error xxx)
	expr := ctx.parseReference(v, v.Name)
	switch x := expr.(type) {
	case *callExpr:
		if x.name == callLoadAlways || x.name == callLoadIf {
			ctx.handleLoadConfig(v, receiver, x.args[0], x.name == callLoadAlways)
		} else if isMakeControlFunc(x.name) {
			receiver.newNode(&callExpr{
				name: x.name,
				args: []starlarkExpr{
					&stringLiteralExpr{ctx.starScript.mkFile},
					x.args[0],
				},
			})
		} else {
			receiver.newNode(expr)
		}
	case *badExpr:
		ctx.wrapBadExpr(x, receiver)
		return
	default:
		ctx.errorf(v, receiver, "cannot handle %s", v.Dump())
		return
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

	for ctx.processBranch(ifDirective, ssSwitch); ctx.hasNode() && ctx.fatalError == nil; {
		node := ctx.getNode()
		switch x := node.(type) {
		case *mkparser.Directive:
			switch x.Name {
			case "else", "elifdef", "elifndef", "elifeq", "elifneq":
				ctx.processBranch(x, ssSwitch)
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
	if ctx.fatalError == nil {
		ctx.fatalError = fmt.Errorf("no matching endif for %s", ifDirective.Dump())
	}
}

// processBranch processes a single branch (if/elseif/else) until the next directive
// on the same level.
func (ctx *parseContext) processBranch(check *mkparser.Directive, receiver nodeReceiver) {
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
				return
			case "ifdef", "ifndef", "ifeq", "ifneq":
				ctx.handleIfBlock(d, &block)
			default:
				ctx.errorf(d, &block, "unexpected directive %s", d.Name)
			}
		default:
			ctx.errorf(node, receiver, "unexpected statement")
		}
	}
	ctx.fatalError = fmt.Errorf("no matching endif for %s", check.Dump())
}

func (ctx *parseContext) newIfDefinedNode(check *mkparser.Directive) (starlarkExpr, bool) {
	if !check.Args.Const() {
		return ctx.newBadExpr(check, "ifdef variable ref too complex: %s", check.Args.Dump()), false
	}
	return &variableDefinedExpr{check.Args.Strings[0]}, true
}

func (ctx *parseContext) parseCondition(check *mkparser.Directive) starlarkNode {
	switch check.Name {
	case "ifdef", "ifndef", "elifdef", "elifndef":
		v, ok := ctx.newIfDefinedNode(check)
		if ok && strings.HasSuffix(check.Name, "ndef") {
			v = &notExpr{v}
		}
		return &ifNode{
			isElif: strings.HasPrefix(check.Name, "elif"),
			expr:   v,
		}
	case "ifeq", "ifneq", "elifeq", "elifneq":
		return &ifNode{
			isElif: strings.HasPrefix(check.Name, "elif"),
			expr:   ctx.parseCompare(check),
		}
	case "else":
		return &elseNode{}
	default:
		panic(fmt.Errorf("%s: unknown directive: %s", ctx.starScript.mkFile, check.Dump()))
	}
}

func (ctx *parseContext) newBadExpr(node mkparser.Node, text string, args ...interface{}) starlarkExpr {
	message := fmt.Sprintf(text, args...)
	ErrorMonitor.NewError(text, node, args)
	ctx.starScript.hasErrors = true
	return &badExpr{node, message}
}

func (ctx *parseContext) parseCompare(cond *mkparser.Directive) starlarkExpr {
	// Strip outer parentheses
	mkArg := cloneMakeString(cond.Args)
	mkArg.Strings[0] = strings.TrimLeft(mkArg.Strings[0], "( ")
	n := len(mkArg.Strings)
	mkArg.Strings[n-1] = strings.TrimRight(mkArg.Strings[n-1], ") ")
	args := mkArg.Split(",")
	// TODO(asmundak): handle the case where the arguments are in quotes and space-separated
	if len(args) != 2 {
		return ctx.newBadExpr(cond, "ifeq/ifneq len(args) != 2 %s", cond.Dump())
	}
	args[1].TrimLeftSpaces()

	isEq := !strings.HasSuffix(cond.Name, "neq")
	switch xLeft := ctx.parseMakeString(cond, args[0]).(type) {
	case *stringLiteralExpr, *variableRefExpr:
		switch xRight := ctx.parseMakeString(cond, args[1]).(type) {
		case *stringLiteralExpr, *variableRefExpr:
			return &eqExpr{left: xLeft, right: xRight, isEq: isEq}
		case *badExpr:
			return xRight
		default:
			expr, ok := ctx.parseCheckFunctionCallResult(cond, xLeft, args[1])
			if ok {
				return expr
			}
			return ctx.newBadExpr(cond, "right operand is too complex: %s", args[1].Dump())
		}
	case *badExpr:
		return xLeft
	default:
		switch xRight := ctx.parseMakeString(cond, args[1]).(type) {
		case *stringLiteralExpr, *variableRefExpr:
			expr, ok := ctx.parseCheckFunctionCallResult(cond, xRight, args[0])
			if ok {
				return expr
			}
			return ctx.newBadExpr(cond, "left operand is too complex: %s", args[1].Dump())
		case *badExpr:
			return xRight
		default:
			return ctx.newBadExpr(cond, "operands are too complex: (%s,%s)", args[0].Dump(), args[1].Dump())
		}
	}
}

func (ctx *parseContext) parseCheckFunctionCallResult(directive *mkparser.Directive, xValue starlarkExpr,
	varArg *mkparser.MakeString) (starlarkExpr, bool) {
	mkSingleVar, ok := singleVar(varArg)
	if !ok {
		return nil, false
	}
	expr := ctx.parseReference(directive, mkSingleVar)
	negate := strings.HasSuffix(directive.Name, "neq")
	switch x := expr.(type) {
	case *callExpr:
		switch x.name {
		case "filter":
			return ctx.parseCheckFilterFuncResult(directive, x, xValue, !negate), true
		case "filter-out":
			return ctx.parseCheckFilterFuncResult(directive, x, xValue, negate), true
		case "wildcard":
			return ctx.parseCompareWildcardFuncResult(directive, x, xValue, negate), true
		case "findstring":
			return ctx.parseCheckFindstringFuncResult(directive, x, xValue, negate), true
		case isBoardPlatform, isBoardPlatformInList, isProductInList, isVendorBoardPlatform:
			return ctx.parseCheckIsBoardXxxMacroResult(directive, x, xValue, negate), true
		default:
			return ctx.newBadExpr(directive, "Unknown function in ifeq: %s", x.name), true
		}
	case *badExpr:
		return x, true
	default:
		return nil, false
	}
}

func (ctx *parseContext) parseCheckFilterFuncResult(cond *mkparser.Directive,
	xCall *callExpr, xValue starlarkExpr, negate bool) starlarkExpr {
	// We handle:
	// *  ifeq/ifneq (,$(filter v1 v2 ..., $(VAR)) becomes if VAR not in/in ["v1", "v2", ...]
	// *  ifeq/ifneq (,$(filter $(VAR), v1 v2 ...) becomes if VAR not in/in ["v1", "v2", ...]
	// *  ifeq/ifneq ($(VAR),$(filter $(VAR), v1 v2 ...) becomes if VAR in/not in ["v1", "v2"]
	// TODO(Asmundak): check the last case works for filter-out, too.
	xPattern := xCall.args[0]
	xText := xCall.args[1]
	var xInList *stringLiteralExpr
	var xVar starlarkExpr
	var ok bool
	switch x := xValue.(type) {
	case *stringLiteralExpr:
		if x.literal != "" {
			return ctx.newBadExpr(cond, "filter comparison to non-empty value: %s", xValue)
		}
		// Either pattern or text should be const, and the
		// non-const one should be varRefExpr
		if xInList, ok = xPattern.(*stringLiteralExpr); ok {
			xVar = xText
		} else if xInList, ok = xText.(*stringLiteralExpr); ok {
			xVar = xPattern
		}
	case *variableRefExpr:
		name := x.name
		if v, ok := xPattern.(*variableRefExpr); ok {
			if xInList, ok = xText.(*stringLiteralExpr); ok && v.name == name {
				xVar = xPattern
				negate = !negate
			}
		}
	}
	if xVar != nil && xInList != nil {
		if _, ok := xVar.(*variableRefExpr); ok {
			return &inExpr{isNot: negate, list: newStringListExpr(strings.Fields(xInList.literal)), expr: xVar}
		}
	}
	return ctx.newBadExpr(cond, "filter arguments are too complex: %s", cond.Dump())
}

func (ctx *parseContext) parseCompareWildcardFuncResult(directive *mkparser.Directive,
	xCall *callExpr, xValue starlarkExpr, negate bool) starlarkExpr {
	if x, ok := xValue.(*stringLiteralExpr); !ok || x.literal != "" {
		return ctx.newBadExpr(directive, "wildcard result can be compared only to empty: %s", xValue)
	}
	callFunc := rtWildcardExists
	if s, ok := xCall.args[0].(*stringLiteralExpr); ok && !strings.ContainsAny(s.literal, "*?{[") {
		callFunc = rtFileExists
	}
	var cc starlarkExpr = &callExpr{name: callFunc, args: xCall.args}
	if !negate {
		cc = &notExpr{cc}
	}
	return cc
}

func (ctx *parseContext) parseCheckFindstringFuncResult(directive *mkparser.Directive,
	xCall *callExpr, xValue starlarkExpr, negate bool) starlarkExpr {
	if x, ok := xValue.(*stringLiteralExpr); !ok || x.literal != "" {
		return ctx.newBadExpr(directive, "findstring result can be compared only to empty: %s", xValue)
	}
	return &eqExpr{
		left:  &callExpr{object: xCall.args[1], name: "find", args: []starlarkExpr{xCall.args[0]}},
		right: &intLiteralExpr{-1},
		isEq:  !negate,
	}
}

func maybeConvertToStringList(expr starlarkExpr) starlarkExpr {
	if xString, ok := expr.(*stringLiteralExpr); ok {
		return newStringListExpr(strings.Fields(xString.literal))
	}
	return expr
}

func (ctx *parseContext) parseCheckIsBoardXxxMacroResult(directive *mkparser.Directive,
	xCall *callExpr, xValue starlarkExpr, negate bool) starlarkExpr {
	if x, ok := xValue.(*stringLiteralExpr); !ok || x.literal != "true" {
		return ctx.newBadExpr(directive, "the result of is-board-xxx can be compared only to 'true'")
	}
	// They have all at least one argument
	if len(xCall.args) < 1 {
		return ctx.newBadExpr(directive, "%s has an argument", xCall.name)
	}
	switch xCall.name {
	case isBoardPlatform:
		xVar, isRef := ctx.newVariableRef(directive, targetBoardPlatformVarName)
		if !isRef {
			return xVar
		}
		return &eqExpr{left: xVar, right: xCall.args[0], isEq: !negate}
	case isVendorBoardPlatform:
		xVar, isRef := ctx.newVariableRef(directive, targetBoardPlatformVarName)
		if !isRef {
			return xVar
		}
		xString, ok := xCall.args[0].(*stringLiteralExpr)
		if !ok {
			return ctx.newBadExpr(directive, "%s argument is not constant", directive.Name)
		}
		xListVar, isRef := ctx.newVariableRef(directive, xString.literal+"_BOARD_PLATFORMS")
		if !isRef {
			return xListVar
		}
		return &inExpr{expr: xVar, list: xListVar, isNot: negate}
	case isBoardPlatformInList:
		xVar, isRef := ctx.newVariableRef(directive, targetBoardPlatformVarName)
		if !isRef {
			return xVar
		}
		return &inExpr{expr: xVar, list: maybeConvertToStringList(xCall.args[0]), isNot: negate}
	case isProductInList:
		xVar, isRef := ctx.newVariableRef(directive, targetProductVarName)
		if !isRef {
			return xVar
		}
		return &inExpr{expr: xVar, list: maybeConvertToStringList(xCall.args[0]), isNot: negate}
	default:
		panic(fmt.Sprintf("unexpected is-board function %s", xCall.name))
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
		case "include", "-include":
			ctx.handleLoadConfig(node, receiver, ctx.parseMakeString(node, x.Args), x.Name[0] != '-')
		default:
			handled = false
		}
	default:
		ctx.errorf(x, receiver, "unsupported line %s", x.Dump())
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

func (ctx *parseContext) wrapBadExpr(xBad *badExpr, receiver nodeReceiver) {
	ctx.insertComment(fmt.Sprintf("# MK2STAR TRANSLATION ERROR: %s", xBad.message), receiver)
	ctx.carryAsComment(xBad.node, receiver)
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

func Convert(mkFile string, reader io.Reader, emitPrint bool, rootDir string) (ss *StarScript, err error) {
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
		topDir:     rootDir,
	}
	ctx := newParseContext(starScript, nodes)
	for ctx.hasNode() && ctx.fatalError == nil {
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
				ctx.errorf(x, starScript, "unexpected directive %s", x.Name)
			}
		default:
			ctx.errorf(x, starScript, "unsupported line")
		}
	}
	if ctx.fatalError != nil {
		return nil, ctx.fatalError
	}
	return starScript, nil
}
