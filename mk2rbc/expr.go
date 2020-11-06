// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mk2rbc

import (
	"fmt"
	"strconv"
	"strings"

	mkparser "android/soong/androidmk/parser"
)

// Represents an expression in the Starlark code. An expression has
// a type, and it can be evaluated.
type starlarkExpr interface {
	starlarkNode
	typ() starlarkType
	// Try to substitute variable values. If substitution result and whether it is
	// the same as the original expression.
	eval(valueMap map[string]starlarkExpr) (expr starlarkExpr, same bool)
}

func maybeString(expr starlarkExpr) (string, bool) {
	if x, ok := expr.(*stringLiteralExpr); ok {
		return x.literal, true
	}
	return "", false
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

func (_ *stringLiteralExpr) typ() starlarkType {
	return starlarkTypeString
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

func (_ *intLiteralExpr) typ() starlarkType {
	return starlarkTypeInt
}

// interpolateExpr represents Starlark's interpolation operator <string> % list
// we break <string> into a list of chunks, i.e., "first%second%third" % (X, Y)
// will have chunks = ["first", "second", "third"] and args = [X, Y]
type interpolateExpr struct {
	chunks []string // string chunks, separated by '%'
	args   []starlarkExpr
}

func (xi *interpolateExpr) emit(gctx *generationContext) {
	if len(xi.chunks) != len(xi.args)+1 {
		panic(fmt.Errorf("#chunks(%d) != #args(%d)+1", len(xi.chunks), len(xi.args)))
	}
	// Generate format as join of chunks, but first escape '%' in them
	format := strings.ReplaceAll(xi.chunks[0], "%", "%%")
	for _, chunk := range xi.chunks[1:] {
		format += "%s" + strings.ReplaceAll(chunk, "%", "%%")
	}
	gctx.writef("%q %% ", format)
	if len(xi.args) == 1 {
		xi.args[0].emit(gctx)
	} else {
		sep := "("
		for _, arg := range xi.args {
			gctx.write(sep)
			sep = ", "
			arg.emit(gctx)
		}
		gctx.write(")")
	}
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

func (_ *interpolateExpr) typ() starlarkType {
	return starlarkTypeString
}

type variableRefExpr struct {
	ref       variable
	isDefined bool
}

func (v *variableRefExpr) eval(map[string]starlarkExpr) (starlarkExpr, bool) {
	if predefined, ok := v.ref.(*predefinedVariable); ok {
		return predefined.value, false
	}
	return v, true
}

func (v *variableRefExpr) emit(gctx *generationContext) {
	v.ref.emitGet(gctx, v.isDefined)
}

func (v *variableRefExpr) typ() starlarkType {
	return v.ref.valueType()
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

func (_ *notExpr) typ() starlarkType {
	return starlarkTypeBool
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
	// Special case: one operand empty, the other is variable reference
	var v variable
	if s, ok := maybeString(eq.left); ok && s == "" {
		if vref, ok := eq.right.(*variableRefExpr); ok {
			v = vref.ref
		}
	} else if s, ok := maybeString(eq.right); ok && s == "" {
		if vref, ok := eq.left.(*variableRefExpr); ok {
			v = vref.ref
		}
	}
	if v != nil {
		if eq.isEq {
			gctx.write(" not ")
		}
		v.emitDefined(gctx)
		// TODO(asmundak): should we also check that the value is the default value?
		return
	}
	eq.left.emit(gctx)
	if eq.isEq {
		gctx.write(" == ")
	} else {
		gctx.write(" != ")
	}
	eq.right.emit(gctx)
}

func (_ *eqExpr) typ() starlarkType {
	return starlarkTypeBool
}

type variableDefinedExpr struct {
	v variable
}

func (v *variableDefinedExpr) eval(_ map[string]starlarkExpr) (expr starlarkExpr, same bool) {
	return v, true

}

func (v *variableDefinedExpr) emit(gctx *generationContext) {
	if v.v != nil {
		v.v.emitDefined(gctx)
		return
	}
	gctx.writef("%s(%q)", cfnWarning, "TODO(VAR)")
}

func (_ *variableDefinedExpr) typ() starlarkType {
	return starlarkTypeBool
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

	gctx.write("[")
	gctx.indentLevel += 2

	for _, item := range l.items {
		gctx.newLine()
		item.emit(gctx)
		gctx.write(",")
	}
	gctx.indentLevel -= 2
	gctx.newLine()
	gctx.write("]")
}

func (_ *listExpr) typ() starlarkType {
	return starlarkTypeList
}

func newStringListExpr(items []string) *listExpr {
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
	if len(c.items) == 1 {
		c.items[0].emit(gctx)
		return
	}

	if !gctx.inAssignment {
		c.items[0].emit(gctx)
		for _, item := range c.items[1:] {
			gctx.write(" + ")
			item.emit(gctx)
		}
		return
	}
	gctx.write("(")
	c.items[0].emit(gctx)
	gctx.indentLevel += 2
	for _, item := range c.items[1:] {
		gctx.write(" +")
		gctx.newLine()
		item.emit(gctx)
	}
	gctx.write(")")
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

func (_ *concatExpr) typ() starlarkType {
	return starlarkTypeList
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

func (_ *inExpr) typ() starlarkType {
	return starlarkTypeBool
}

type indexExpr struct {
	array starlarkExpr
	index starlarkExpr
}

func (ix indexExpr) emit(gctx *generationContext) {
	ix.array.emit(gctx)
	gctx.write("[")
	ix.index.emit(gctx)
	gctx.write("]")
}

func (ix indexExpr) typ() starlarkType {
	return starlarkTypeString
}

func (ix indexExpr) eval(valueMap map[string]starlarkExpr) (expr starlarkExpr, same bool) {
	newArray, isSameArray := ix.array.eval(valueMap)
	newIndex, isSameIndex := ix.index.eval(valueMap)
	if isSameArray && isSameIndex {
		return ix, true
	}
	return &indexExpr{newArray, newIndex}, false
}

type callExpr struct {
	object     starlarkExpr // nil if static call
	name       string
	args       []starlarkExpr
	returnType starlarkType
}

func (cx *callExpr) eval(valueMap map[string]starlarkExpr) (expr starlarkExpr, same bool) {
	newCallExpr := &callExpr{name: cx.name, args: make([]starlarkExpr, len(cx.args)),
		returnType: cx.returnType}
	if cx.object != nil {
		newCallExpr.object, same = cx.object.eval(valueMap)
	} else {
		same = true
	}
	for i, args := range cx.args {
		var s bool
		newCallExpr.args[i], s = args.eval(valueMap)
		same = same && s
	}
	if same {
		return cx, true
	}
	return newCallExpr, false
}

func (cx *callExpr) emit(gctx *generationContext) {
	if cx.object != nil {
		gctx.write("(")
		cx.object.emit(gctx)
		gctx.write(")")
		gctx.write(".", cx.name, "(")
	} else {
		kf, found := knownFunctions[cx.name]
		if !found {
			panic(fmt.Errorf("callExpr with unknown function %q", cx.name))
		}
		if kf.rtName[0] == '!' {
			panic(fmt.Errorf("callExpr for %q should not be there", cx.name))
		}
		gctx.write(kf.rtName, "(")
	}
	sep := ""
	for _, arg := range cx.args {
		gctx.write(sep)
		arg.emit(gctx)
		sep = ", "
	}
	gctx.write(")")
}

func (cx *callExpr) typ() starlarkType {
	return cx.returnType
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

func (_ *badExpr) typ() starlarkType {
	return starlarkTypeUnknown
}

func maybeConvertToStringList(expr starlarkExpr) starlarkExpr {
	if xString, ok := expr.(*stringLiteralExpr); ok {
		return newStringListExpr(strings.Fields(xString.literal))
	}
	return expr
}
