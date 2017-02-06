package parser

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"text/scanner"
)

var errTooManyErrors = errors.New("too many errors")

const maxErrors = 100

type ParseError struct {
	Err error
	Pos scanner.Position
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("%s: %s", e.Pos, e.Err)
}

//func (p *parser) appendComment(commentNode *Comment) {
//	p.addPendingComment(commentNode)
//}

type parser struct {
	scanner    scanner.Scanner
	tok        rune
	errors     []error
	syntaxTree SyntaxTree
	//currentNode     ParseNode
	pendingComments [](*Comment)
	Strict          bool
}

func (p *parser) addNodeComment(existingNode ParseNode, commentNode *Comment) {
	if existingNode == nil {
		panic("Illegal nil value passed for existingNode")
	}
	//if existingNode == commentNode {
	//	panic("a comment cannot follow itself")
	//}
	var commentContainer = p.syntaxTree.getComments(existingNode)
	fmt.Printf("androidmk parser.go adding node comment : %v at %p after %#v (%p)\n", *commentNode, *commentNode, existingNode, existingNode)
	commentContainer.addPostComment(*commentNode)
	if p.syntaxTree.getComments(existingNode).PostComments() < 1 {
		panic("failed to add comment")
	}
}

func (p *parser) addPendingComment(commentNode *Comment) {
	fmt.Println("androidmk/parser adding 1 pending comment: ", commentNode)
	p.pendingComments = append(p.pendingComments, commentNode)
}

func (p *parser) attachUnassociatedCommentsTo(parseNode ParseNode) {
	var pendingComments = p.pendingComments
	if len(pendingComments) > 0 {
		var commentContainer = p.syntaxTree.getComments(parseNode)
		fmt.Println("parser.go flushing ", len(pendingComments), " comments to attach to ", parseNode)

		for _, comment := range pendingComments {
			fmt.Printf("flushing comment '%v', adding before %#v\n", comment.Dump(), parseNode)
			commentContainer.addPreComment(*comment)
		}
		if len(p.syntaxTree.getComments(parseNode).preComments) < 1 {
			panic("failed to add pending comments")
		}
		fmt.Println("done flushing comments")
	}
}

func (p *parser) dumpUnassociatedComments() {
	var pendingComments = p.pendingComments
	if len(pendingComments) > 0 {
		fmt.Println("android/parser dumping ", len(pendingComments), " comments")
		p.pendingComments = make([](*Comment), 0)
		for _, comment := range pendingComments {
			p.addNode(comment)
		}
	}
}

func (p *parser) addNode(node ParseNode) {
	p.dumpUnassociatedComments()
	p.syntaxTree.addNode(node)
	//p.currentNode = node
	//fmt.Printf("saved currentNode as %#v", node)
	//panic("stack")
	//p.attachUnassociatedCommentsTo(node)
}

func (p *parser) Parse() (SyntaxTree, []error) {
	if !p.Strict {
		defer func() {
			if r := recover(); r != nil {
				if r == errTooManyErrors {
					return
				}
				panic(r)
			}
		}()
	}
	p.parseLines()
	p.accept(scanner.EOF)

	return p.syntaxTree, p.errors
}

func NewSyntaxTree() (tree SyntaxTree) {
	tree.comments = map[ParseNode]*CommentPair{}
	return tree
}

func NewParser(filename string, r io.Reader) *parser {
	p := &parser{}
	p.syntaxTree = NewSyntaxTree()
	p.scanner.Init(r)
	p.scanner.Error = func(sc *scanner.Scanner, msg string) {
		p.errorf(msg)
	}
	p.scanner.Whitespace = 0
	p.scanner.IsIdentRune = func(ch rune, i int) bool {
		return ch > 0 && ch != ':' && ch != '#' && ch != '=' && ch != '+' && ch != '$' &&
			ch != '\\' && ch != '(' && ch != ')' && ch != '{' && ch != '}' && ch != ';' &&
			ch != '|' && ch != '?' && ch != '\r' && !isWhitespace(ch)
	}
	p.scanner.Mode = scanner.ScanIdents
	p.scanner.Filename = filename
	p.next()
	return p
}

func (p *parser) errorf(format string, args ...interface{}) {
	err := &ParseError{
		Err: fmt.Errorf(format, args...),
		Pos: p.scanner.Position,
	}
	p.errors = append(p.errors, err)
	if len(p.errors) >= maxErrors {
		panic(errTooManyErrors)
	}
}

func (p *parser) accept(toks ...rune) bool {
	for _, tok := range toks {
		if p.tok != tok {
			p.errorf("expected %s, found %s", scanner.TokenString(tok),
				scanner.TokenString(p.tok))
			return false
		}
		p.next()
	}
	return true
}

func (p *parser) next() {
	if p.tok != scanner.EOF {
		p.tok = p.scanner.Scan()
		for p.tok == '\r' {
			p.tok = p.scanner.Scan()
		}
	}
}

func (p *parser) parseLines() {
	for {
		p.ignoreWhitespace()

		if p.parseDirective() {
			continue
		}

		ident := p.parseExpression('=', '?', ':', '#', '\n')

		p.ignoreSpaces()

		switch p.tok {
		case '?':
			p.accept('?')
			if p.tok == '=' {
				p.parseAssignment("?=", nil, ident)
			} else {
				p.errorf("expected = after ?")
			}
		case '+':
			p.accept('+')
			if p.tok == '=' {
				p.parseAssignment("+=", nil, ident)
			} else {
				p.errorf("expected = after +")
			}
		case ':':
			p.accept(':')
			switch p.tok {
			case '=':
				p.parseAssignment(":=", nil, ident)
			default:
				p.parseRule(ident)
			}
		case '=':
			p.parseAssignment("=", nil, ident)
		case '#', '\n', scanner.EOF:
			ident.TrimRightSpaces()
			if v, ok := toVariable(ident); ok {
				p.syntaxTree.addNode(v)
			} else if !ident.Empty() {
				p.errorf("expected directive, rule, or assignment after ident " + ident.Dump())
			}
			switch p.tok {
			case scanner.EOF:
				return
			case '\n':
				p.accept('\n')
			case '#':
				var comment = p.parseComment()
				p.addPendingComment(comment)

			}
		default:
			p.errorf("expected assignment or rule definition, found %s\n",
				p.scanner.TokenText())
			return
		}
	}
	p.dumpUnassociatedComments()
}

func (p *parser) parseDirective() bool {
	if p.tok != scanner.Ident || !isDirective(p.scanner.TokenText()) {
		return false
	}

	d := p.scanner.TokenText()
	p.accept(scanner.Ident)

	expression := SimpleMakeString("")

	switch d {
	case "endif", "endef", "else":
		// Nothing
	case "define":
		expression = p.parseDefine()
	default:
		p.ignoreSpaces()
		expression = p.parseExpression()
	}

	if d == "include" && expression.Dump() == "$(CLEAR_VARS)" {
		p.dumpUnassociatedComments()
	}

	p.syntaxTree.addNode(&Directive{
		Name: d,
		Args: expression,
	})
	fmt.Printf("parsed directive %#v\n", expression.Dump())
	return true
}

func (p *parser) parseDefine() *MakeString {
	value := SimpleMakeString("")

loop:
	for {
		switch p.tok {
		case scanner.Ident:
			value.appendString(p.scanner.TokenText())
			if p.scanner.TokenText() == "endef" {
				p.accept(scanner.Ident)
				break loop
			}
			p.accept(scanner.Ident)
		case '\\':
			p.parseEscape()
			switch p.tok {
			case '\n':
				value.appendString(" ")
			case scanner.EOF:
				p.errorf("expected escaped character, found %s",
					scanner.TokenString(p.tok))
				break loop
			default:
				value.appendString(`\` + string(p.tok))
			}
			p.accept(p.tok)
		//TODO: handle variables inside defines?  result depends if
		//define is used in make or rule context
		//case '$':
		//	variable := p.parseVariable()
		//	value.appendVariable(variable)
		case scanner.EOF:
			p.errorf("unexpected EOF while looking for endef")
			break loop
		default:
			value.appendString(p.scanner.TokenText())
			p.accept(p.tok)
		}
	}

	return value
}

func (p *parser) parseEscape() {
	p.scanner.Mode = 0
	p.accept('\\')
	p.scanner.Mode = scanner.ScanIdents
}

func (p *parser) parseExpression(end ...rune) *MakeString {
	value := SimpleMakeString("")

	endParen := false
	for _, r := range end {
		if r == ')' {
			endParen = true
		}
	}
	parens := 0

loop:
	for {
		if endParen && parens > 0 && p.tok == ')' {
			parens--
			value.appendString(")")
			p.accept(')')
			continue
		}

		for _, r := range end {
			if p.tok == r {
				break loop
			}
		}

		switch p.tok {
		case '\n':
			break loop
		case scanner.Ident:
			value.appendString(p.scanner.TokenText())
			p.accept(scanner.Ident)
		case '\\':
			p.parseEscape()
			switch p.tok {
			case '\n':
				value.appendString(" ")
			case scanner.EOF:
				p.errorf("expected escaped character, found %s",
					scanner.TokenString(p.tok))
				return value
			default:
				value.appendString(`\` + string(p.tok))
			}
			p.accept(p.tok)
		case '#':
			var comment = p.parseComment()
			comment.ApplicableTo = value
			p.addNodeComment(value, comment)
			//p.addPendingComment(comment)
			break loop
		case '$':
			var variable Variable
			variable = p.parseVariable()
			value.appendVariable(variable)
		case scanner.EOF:
			break loop
		case '(':
			if endParen {
				parens++
			}
			value.appendString("(")
			p.accept('(')
		default:
			value.appendString(p.scanner.TokenText())
			p.accept(p.tok)
		}
	}

	if parens > 0 {
		p.errorf("expected closing paren %s", value.Dump())
	}
	return value
}

func (p *parser) parseVariable() Variable {
	p.accept('$')
	var name *MakeString
	switch p.tok {
	case '(':
		return p.parseBracketedVariable('(', ')')
	case '{':
		return p.parseBracketedVariable('{', '}')
	case '$':
		name = SimpleMakeString("__builtin_dollar")
	case scanner.EOF:
		p.errorf("expected variable name, found %s",
			scanner.TokenString(p.tok))
	default:
		name = p.parseExpression(variableNameEndRunes...)
	}

	return p.nameToVariable(name)
}

func (p *parser) parseBracketedVariable(start, end rune) Variable {
	p.accept(start)
	name := p.parseExpression(end)
	p.accept(end)
	return p.nameToVariable(name)
}

func (p *parser) nameToVariable(name *MakeString) Variable {
	return Variable{
		Name: name,
	}
}

func (p *parser) parseRule(target *MakeString) {
	prerequisites, newLine := p.parseRulePrerequisites(target)

	recipe := ""
loop:
	for {
		if newLine {
			if p.tok == '\t' {
				p.accept('\t')
				newLine = false
				continue loop
			} else if p.parseDirective() {
				newLine = false
				continue
			} else {
				break loop
			}
		}

		newLine = false
		switch p.tok {
		case '\\':
			p.parseEscape()
			recipe += string(p.tok)
			p.accept(p.tok)
		case '\n':
			newLine = true
			recipe += "\n"
			p.accept('\n')
		case scanner.EOF:
			break loop
		default:
			recipe += p.scanner.TokenText()
			p.accept(p.tok)
		}
	}

	if prerequisites != nil {
		p.syntaxTree.addNode(&Rule{
			Target:        target,
			Prerequisites: prerequisites,
			Recipe:        recipe,
		})
	}
}

func (p *parser) parseRulePrerequisites(target *MakeString) (*MakeString, bool) {
	newLine := false

	p.ignoreSpaces()

	prerequisites := p.parseExpression('#', '\n', ';', ':', '=')

	switch p.tok {
	case '\n':
		p.accept('\n')
		newLine = true
	case '#':
		var comment = p.parseComment()
		comment.ApplicableTo = prerequisites
		//p.addNodeComment(prerequisites, comment)
		p.addPendingComment(comment)

		newLine = true
	case ';':
		p.accept(';')
	case ':':
		p.accept(':')
		if p.tok == '=' {
			p.parseAssignment(":=", target, prerequisites)
			return nil, true
		} else {
			more := p.parseExpression('#', '\n', ';')
			prerequisites.appendMakeString(more)
		}
	case '=':
		p.parseAssignment("=", target, prerequisites)
		return nil, true
	default:
		p.errorf("unexpected token %s after rule prerequisites", scanner.TokenString(p.tok))
	}

	return prerequisites, newLine
}

func (p *parser) parseComment() *Comment {
	p.accept('#')
	commentText := ""
loop:
	for {
		switch p.tok {
		case '\\':
			p.parseEscape()
			if p.tok == '\n' {
				commentText += "\n"
			} else {
				commentText += "\\" + p.scanner.TokenText()
			}
			p.accept(p.tok)
		case '\n':
			p.accept('\n')
			break loop
		case scanner.EOF:
			break loop
		default:
			commentText += p.scanner.TokenText()
			p.accept(p.tok)
		}
	}

	fmt.Println(fmt.Sprintf("parser parsed comment '%v'", commentText))
	return &Comment{commentText, nil}
}

func (p *parser) parseAssignment(t string, target *MakeString, ident *MakeString) {
	fmt.Println("parser starting to parse assignment")
	// The value of an assignment is everything including and after the first
	// non-whitespace character after the = until the end of the logical line,
	// which may included escaped newlines
	p.accept('=')
	p.dumpUnassociatedComments()
	value := p.parseExpression()
	value.TrimLeftSpaces()
	if ident.EndsWith('+') && t == "=" {
		ident.TrimRightOne()
		t = "+="
	}

	ident.TrimRightSpaces()

	p.attachUnassociatedCommentsTo(value)

	p.addNode(&Assignment{
		Name:   ident,
		Value:  value,
		Target: target,
		Type:   t,
	})
	fmt.Println("reassigned currentNode to ", value)
	fmt.Println("parser done parsing assignment")
}

type androidMkModule struct {
	assignments map[string]string
}

type androidMkFile struct {
	assignments map[string]string
	modules     []androidMkModule
	includes    []string
}

var directives = [...]string{
	"define",
	"else",
	"endef",
	"endif",
	"ifdef",
	"ifeq",
	"ifndef",
	"ifneq",
	"include",
	"-include",
}

var functions = [...]string{
	"abspath",
	"addprefix",
	"addsuffix",
	"basename",
	"dir",
	"notdir",
	"subst",
	"suffix",
	"filter",
	"filter-out",
	"findstring",
	"firstword",
	"flavor",
	"join",
	"lastword",
	"patsubst",
	"realpath",
	"shell",
	"sort",
	"strip",
	"wildcard",
	"word",
	"wordlist",
	"words",
	"origin",
	"foreach",
	"call",
	"info",
	"error",
	"warning",
	"if",
	"or",
	"and",
	"value",
	"eval",
	"file",
}

func init() {
	sort.Strings(directives[:])
	sort.Strings(functions[:])
}

func isDirective(s string) bool {
	for _, d := range directives {
		if s == d {
			return true
		} else if s < d {
			return false
		}
	}
	return false
}

func isFunctionName(s string) bool {
	for _, f := range functions {
		if s == f {
			return true
		} else if s < f {
			return false
		}
	}
	return false
}

func isWhitespace(ch rune) bool {
	return ch == ' ' || ch == '\t' || ch == '\n'
}

func isValidVariableRune(ch rune) bool {
	return ch != scanner.Ident && ch != ':' && ch != '=' && ch != '#'
}

var whitespaceRunes = []rune{' ', '\t', '\n'}
var variableNameEndRunes = append([]rune{':', '=', '#', ')', '}'}, whitespaceRunes...)

func (p *parser) ignoreSpaces() int {
	skipped := 0
	for p.tok == ' ' || p.tok == '\t' {
		p.accept(p.tok)
		skipped++
	}
	return skipped
}

func (p *parser) ignoreWhitespace() {
	for isWhitespace(p.tok) {
		p.accept(p.tok)
	}
}
