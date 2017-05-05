package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	mkparser "android/soong/androidmk/parser"

	bpparser "github.com/google/blueprint/parser"
)

// TODO: non-expanded variables with expressions

type bpFromMkBuilder struct {
	//bpBuilder      bpparser.SyntaxTree
	bpBuilder            bpparser.Builder
	localAssignments     map[string]*bpparser.Property
	globalAssignments    map[string]*bpparser.Expression
	scope                mkparser.Scope
	module               *bpparser.Module
	latestNode           bpparser.ParseNode
	inModule             bool
	commentsWithinModule map[bpparser.ParseNode](*bpparser.CommentPair)
	pendingComments      [](*bpparser.Comment)
	mkSyntaxTree         *mkparser.ParseTree
}

func newBpFromMkBuilder() *bpFromMkBuilder {
	return &bpFromMkBuilder{
		scope:             androidScope(),
		localAssignments:  make(map[string]*bpparser.Property),
		globalAssignments: make(map[string]*bpparser.Expression),
		bpBuilder:         bpparser.NewBuilder(),
	}

}

func (b *bpFromMkBuilder) addNode(node bpparser.ParseNode) {
	b.dumpPendingComments()
	b.bpBuilder.AddNode(node)
}

func (b *bpFromMkBuilder) appendComment(commentNode *bpparser.Comment) {
	b.bpBuilder.AddNode(commentNode)
}

func (b *bpFromMkBuilder) addModuleComment(module *bpparser.Module, commentNode *bpparser.Comment) {
	b.addNodeComment(module.Type, commentNode)
}

func (b *bpFromMkBuilder) addNodeComment(existingNode bpparser.ParseNode, commentNode *bpparser.Comment) {
	if existingNode == nil {
		panic("Illegal nil value passed for existingNode")
	}
	b.bpBuilder.AppendPostComment(existingNode, commentNode)
}

func (b *bpFromMkBuilder) addPendingComment(commentNode *bpparser.Comment) {
	b.pendingComments = append(b.pendingComments, commentNode)
}

func (b *bpFromMkBuilder) attachUnassociatedCommentsBefore(parseNode bpparser.ParseNode) {
	var pendingComments = b.pullPendingComments()
	b.bpBuilder.AppendPreComments(parseNode, pendingComments)
}

func (b *bpFromMkBuilder) attachUnassociatedCommentsAfter(parseNode bpparser.ParseNode) {
	var pendingComments = b.pullPendingComments()
	b.bpBuilder.AppendPostComments(parseNode, pendingComments)
}

func (b *bpFromMkBuilder) attachUnassociatedCommentsAround(parseNode bpparser.ParseNode) {
	pendingComments := b.pullPendingComments()
	b.bpBuilder.AddCommentsAround(parseNode, pendingComments)
}

func (b *bpFromMkBuilder) dumpPendingComments() {
	var pendingComments = b.pullPendingComments()
	for _, comment := range pendingComments {
		b.appendComment(comment)
	}
}

func (b *bpFromMkBuilder) pullPendingComments() [](*bpparser.Comment) {
	var pendingComments = b.pendingComments
	b.pendingComments = make([](*bpparser.Comment), 0)
	return pendingComments
}

func (b *bpFromMkBuilder) errorf(node mkparser.ParseNode, s string, args ...interface{}) {
	originalText := node.Dump()
	s = fmt.Sprintf(s, args...)
	var newComments = make([](*bpparser.Comment), 0)
	sourcePos := b.mkSyntaxTree.GetSourcePosition(node)
	errorComment := bpparser.NewFullLineComment(fmt.Sprintf(" ANDROIDMK TRANSLATION ERROR at %v:%v : %s", sourcePos.Line, sourcePos.Column, s))
	newComments = append(newComments, bpparser.NewBlankLine(), errorComment)

	lines := strings.Split(originalText, "\n")
	for _, line := range lines {
		if len(line) > 0 {
			line = " " + line
		}
		if strings.HasSuffix(line, " ") {
			line = line[:len(line)-1]
		}
		newComments = append(newComments, bpparser.NewFullLineComment(line))
	}
	newComments = append(newComments, bpparser.NewBlankLine())

	for _, comment := range newComments {
		if b.inModule {
			b.appendComment(comment)
		} else {
			b.addPendingComment(comment)
		}
	}
}

func makeCommentToBlueprint(makeComment *mkparser.Comment) (blueprintComment *bpparser.Comment) {
	switch makeComment.Type {
	case mkparser.FullLineText:
		return bpparser.NewFullLineComment(makeComment.Text)
	case mkparser.FullLineBlank:
		return bpparser.NewBlankLine()
	default:
		panic(fmt.Sprintf("Unrecognized comment %#v of type %#v", makeComment, makeComment.Type))
	}
}

type attemptedConditional interface {
	IsConditional()
	Text() string
}

type conditional struct {
	cond string
	eq   bool
}

func (c *conditional) IsConditional() {
}
func (c *conditional) Text() (text string) {
	return c.cond
}

type malformedConditional struct {
	text string
}

func (m *malformedConditional) IsConditional() {
}
func (m *malformedConditional) Text() (text string) {
	return m.text
}

func main() {
	b, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	output, errs := convertFile(os.Args[1], bytes.NewBuffer(b))
	if len(errs) > 0 {
		for _, err := range errs {
			fmt.Fprintln(os.Stderr, "ERROR: ", err)
		}
		os.Exit(1)
	}

	fmt.Print(output)
}

func convertFile(filePath string, buffer *bytes.Buffer) (result string, errs []error) {
	result, errs, _, _ = convertFileImpl(filePath, buffer)
	return result, errs
}

// The reason convertFileImpl is separate from convertFile is make it easy to obtain debugging information from within a test and still be sure that the test is testing the correct process
// In most cases, the correct function to call is actually convertFile
func convertFileImpl(filePath string, buffer *bytes.Buffer) (result string, errs []error, mkParse *mkparser.ParseTree, bpTree *bpparser.SyntaxTree) {
	succeeded := false

	defer func() {
		if !succeeded {
			os.Stderr.WriteString(fmt.Sprintf("\nandroidmk.go failed to convert %s with %v errors %s\n\n", filePath, len(errs), errs))
		}
	}()

	p := mkparser.NewParser(filePath, buffer)
	p.Strict = true

	mkParse, errs = p.Parse()

	builder := newBpFromMkBuilder()
	builder.mkSyntaxTree = mkParse

	var unclosedIfs []attemptedConditional
	var assignmentCond *conditional

	var allowDanglingComments = false

	for _, node := range mkParse.Nodes {

		sourcePos := mkParse.GetSourcePosition(node)

		switch x := node.(type) {
		case *mkparser.Comment:
			bpComment := makeCommentToBlueprint(x)
			builder.addPendingComment(bpComment)
		case *mkparser.Assignment:
			handleAssignment(builder, x, assignmentCond)
		case *mkparser.ParseError:
			builder.addPendingComment(bpparser.NewFullLineComment(fmt.Sprintf(" ANDROID MK PARSE ERROR at %v:%v: %s", sourcePos.Line, sourcePos.Column, x.Dump())))
		case *mkparser.Directive:
			switch x.Name {
			case "include":
				val := x.Args.Value(builder.scope)
				switch {
				case soongModuleTypes[val]:
					handleModuleConditionals(builder, x, unclosedIfs)
					makeModule(builder, val)
				case val == clear_vars:
					if builder.inModule {
						builder.errorf(x, "cleared module before finishing its definition")
						// TODO if a module is malformed and gets skipped, but also has comments (including errors that generate comments) should we output them anyway?
						allowDanglingComments = true
					}
					resetModule(builder)
				default:
					builder.errorf(x, "unsupported include")
					continue
				}
			case "ifeq", "ifneq", "ifdef", "ifndef":
				args := x.Args.Dump()
				eq := x.Name == "ifeq" || x.Name == "ifdef"
				if _, ok := conditionalTranslations[args]; ok {
					newCond := conditional{args, eq}
					unclosedIfs = append(unclosedIfs, &newCond)
					if builder.inModule {
						if assignmentCond == nil {
							assignmentCond = &newCond
						} else {
							builder.errorf(x, "unsupported nested conditional in module")
						}
					}
				} else {
					builder.errorf(x, "unsupported conditional")
					unclosedIfs = append(unclosedIfs, &malformedConditional{args})
				}
			case "else":
				if len(unclosedIfs) == 0 {
					builder.errorf(x, "missing if before else")
					continue
				} else {
					lastItem := unclosedIfs[len(unclosedIfs)-1]
					_, ok := lastItem.(*conditional)
					if !ok {
						builder.errorf(x, "else from unsupported conditional '%s' ", lastItem.Text())
						continue
					}
				}
			case "endif":
				if len(unclosedIfs) == 0 {
					builder.errorf(x, "missing if before endif")
					continue
				} else {
					lastItem := unclosedIfs[len(unclosedIfs)-1]
					if _, ok := lastItem.(*conditional); ok {
						if assignmentCond == unclosedIfs[len(unclosedIfs)-1] {
							assignmentCond = nil
						}
					} else {
						builder.errorf(x, fmt.Sprintf("endif from unsupported conditional: %s", lastItem.Text()))
					}
					unclosedIfs = unclosedIfs[:len(unclosedIfs)-1]
				}

			default:
				builder.errorf(x, "unsupported directive")
			}
		default:
			builder.errorf(x, "unsupported line")
		}
	}
	builder.dumpPendingComments()
	if builder.inModule {
		allowDanglingComments = true
	}
	if allowDanglingComments {
		// if a module was left unfinished, then we might have given the Blueprint tree builder some unattached comments that are ok to skip
		builder.bpBuilder.AllowDanglingComments()
	}

	bpTree = builder.bpBuilder.BuildAndAttemptToReformat()
	printer := bpparser.NewPrinter(bpTree)
	out := string(printer.PrintTree())

	succeeded = true

	return out, errs, mkParse, bpTree
}

func handleAssignment(file *bpFromMkBuilder, assignment *mkparser.Assignment, c *conditional) {
	if !assignment.Name.Const() {
		file.errorf(assignment, "unsupported non-const variable name")
		return
	}

	if assignment.Target != nil {
		file.errorf(assignment, "unsupported target assignment")
		return
	}

	name := assignment.Name.Value(nil)
	prefix := ""

	if strings.HasPrefix(name, "LOCAL_") {
		for _, x := range propertyPrefixes {
			if strings.HasSuffix(name, "_"+x.mk) {
				name = strings.TrimSuffix(name, "_"+x.mk)
				prefix = x.bp
				break
			}
		}

		if c != nil {
			if prefix != "" {
				file.errorf(assignment, "prefix assignment inside conditional, skipping conditional")
			} else {
				var ok bool
				if prefix, ok = conditionalTranslations[c.cond][c.eq]; !ok {
					panic(fmt.Sprintf("unknown conditional %s found in %s\n", c, assignment))
				}
			}
		}
	} else {
		if c != nil {
			eq := "eq"
			if !c.eq {
				eq = "neq"
			}
			file.errorf(assignment, "conditional %s %s on global assignment", eq, c.cond)
		}
	}

	appendVariable := assignment.Type == "+="

	var err error
	if prop, ok := rewriteProperties[name]; ok {
		err = prop(variableAssignmentContext{file, prefix, assignment.Value, appendVariable})
	} else {
		switch {
		case name == "LOCAL_ARM_MODE":
			// This is a hack to get the LOCAL_ARM_MODE value inside
			// of an arch: { arm: {} } block.
			armModeAssign := assignment
			armModeAssign.Name = mkparser.SimpleMakeString("LOCAL_ARM_MODE_HACK_arm")
			handleAssignment(file, armModeAssign, c)
		case strings.HasPrefix(name, "LOCAL_"):
			file.errorf(assignment, "unsupported assignment to %s", name)
			return
		default:
			var val bpparser.Expression
			val, err = makeVariableToBlueprint(file, assignment.Value, bpparser.ListType)
			if err == nil {
				_, err = setVariable(file, appendVariable, prefix, name, val, false)
			}
		}
	}
	if err != nil {
		file.errorf(assignment, err.Error())
	}
}

func handleModuleConditionals(file *bpFromMkBuilder, directive *mkparser.Directive, conds []attemptedConditional) {
	for _, attemptedConditional := range conds {

		c, isConditional := attemptedConditional.(*conditional)
		if !isConditional {
			continue
		}

		if _, ok := conditionalTranslations[c.cond]; !ok {
			panic("unknown conditional " + c.cond)
		}

		disabledPrefix := conditionalTranslations[c.cond][!c.eq]

		// Create a fake assignment with enabled = false
		val, err := makeVariableToBlueprint(file, mkparser.SimpleMakeString("false"), bpparser.BoolType)
		if err == nil {
			_, err = setVariable(file, false, disabledPrefix, "enabled", val, true)
		}
		if err != nil {
			file.errorf(directive, err.Error())
		}
	}
}

func makeModule(file *bpFromMkBuilder, t string) {
	if !file.inModule {
		// if the input Android.mk file forgot to reset the module, then do it now
		resetModule(file)
	}
	file.module.Type = &bpparser.Token{t}
	file.attachUnassociatedCommentsAfter(file.module.Map.MapBody)
	file.addNode(file.module)

	file.inModule = false
}

func resetModule(file *bpFromMkBuilder) {
	file.dumpPendingComments()
	file.module = &bpparser.Module{}
	file.module.Map = bpparser.NewMap([]*bpparser.Property{})
	file.localAssignments = make(map[string]*bpparser.Property)
	file.inModule = true
}

func makeVariableToBlueprint(file *bpFromMkBuilder, val *mkparser.MakeString,
	typ bpparser.Type) (bpparser.Expression, error) {

	var exp bpparser.Expression
	var err error
	switch typ {
	case bpparser.ListType:
		exp, err = makeToListExpression(val, file.scope)
	case bpparser.StringType:
		exp, err = makeToStringExpression(val, file.scope)
	case bpparser.BoolType:
		exp, err = makeToBoolExpression(val)
	default:
		panic("unknown type")
	}

	// get all the comments in the Makefile that apply to this expression and copy them onto the blueprint expression
	var comments = file.mkSyntaxTree.GetAllComments(val)
	for _, comment := range comments.PreComments() {
		file.bpBuilder.AppendPreComment(exp, bpparser.NewFullLineComment(comment.Text))
	}
	for _, comment := range comments.PostComments() {
		file.bpBuilder.AppendPostComment(exp, bpparser.NewFullLineComment(comment.Text))
	}

	if err != nil {
		return nil, err
	}

	return exp, nil
}

func setVariable(builder *bpFromMkBuilder, plusequals bool, prefix, name string, value bpparser.Expression, local bool) (bpparser.ParseNode, error) {
	if prefix != "" {
		name = prefix + "." + name
	}

	var oldValue *bpparser.Expression
	if local {
		oldProp := builder.localAssignments[name]
		if oldProp != nil {
			oldValue = &oldProp.Value
		}
	} else {
		oldValue = builder.globalAssignments[name]
	}

	if local {
		if oldValue != nil && plusequals {
			newValue, err := addValues(*oldValue, value)
			if err != nil {
				return nil, fmt.Errorf("unsupported addition: %s", err.Error())
			}
			*oldValue = newValue
			var prop = builder.localAssignments[name]

			// adjust the comments slightly and attach them
			comments := builder.pullPendingComments()
			// If a list was constructed with a '+=' operator and the comments were few and started with a newline,
			// then we don't need persist the newline in Blueprint since it's more annoying than relevant
			if len(comments) > 0 && comments[0].Type == bpparser.FullLineBlank && len(comments) <= 2 {
				comments = comments[1:]
			}
			// If there's only one comment, then make it into a slash-star comment that can be less intrusive
			if len(comments) == 1 {
				for _, comment := range comments {
					if comment.Type == bpparser.FullLineText {
						comment.Type = bpparser.InlineText
						comment.Text += " "
					}
				}
			}
			// put comments in front of the value
			builder.bpBuilder.AppendPreComments(value, comments)
			builder.latestNode = prop
			return prop, nil
		} else {
			names := strings.Split(name, ".")
			if !builder.inModule {
				// the input Android.mk file forgot to start the module via 'include $(CLEAR_VARS)' ; start the module now
				resetModule(builder)
			}
			var module = builder.module
			var container = &module.Properties

			for i, n := range names[:len(names)-1] {
				fqn := strings.Join(names[0:i+1], ".")
				prop := builder.localAssignments[fqn]
				if prop == nil {
					prop = &bpparser.Property{
						Name:  n,
						Value: bpparser.NewMap([]*bpparser.Property{}),
					}
					builder.localAssignments[fqn] = prop
					*container = append(*container, prop)
				}
				container = &prop.Value.(*bpparser.Map).Properties
			}

			prop := &bpparser.Property{
				Name:  names[len(names)-1],
				Value: value,
			}
			builder.localAssignments[name] = prop
			*container = append(*container, prop)
			builder.attachUnassociatedCommentsBefore(prop)
			builder.latestNode = prop
			// if any comments were put on the value, then move them to the property line instead
			builder.bpBuilder.MoveComments(value, prop)
			return prop, nil
		}
	} else {
		var a *bpparser.Assignment
		if oldValue != nil && plusequals {
			a = bpparser.NewAssignment(name, value, value, "+=", false)
			builder.addNode(a)
		} else {
			a = bpparser.NewAssignment(name, value, value, "=", false)
			builder.globalAssignments[name] = &a.Value
			builder.addNode(a)
		}
		builder.attachUnassociatedCommentsBefore(a)
		return a, nil
	}
}
