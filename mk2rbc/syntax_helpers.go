package mk2rbc

import "go.starlark.net/syntax"

func newStringLiteral(literal string) *syntax.Literal {
	return &syntax.Literal{
		Token:    syntax.STRING,
		Value:    literal,
	}
}

func newIntLiteral(literal int64) *syntax.Literal {
	return &syntax.Literal{
		Token:    syntax.INT,
		Value:    literal,
	}
}

func newBoolLiteral(literal bool) *syntax.Ident {
	if literal {
		return &syntax.Ident{Name: "True"}
	} else {
		return &syntax.Ident{Name: "False"}
	}
}
