package parser

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"text/scanner"
)

// The parser defined in this file parses Makefiles and returns a syntaxTree that is essentially a list of statements

var errTooManyErrors = errors.New("too many errors")

const maxErrors = 100

type ParseError struct {
	Err error
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("%s", e.Err)
}
func (e *ParseError) Children() []ParseNode {
	return make([]ParseNode, 0)
}
func (e *ParseError) Dump() string {
	return fmt.Sprintf("%s", e.Err)
}

type ParseTree struct {
	*SyntaxTree

	// sourcePositions tells where tokens were originally found in the source text
	// Because an androidmk SyntaxTree is never generated programatically and is only ever generated via a parse,
	// it would still be ok to move this information to each individual ParseNode
	// However, the Blueprint SyntaxTree can be created programatically (in fact, androidmk creates one)
	// and the position doesn't always make sense, so in Blueprint it's stored separately.
	// The reason androidmk does this is to be easy to remember by mirroring Blueprint
	sourcePositions map[ParseNode]scanner.Position
}

func NewParseTree() *ParseTree {
	tree := &ParseTree{}
	tree.SyntaxTree = NewSyntaxTree()
	tree.sourcePositions = make(map[ParseNode]scanner.Position, 0)
	return tree
}
func (p *ParseTree) GetSourcePosition(node ParseNode) scanner.Position {
	return p.sourcePositions[node]
}
func (p *ParseTree) setPosition(parseNode ParseNode, pos scanner.Position) {
	p.sourcePositions[parseNode] = pos
}

type parser struct {
	// low-level token scanner
	scanner scanner.Scanner
	// current token being parsed
	tok rune
	// problems with parsing
	errors []error

	// the resultant parse
	parseTree *ParseTree

	// comments that have been parsed but that haven't yet been attached to other statements (generally comments get attached to the statement following them)
	pendingComments [](*Comment)
	// whether to throw an exception on the first error
	Strict bool
	// line number of current token
	lineNumber int
	// location of previous token
	prevPosition scanner.Position
	// the latest position at which the parse stack was empty
	prevRootLocation scanner.Position
}

func (p *parser) addCommentAfter(existingNode ParseNode, commentNode *Comment) {
	p.parseTree.getComments(existingNode).addPostComment(*commentNode)
}

func (p *parser) addPendingComment(commentNode *Comment) {
	p.pendingComments = append(p.pendingComments, commentNode)
}

func (p *parser) attachUnassociatedCommentsBefore(parseNode ParseNode) {
	var pendingComments = p.pullPendingComments()
	if len(pendingComments) > 0 {
		var commentContainer = p.parseTree.getComments(parseNode)
		for _, comment := range pendingComments {
			commentContainer.addPreComment(*comment)
		}
	}
}

func (p *parser) dumpUnassociatedComments() {
	var pendingComments = p.pullPendingComments()
	if len(pendingComments) > 0 {
		for _, comment := range pendingComments {
			p.addNode(comment)
		}
	}
}

func (p *parser) pullPendingComments() [](*Comment) {
	var pendingComments = p.pendingComments
	p.pendingComments = make([](*Comment), 0)
	return pendingComments
}

func (p *parser) addNode(node ParseNode) {
	p.dumpUnassociatedComments()

	p.parseTree.addNode(node)

	// TODO always assign the exact line number to a node, rather than using the end-of-line default assigned here
	p.setDefaultNodePosition(node)
}

func (p *parser) setDefaultNodePosition(node ParseNode) {

	p.useNodePositionIfEmpty(node, p.prevRootLocation)
}

func (p *parser) useNodePositionIfEmpty(node ParseNode, position scanner.Position) {
	existingPos, found := p.parseTree.sourcePositions[node]
	if found {
		position = existingPos
	}
	p.parseTree.sourcePositions[node] = position
	for _, child := range node.Children() {
		p.useNodePositionIfEmpty(child, position)
	}
}

func (p *parser) Parse() (*ParseTree, []error) {
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

	return p.parseTree, p.errors
}

func NewSyntaxTree() (tree *SyntaxTree) {
	tree = &SyntaxTree{}
	tree.comments = map[ParseNode]*CommentPair{}
	return tree
}

func NewParser(filename string, r io.Reader) *parser {
	p := &parser{}
	p.parseTree = NewParseTree()

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
	p.lineNumber = 1
	p.next()
	return p
}

func (p *parser) errorf(format string, args ...interface{}) {
	err := &ParseError{
		Err: fmt.Errorf(format, args...),
	}
	p.parseTree.setPosition(err, p.scanner.Position)
	p.errors = append(p.errors, err)
	p.addNode(err)
	if len(p.errors) >= maxErrors {
		fmt.Println(err)
		panic(errTooManyErrors)
	}
}

func (p *parser) accept(toks ...rune) bool {
	for _, tok := range toks {
		if p.tok != tok {
			p.errorf("'accept' method expected %s, found %s", scanner.TokenString(tok),
				scanner.TokenString(p.tok))
			return false
		}
		p.next()
	}
	return true
}

func (p *parser) next() {
	if p.tok != scanner.EOF {
		p.prevPosition = p.scanner.Position
		p.tok = p.scanner.Scan()
		for p.tok == '\r' {
			p.tok = p.scanner.Scan()
		}
		p.lineNumber = p.scanner.Line
	}
}

func (p *parser) parseLines() {
loop:
	for {

		var prevLineNumber = p.prevPosition.Line

		p.ignoreWhitespace()

		p.prevRootLocation = p.scanner.Position

		var newLineNumber = p.lineNumber

		if newLineNumber > prevLineNumber+1 {
			p.addPendingComment(newBlankLine())
		}
		p.dumpUnassociatedComments()

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
				p.addNode(v)
			} else if !ident.Empty() {
				p.errorf("expected directive, rule, or assignment after ident '" + ident.Dump() + "'")
				break
			}
			switch p.tok {
			case scanner.EOF:
				break loop
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
		expression.TrimRightSpaces()
	}

	directive := &Directive{
		Name: d,
		Args: expression,
	}
	p.addNode(directive)
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
	prevMode := p.scanner.Mode
	p.scanner.Mode = 0
	p.accept('\\')
	p.scanner.Mode = prevMode
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
			p.addCommentAfter(value, comment)
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

	inlineComments := make([]*Comment, 0)
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
			} else if p.tok == '#' {
				inlineComments = append(inlineComments, p.parseComment())
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
		rule := &Rule{
			Target:        target,
			Prerequisites: prerequisites,
			Recipe:        recipe,
		}
		p.addNode(rule)
	}
	// inline comments are to be added after the rule
	for _, comment := range inlineComments {
		p.addPendingComment(comment)
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

	return newFulllineComment(commentText)
}

func newFulllineComment(text string) (comment *Comment) {
	return &Comment{text, FullLineText}
}
func newBlankLine() (comment *Comment) {
	return &Comment{"\n", FullLineBlank}
}

func (p *parser) parseAssignment(t string, target *MakeString, ident *MakeString) {
	// The value of an assignment is everything including and after the first
	// non-whitespace character after the = until the end of the logical line,
	// which may included escaped newlines
	p.accept('=')
	value := p.parseExpression()
	value.TrimLeftSpaces()
	value.TrimRightSpaces()
	if ident.EndsWith('+') && t == "=" {
		ident.TrimRightOne()
		t = "+="
	}

	ident.TrimRightSpaces()

	p.attachUnassociatedCommentsBefore(value)

	p.addNode(&Assignment{
		Name:   ident,
		Value:  value,
		Target: target,
		Type:   t,
	})
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
