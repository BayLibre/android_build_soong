package parser

import "fmt"

type Assignment struct {
	Target *MakeString
	Name   *MakeString
	Value  *MakeString
	Type   string
}

func (x Assignment) Children() []ParseNode {
	return []ParseNode{x.Target, x.Name, x.Value}
}

func (x *Assignment) Dump() string {
	target := ""
	if x.Target != nil {
		target = x.Target.Dump() + ": "
	}
	return target + x.Name.Dump() + " " + x.Type + " " + x.Value.Dump()
}

type ParseNode interface {
	//IsMkParseNode() // By requiring types to specify this unused method, it makes it easier to ensure that the intended types are passed everywhere instead of accidentally passing a pointer (which would satisfy the interface if the interface were empty)
	Dump() string
	Children() []ParseNode
}

type SyntaxTree struct {
	Nodes    []ParseNode
	comments map[ParseNode](*CommentPair)
}

func (t *SyntaxTree) addNode(node ParseNode) {
	t.Nodes = append(t.Nodes, node)
}
func (t *SyntaxTree) getComments(parseNode ParseNode) *CommentPair {
	if parseNode == nil {
		panic("Invalid nil value for parseNode")
	}
	var comments, ok = t.comments[parseNode]
	if !ok {
		comments = &CommentPair{}
		t.comments[parseNode] = comments
	}
	return comments
}

// returns a list of all comments held by the given ParseNode or any of its descendents
func (t *SyntaxTree) GetAllComments(parseNode ParseNode) (comments []Comment) {
	fmt.Printf("Getting all comments of %#v\n", parseNode)
	comments = make([]Comment, 0)
	var nodeComments = t.getComments(parseNode)
	comments = append(comments, nodeComments.preComments...)
	comments = append(comments, nodeComments.postComments...)
	for _, child := range parseNode.Children() {
		// recurse into child node
		var childComments = t.GetAllComments(child)
		// append to result
		comments = append(comments, childComments...)
	}
	fmt.Println("Got ", len(comments), " comments")
	return comments
}

type CommentPair struct {
	preComments  []Comment
	postComments []Comment
}

func (c *CommentPair) addPreComment(comment Comment) {
	c.preComments = append(c.preComments, comment)
}
func (c *CommentPair) addPostComment(comment Comment) {
	fmt.Printf("appending post comments at %p\n", c)
	c.postComments = append(c.postComments, comment)
}
func (c *CommentPair) PostComments() (count int) {
	fmt.Printf("getting post comments at %p\n", c)
	return len(c.postComments)
}

type Comment struct {
	Text         string
	ApplicableTo ParseNode
}

func (x Comment) Children() []ParseNode {
	return []ParseNode{}
}

func (x *Comment) Dump() string {
	return "#" + x.Text
}

type Directive struct {
	Name string
	Args *MakeString
}

func (x Directive) Children() []ParseNode {
	return []ParseNode{x.Args}
}

func (x *Directive) Dump() string {
	return x.Name + " " + x.Args.Dump()
}

type Rule struct {
	Target        *MakeString
	Prerequisites *MakeString
	Recipe        string
}

func (x Rule) Children() []ParseNode {
	return []ParseNode{x.Target, x.Prerequisites}
}

func (x *Rule) Dump() string {
	recipe := ""
	if x.Recipe != "" {
		recipe = "\n" + x.Recipe
	}
	return "rule:       " + x.Target.Dump() + ": " + x.Prerequisites.Dump() + recipe
}

type Variable struct {
	Name *MakeString
}

func (v Variable) Children() [](ParseNode) {
	return [](ParseNode){*(v.Name)}
}
func (x Variable) Dump() string {
	return "$(" + x.Name.Dump() + ")"
}
