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

type bpFileBuilder struct {
	bpSyntaxTree      bpparser.SyntaxTree
	localAssignments  map[string]*bpparser.Property
	globalAssignments map[string]*bpparser.Expression
	scope             mkparser.Scope
	module            *bpparser.Module
	latestNode        bpparser.ParseNode
	inModule          bool
	pendingComments   [](*bpparser.Comment)
	mkSyntaxTree      mkparser.SyntaxTree
}

func newBpFile() *bpFileBuilder {
	return &bpFileBuilder{
		scope:             androidScope(),
		localAssignments:  make(map[string]*bpparser.Property),
		globalAssignments: make(map[string]*bpparser.Expression),
		bpSyntaxTree:      *(bpparser.NewSyntaxTree()),
	}

}

func (f *bpFileBuilder) addNode(node bpparser.ParseNode) {
	//f.latestNode = node
	f.bpSyntaxTree.AddNode(node)
	//f.handlePendingComments()
}

func (f *bpFileBuilder) appendComment(commentNode *bpparser.Comment) {
	f.bpSyntaxTree.AddNode(commentNode)
	//if f.inModule {
	//	f.addModuleComment(f.module, commentNode)
	//} else {
	//	if f.latestNode == nil {
	//		f.addPendingComment(commentNode)
	//	} else {
	//		f.addNodeComment(f.latestNode, commentNode)
	//	}
	//}
}

func (f *bpFileBuilder) addModuleComment(module *bpparser.Module, commentNode *bpparser.Comment) {
	f.addNodeComment(module.Type, commentNode)
}

func (f *bpFileBuilder) addNodeComment(existingNode bpparser.ParseNode, commentNode *bpparser.Comment) {
	if existingNode == nil {
		panic("Illegal nil value passed for existingNode")
	}
	var commentContainer = f.bpSyntaxTree.GetComments(existingNode)
	fmt.Printf("androidmk adding node comment : %v after %#v (%p)\n", commentNode, existingNode, existingNode)
	commentContainer.AddPostComment(*commentNode)
	var commentCount = len(f.bpSyntaxTree.GetComments(existingNode).PostComments())
	if commentCount < 1 {
		panic("failed to add comment")
	}
}

func (f *bpFileBuilder) addPendingComment(commentNode *bpparser.Comment) {
	fmt.Println("androidmk adding 1 pending comment: ", commentNode)
	f.pendingComments = append(f.pendingComments, commentNode)
}

func (f *bpFileBuilder) attachUnassociatedCommentsBefore(parseNode bpparser.ParseNode) {
	var pendingComments = f.pullPendingComments()
	fmt.Println("androidmk.go attaching ", len(pendingComments), " comments before ", parseNode)
	for _, comment := range pendingComments {
		f.bpSyntaxTree.GetComments(parseNode).AddPreComment(*comment)
	}
}

func (f *bpFileBuilder) attachUnassociatedCommentsAfter(parseNode bpparser.ParseNode) {
	var pendingComments = f.pullPendingComments()
	fmt.Println("androidmk.go attaching ", len(pendingComments), " comments before ", parseNode)
	for _, comment := range pendingComments {
		f.bpSyntaxTree.GetComments(parseNode).AddPostComment(*comment)
	}
}

func (f *bpFileBuilder) dumpPendingComments() {
	var pendingComments = f.pullPendingComments()
	fmt.Println("androidmk.go flushing ", len(pendingComments), " comments")
	for _, comment := range pendingComments {
		f.appendComment(comment)
	}
}

func (f *bpFileBuilder) pullPendingComments() [](*bpparser.Comment) {
	var pendingComments = f.pendingComments
	f.pendingComments = make([](*bpparser.Comment), 0)
	return pendingComments
}

func (f *bpFileBuilder) errorf(node mkparser.ParseNode, s string, args ...interface{}) {
	orig := node.Dump()
	s = fmt.Sprintf(s, args...)
	f.appendComment(bpparser.NewFullLineComment(fmt.Sprintf(" ANDROIDMK TRANSLATION ERROR: %s", s)))

	lines := strings.Split(orig, "\n")
	for _, line := range lines {
		f.appendComment(bpparser.NewFullLineComment(line))
	}
}

type conditional struct {
	cond string
	eq   bool
}

func main() {
	b, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		fmt.Println(err.Error())
		return
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

func convertFile(filename string, buffer *bytes.Buffer) (string, []error) {
	fmt.Println(fmt.Sprint("converting file ", filename))
	p := mkparser.NewParser(filename, buffer)
	p.Strict = true

	fmt.Println("starting to parse android mk file ****************************************************")
	mkParse, errs := p.Parse()
	fmt.Println("done parsing android mk file ****************************************************")
	if len(errs) > 0 {
		return "", errs
	}

	file := newBpFile()
	file.mkSyntaxTree = mkParse

	var unclosedIfs []*conditional
	var assignmentCond *conditional

	for _, node := range mkParse.Nodes {
		//file.appendComment(bpparser.NewFullLineComment(fmt.Sprint("line = ", i)))

		fmt.Println(fmt.Sprint("androidmk parsing line ", node.Dump()))
		switch x := node.(type) {
		case *mkparser.Comment:
			//if file.inModule {
			//	fmt.Println("adding module comment ", x.Text, " to ", file.module)
			//	file.addModuleComment(file.module, bpparser.NewFullLineComment(x.Text))
			//} else {
			//	file.appendComment(bpparser.NewFullLineComment(x.Text))
			//}
			var newComment = bpparser.NewFullLineComment(x.Text)
			file.addPendingComment(newComment)
		case *mkparser.Assignment:
			handleAssignment(file, x, assignmentCond)
		case *mkparser.Directive:
			switch x.Name {
			case "include":
				val := x.Args.Value(file.scope)
				switch {
				case soongModuleTypes[val]:
					handleModuleConditionals(file, x, unclosedIfs)
					makeModule(file, val)
				case val == clear_vars:
					fmt.Println("resetting module")
					resetModule(file)
				default:
					file.errorf(x, "unsupported include")
					continue
				}
			case "ifeq", "ifneq", "ifdef", "ifndef":
				args := x.Args.Dump()
				eq := x.Name == "ifeq" || x.Name == "ifdef"
				if _, ok := conditionalTranslations[args]; ok {
					newCond := conditional{args, eq}
					unclosedIfs = append(unclosedIfs, &newCond)
					if file.inModule {
						if assignmentCond == nil {
							assignmentCond = &newCond
						} else {
							file.errorf(x, "unsupported nested conditional in module")
						}
					}
				} else {
					file.errorf(x, "unsupported conditional")
					unclosedIfs = append(unclosedIfs, nil)
					continue
				}
			case "else":
				if len(unclosedIfs) == 0 {
					file.errorf(x, "missing if before else")
					continue
				} else if unclosedIfs[len(unclosedIfs)-1] == nil {
					file.errorf(x, "else from unsupported contitional")
					continue
				}
				unclosedIfs[len(unclosedIfs)-1].eq = !unclosedIfs[len(unclosedIfs)-1].eq
			case "endif":
				if len(unclosedIfs) == 0 {
					file.errorf(x, "missing if before endif")
					continue
				} else if unclosedIfs[len(unclosedIfs)-1] == nil {
					file.errorf(x, "endif from unsupported contitional")
					unclosedIfs = unclosedIfs[:len(unclosedIfs)-1]
				} else {
					if assignmentCond == unclosedIfs[len(unclosedIfs)-1] {
						assignmentCond = nil
					}
					unclosedIfs = unclosedIfs[:len(unclosedIfs)-1]
				}
			default:
				file.errorf(x, "unsupported directive")
				continue
			}
		default:
			file.errorf(x, "unsupported line")
		}
	}
	out := bpparser.PrintTree(&file.bpSyntaxTree)

	return string(out), nil
}

func handleAssignment(file *bpFileBuilder, assignment *mkparser.Assignment, c *conditional) {
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
					panic("unknown conditional")
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
				err = setVariable(file, appendVariable, prefix, name, val, false)
			}
		}
	}
	if err != nil {
		file.errorf(assignment, err.Error())
	}
}

func handleModuleConditionals(file *bpFileBuilder, directive *mkparser.Directive, conds []*conditional) {
	for _, c := range conds {
		if c == nil {
			continue
		}

		if _, ok := conditionalTranslations[c.cond]; !ok {
			panic("unknown conditional " + c.cond)
		}

		disabledPrefix := conditionalTranslations[c.cond][!c.eq]

		// Create a fake assignment with enabled = false
		val, err := makeVariableToBlueprint(file, mkparser.SimpleMakeString("false"), bpparser.BoolType)
		if err == nil {
			err = setVariable(file, false, disabledPrefix, "enabled", val, true)
		}
		if err != nil {
			file.errorf(directive, err.Error())
		}
	}
}

func makeModule(file *bpFileBuilder, t string) {
	fmt.Println("Making module", t)
	file.module.Type = &bpparser.LeafNode{t}
	file.addNode(file.module)

	//file.appendComment(bpparser.NewFullLineComment("sample"))

	//var appendedComments = file.bpSyntaxTree.GetComments(file.module.Type)
	//var numAppendedComments = len(appendedComments.PostComments())
	//if numAppendedComments < 1 {
	//	fmt.Println("tree = ", file.bpSyntaxTree)
	//	fmt.Println("addr of latest node is ", &file.latestNode)
	//	panic(fmt.Sprint("appended ", numAppendedComments, " comments"))
	//}

	//fmt.Println("mini output ", bpparser.PrintTree(&file.bpSyntaxTree))

	//file.latestNode = nil

	file.inModule = false
	fmt.Println("Made module", t)
}

func resetModule(file *bpFileBuilder) {
	file.dumpPendingComments()
	file.module = &bpparser.Module{}
	file.localAssignments = make(map[string]*bpparser.Property)
	file.inModule = true
}

func makeVariableToBlueprint(file *bpFileBuilder, val *mkparser.MakeString,
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
	for _, comment := range comments {
		file.bpSyntaxTree.GetComments(exp).AddPostComment(*bpparser.NewFullLineComment(comment.Text))
	}

	if err != nil {
		return nil, err
	}

	return exp, nil
}

func setVariable(file *bpFileBuilder, plusequals bool, prefix, name string, value bpparser.Expression, local bool) error {
	//file.handlePendingComments()
	if prefix != "" {
		name = prefix + "." + name
	}

	var oldValue *bpparser.Expression
	if local {
		oldProp := file.localAssignments[name]
		if oldProp != nil {
			oldValue = &oldProp.Value
		}
	} else {
		oldValue = file.globalAssignments[name]
	}

	if local {
		if oldValue != nil && plusequals {
			newValue, err := addValues(*oldValue, value)
			if err != nil {
				return fmt.Errorf("unsupported addition: %s", err.Error())
			}
			//file.bpSyntaxTree.MoveComments(*oldValue, newValue)
			file.latestNode = newValue
			*oldValue = newValue
			var prop = file.localAssignments[name]
			file.attachUnassociatedCommentsAfter(prop)
		} else {
			names := strings.Split(name, ".")
			container := &file.module.Properties

			for i, n := range names[:len(names)-1] {
				fqn := strings.Join(names[0:i+1], ".")
				prop := file.localAssignments[fqn]
				if prop == nil {
					prop = &bpparser.Property{
						Name: n,
						Value: &bpparser.Map{
							Properties: []*bpparser.Property{},
						},
					}
					file.localAssignments[fqn] = prop
					*container = append(*container, prop)
				}
				container = &prop.Value.(*bpparser.Map).Properties
			}

			prop := &bpparser.Property{
				Name:  names[len(names)-1],
				Value: value,
			}
			file.localAssignments[name] = prop
			*container = append(*container, prop)
			file.attachUnassociatedCommentsBefore(prop)
		}
	} else {
		var a *bpparser.Assignment
		if oldValue != nil && plusequals {
			a = bpparser.NewAssignment(name, value, value, "+=", false)
			file.addNode(a)
		} else {
			a = bpparser.NewAssignment(name, value, value, "=", false)
			file.globalAssignments[name] = &a.Value
			file.addNode(a)
		}
		file.attachUnassociatedCommentsAfter(value)
	}
	return nil
}
