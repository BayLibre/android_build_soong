package parser

//import "fmt"

type Assignment struct {
	Target *MakeString
	Name   *MakeString
	Value  *MakeString
	Type   string
}

func (x Assignment) Children() []ParseNode {
	results := make([]ParseNode, 0)

	// Unfortunately if we consolidate these functions into a loop, then the '!= nil' check always returns true
	// because Go considers variables whose declared data type is an interface to be non-nil
	// So we have to either list each if-statement separately or use reflection to get a list of all the children
	if x.Target != nil {
		results = append(results, x.Target)
	}
	if x.Name != nil {
		results = append(results, x.Name)
	}
	if x.Value != nil {
		results = append(results, x.Value)
	}

	return results
}

func (x *Assignment) Dump() string {
	target := ""
	if x.Target != nil {
		target = x.Target.Dump() + ": "
	}
	return target + x.Name.Dump() + " " + x.Type + " " + x.Value.Dump()
}

type ParseNode interface {
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
		panic("Cannot lookup the comment container for a nil value for parseNode")
	}
	var comments, ok = t.comments[parseNode]
	if !ok {
		comments = &CommentPair{}
		t.comments[parseNode] = comments
	}
	return comments
}

// returns a list of all comments held by the given ParseNode or any of its descendents
func (t *SyntaxTree) GetAllComments(parseNode ParseNode) (comments *CommentPair) {
	comments = &CommentPair{}
	var nodeComments = t.getComments(parseNode)
	comments.preComments = append(comments.preComments, nodeComments.preComments...)
	comments.postComments = append(comments.postComments, nodeComments.postComments...)
	for _, child := range parseNode.Children() {
		// recurse into child node
		var childComments = t.GetAllComments(child)
		// append to result
		comments.preComments = append(comments.preComments, childComments.preComments...)
		comments.postComments = append(comments.postComments, childComments.postComments...)
	}
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
	c.postComments = append(c.postComments, comment)
}
func (c *CommentPair) PostComments() []Comment {
	return c.postComments
}
func (c *CommentPair) PreComments() []Comment {
	return c.preComments
}

type commentType int

const (
	FullLineText  = iota // starts with "//"
	FullLineBlank        // just a single newline
	//InlineBlank          // only spaces and tabs - not currently needed but could be added in the future
)

type Comment struct {
	Text string
	Type commentType
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
	return [](ParseNode){v.Name}
}
func (x Variable) Dump() string {
	return "$(" + x.Name.Dump() + ")"
}
