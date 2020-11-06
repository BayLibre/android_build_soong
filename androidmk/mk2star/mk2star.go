package mk2star

import (
	mkparser "android/soong/androidmk/parser"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/scanner"
)

type ErrorMonitorCB interface {
	NewError(s string, node mkparser.Node)
}

type emptyErrorMonitor struct{}

func (_ emptyErrorMonitor) NewError(_ string, _ mkparser.Node) {}

var StarlarkSuffix = ".star"
var ErrorMonitor ErrorMonitorCB = emptyErrorMonitor{}

// Derives module name for a given file. It is base name
// (file name without suffix), with some characters replaced to make it a Starlark ideintifier
func moduleNameForFile(mkFile string) string {
	base := strings.TrimSuffix(filepath.Base(mkFile), filepath.Ext(mkFile))
	// TODO(asmundak): what else can be in the product file names?
	return strings.ReplaceAll(base, "-", "_")
}

type sink struct {
	buf strings.Builder
}

func (sink *sink) write(ss ...string) {
	for _, s := range ss {
		sink.buf.WriteString(s)
	}
}

func (sink *sink) writeln(ss ...string) {
	sink.write(ss...)
	sink.write("\n")
}

func (sink *sink) emitLoadInit() {
	sink.writeln(`load("//build/make/target/product:product_config`, StarlarkSuffix, `", "prodconf")`)
}

type productConfigVariables struct {
	vars map[string]string
}

func (pcv *productConfigVariables) newVariable(name string, flavor string) error {
	if oldflavor, ok := pcv.vars[name]; ok && oldflavor != flavor {
		return fmt.Errorf("redefining %s as %s (was %s)", name, flavor, oldflavor)
	}
	pcv.vars[name] = flavor
	return nil
}

// All known product variables.
var ConfigVariables = productConfigVariables{make(map[string]string)}

// Types used to keep processed makefile data:
type comment struct {
	mkPos scanner.Position
	text  string
}

type subConfig struct {
	mkPos        scanner.Position
	path         string
	originalPath string
	moduleName   string
	loadAlways   bool
}

type varAssignment struct {
	mkPos    scanner.Position
	name     string
	value    *mkparser.MakeString
	isAppend bool
	isFirst  bool
}

func (asgn varAssignment) referenceName() string {
	return fmt.Sprintf("_vars[\"%s\"]", asgn.name)
}

type StarScript struct {
	mkFile            string
	moduleName        string
	mkPos             scanner.Position
	comments          []comment
	subconfigs        []subConfig
	assignments       []varAssignment
	ifLevel           int
	hasErrors         bool
	varLastAssignment map[string]int // Maps variable name to its last assignment index
}

func (ss *StarScript) insertComment(commentText string) {
	ss.comments = append(ss.comments, comment{ss.mkPos, strings.TrimSpace(commentText)})
}

func (ss *StarScript) carryAsComment(failedNode mkparser.Node) {
	for _, line := range strings.Split(failedNode.Dump(), "\n") {
		ss.insertComment("# " + line)
	}
}

// records that the given node failed to be converted and includes an explanatory message
func (ss *StarScript) errorf(failedNode mkparser.Node, message string, args ...interface{}) {
	ErrorMonitor.NewError(message, failedNode)
	message = fmt.Sprintf(message, args...)
	ss.insertComment(fmt.Sprintf("# MK2STAR TRANSLATION ERROR: %s", message))
	ss.carryAsComment(failedNode)
	ss.hasErrors = true
}

func (ss *StarScript) setMkPos(pos, end scanner.Position) {
	// It is unusual but not forbidden for pos.Line to be smaller than f.mkPos.Line
	// For example:
	//
	// if true                       # this line is emitted 1st
	// if true                       # this line is emitted 2nd
	// some-target: some-file        # this line is emitted 3rd
	//         echo doing something  # this recipe is emitted 6th
	// endif #some comment           # this endif is emitted 4th; this comment is part of the recipe
	//         echo doing more stuff # this is part of the recipe
	// endif                         # this endif is emitted 5th
	//
	// However, if pos.Line < f.mkPos.Line, we treat it as though it were equal
	if pos.Line >= ss.mkPos.Line {
		ss.mkPos = end
	}
}

func (ss *StarScript) path2BazelRef(path string) string {
	relPath, err := filepath.Rel(filepath.Dir(ss.mkFile), path)
	if err != nil || relPath[0] == '.' {
		return "//" + filepath.Dir(path) + ":" + filepath.Base(path)
	}
	return ":" + filepath.Base(path)
}

func (ss StarScript) emitConfigVariableArguments(sink *sink) {
	// Sort by variable name to facilitate testing
	var sorted []string
	for name := range ss.varLastAssignment {
		sorted = append(sorted, name)
	}
	sort.Strings(sorted)
	for _, name := range sorted {
		asgn := ss.assignments[ss.varLastAssignment[name]]
		sink.writeln("    ", asgn.name, " = ", asgn.referenceName(), ",")
	}
}

func (ss *StarScript) emitSubConfigsArgument(sink *sink) {
	sink.write("    [")
	sep := ""
	for _, sc := range ss.subconfigs {
		if sc.loadAlways {
			sink.write(sep, sc.moduleName)
			sep = ", "
		}
	}
	sink.writeln("],")
}

func (ss *StarScript) emitModuleDeclaration(sink *sink) {
	sink.writeln(ss.moduleName, ` = prodconf(`)
	sink.writeln(`    "`, ss.moduleName, `",`)
	ss.emitSubConfigsArgument(sink)
	ss.emitConfigVariableArguments(sink)
	sink.writeln(")")
}

func (ss *StarScript) emitAssignment(sink *sink, asgn varAssignment) {
	flavor, ok := ConfigVariables.vars[asgn.name]
	if !ok {
		panic(fmt.Errorf("unknown variable %s", asgn.name))
	}
	sink.write(asgn.referenceName())
	value := strings.TrimSpace(asgn.value.Dump())
	if flavor == "item" {
		if asgn.isAppend && !asgn.isFirst {
			sink.writeln(` += `, asgn.referenceName(), ` + " " + "`, value, `"`)
		} else {
			sink.writeln(` = "`, value, `"`)
		}
	} else {
		sink.write(` = `)
		if asgn.isAppend && !asgn.isFirst {
			sink.write(asgn.referenceName(), ` + `)
		}
		items := []string{}
		chunks := strings.Split(value, " ")
		for _, item := range chunks {
			if item != "" {
				items = append(items, item)
			}
		}
		if len(items) == 1 {
			sink.writeln(`["`, items[0], `"]`)
		} else {
			sink.writeln("[")
			for _, item := range items {
				sink.writeln(`    "`, item, `",`)
			}
			sink.writeln("]")
		}
	}
}

func (ss StarScript) String() string {
	var sink sink
	iComment, iSubConfig, iAssign := 0, 0, 0
	var offComment, offSubConfig, offAssign int
	loadInitEmitted := false
	wroteComment := false
	wroteStatement := false
	wroteVarsDict := false
	for iComment < len(ss.comments) || iSubConfig < len(ss.subconfigs) || iAssign < len(ss.assignments) {
		if iComment < len(ss.comments) {
			offComment = ss.comments[iComment].mkPos.Offset
		} else {
			offComment = math.MaxInt32
		}
		if iSubConfig < len(ss.subconfigs) {
			offSubConfig = ss.subconfigs[iSubConfig].mkPos.Offset
		} else {
			offSubConfig = math.MaxInt32
		}
		if iAssign < len(ss.assignments) {
			offAssign = ss.assignments[iAssign].mkPos.Offset
		} else {
			offAssign = math.MaxInt32
		}
		if offComment < offSubConfig && offComment < offAssign {
			if wroteStatement {
				sink.writeln()
				wroteStatement = false
			}
			sink.writeln(ss.comments[iComment].text)
			wroteComment = true
			iComment++
		} else if offSubConfig < offComment && offSubConfig < offAssign {
			// print subconfig
			if wroteComment {
				sink.writeln()
				wroteComment = false
			}
			if !loadInitEmitted {
				sink.emitLoadInit()
				loadInitEmitted = true
			}
			sc := ss.subconfigs[iSubConfig]
			if sc.loadAlways {
				sink.writeln(`load("`, ss.path2BazelRef(sc.path), `", "`, sc.moduleName, `")`)
			} else {
				sink.writeln("# MK2STAR TRANSLATION ERROR: conditional load not supported")
				sink.writeln(`# conditional_load("`, ss.path2BazelRef(sc.path), `", "`, sc.moduleName, `")`)
			}
			wroteStatement = true
			iSubConfig++
		} else {
			if wroteComment {
				wroteComment = false
			}
			if !wroteVarsDict {
				sink.writeln("_vars = dict()")
				wroteVarsDict = true
			}
			ss.emitAssignment(&sink, ss.assignments[iAssign])
			wroteStatement = true
			iAssign++
		}
	}
	if !loadInitEmitted {
		sink.emitLoadInit()
	}
	ss.emitModuleDeclaration(&sink)
	if ss.hasErrors {
		sink.writeln()
		sink.writeln(`print("The conversion of `, ss.mkFile,
			` to this script was only partially successful. `,
			`Please fix the problems by inspecting commented out lines `,
			`and then remove this print statement")`)
	}
	return sink.buf.String()
}

//goland:noinspection SpellCheckingInspection
func (ss *StarScript) handleAssignment(a *mkparser.Assignment) {
	// Handle only simple variables
	if len(a.Name.Strings) > 1 {
		ss.errorf(a, "Only simple variables are handled")
		return
	}
	name := a.Name.Strings[0]
	if _, ok := ConfigVariables.vars[name]; !ok {
		ss.errorf(a, fmt.Sprintf("%s is not listed as product configuration variable", name))
		return
	}

	// TODO(asmundak); verify that the value does not contain nasty constructs like
	//   PRODUCT_XXX = $(call foo, bar)

	_, isNotFirst := ss.varLastAssignment[name]
	asgn := varAssignment{
		mkPos: ss.mkPos, name: name, value: a.Value, isAppend: a.Type == "+=", isFirst: !isNotFirst}
	ss.assignments = append(ss.assignments, asgn)
	ss.varLastAssignment[name] = len(ss.assignments) - 1
}

func (ss *StarScript) handleVariable(v *mkparser.Variable) {
	// The result of parsing
	//   $(call inherit-product, foo)
	// is a variable (its first token is 'call inherit-product ')
	name := v.Name
	first := strings.TrimSpace(name.Strings[0])
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
			ss.errorf(v,
				fmt.Sprintf("cannot convert calling '%s'",
					strings.TrimSpace(strings.SplitN(strings.TrimPrefix(first, "call "), ",", 2)[0])))
		} else {
			ss.errorf(v, "unsupported variable")
		}
		return
	}
	var path string
	if len(name.Strings) == 1 {
		path = first
	} else if first != "" || len(name.Strings) != 2 || len(name.Variables) != 1 {
		ss.errorf(v, "unsupported line")
		return
	} else {
		varName := name.Variables[0].Name
		if len(varName.Strings) != 1 || varName.Strings[0] != "SRC_TARGET_DIR" {
			ss.errorf(v, "unsupported line")
			return
		}
		path = filepath.Join("build", "make", "target", strings.TrimSpace(name.Strings[1]))
	}
	suffix := filepath.Ext(path)
	starPath := strings.TrimSuffix(path, suffix) + StarlarkSuffix
	ss.subconfigs = append(ss.subconfigs, subConfig{
		mkPos:        ss.mkPos,
		path:         starPath,
		originalPath: path,
		moduleName:   moduleNameForFile(path),
		loadAlways:   loadAlways,
	})
}

func (ss *StarScript) SubConfigFiles() []string {
	var subs []string
	for _, sc := range ss.subconfigs {
		subs = append(subs, sc.originalPath)
	}
	return subs
}

func (ss *StarScript) HasErrors() bool {
	return ss.hasErrors
}

func Convert(mkFile string, reader io.Reader) (*StarScript, error) {
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
	out := StarScript{moduleName: moduleNameForFile(mkFile), mkFile: mkFile, varLastAssignment: make(map[string]int)}
	for _, node := range nodes {
		out.setMkPos(parser.Unpack(node.Pos()), parser.Unpack(node.End()))
		// We do not support if blocks at all
		if out.ifLevel > 0 {
			switch x := node.(type) {
			case *mkparser.Comment:
				out.insertComment("# " + x.Comment)
			case *mkparser.Directive:
				out.carryAsComment(node)
				switch x.Name {
				case "ifeq", "ifneq", "ifdef", "ifndef":
					out.ifLevel++
				case "endif":
					out.ifLevel--
				default:
				}
			default:
				out.carryAsComment(node)
			}
			continue
		}
		switch x := node.(type) {
		case *mkparser.Comment:
			out.insertComment("# " + x.Comment)
		case *mkparser.Assignment:
			out.handleAssignment(x)
		case *mkparser.Variable:
			out.handleVariable(x)
		case *mkparser.Directive:
			switch x.Name {
			case "ifeq", "ifneq", "ifdef", "ifndef":
				out.ifLevel++
				out.errorf(x, "conditionals are not supported")
			case "endif":
				out.ifLevel--
			default:
				out.errorf(x, "unsupported directive "+x.Name)
				continue
			}
		default:
			out.errorf(x, "unsupported line")
		}
	}
	return &out, nil
}
