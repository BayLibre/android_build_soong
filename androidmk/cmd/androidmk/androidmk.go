package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"text/scanner"

	mkparser "android/soong/androidmk/parser"

	bpparser "blueprint/parser"
)

// TODO: non-expanded variables with expressions

type bpFile struct {
	comments          []bpparser.Comment
	defs              []bpparser.Definition
	localAssignments  map[string]*bpparser.Value
	globalAssignments map[string]*bpparser.Value
	scope             mkparser.Scope
	module            *bpparser.Module
	pos               scanner.Position
}

func (f *bpFile) errorf(thing mkparser.MakeThing, s string, args ...interface{}) {
	orig := thing.Dump()
	s = fmt.Sprintf(s, args...)
	comment := bpparser.Comment{
		Comment: fmt.Sprintf("/* ANDROIDMK TRANSLATION ERROR: %s from %s */", s, orig),
		Pos:     f.pos,
	}
	f.comments = append(f.comments, comment)
}

func main() {
	b, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	p := mkparser.NewParser(os.Args[1], bytes.NewBuffer(b))

	things, errs := p.Parse()
	if len(errs) > 0 {
		for _, err := range errs {
			fmt.Println("ERROR: ", err)
		}
		return
	}

	file := &bpFile{
		scope:             androidScope(),
		localAssignments:  make(map[string]*bpparser.Value),
		globalAssignments: make(map[string]*bpparser.Value),
	}

	prevLine := 0
	line := 0
	for _, t := range things {
		line++
		file.pos = t.Pos()
		if file.pos.Line > prevLine+1 {
			line++
		}
		file.pos.Line = line
		prevLine = t.EndPos().Line

		if comment, ok := t.AsComment(); ok {
			file.comments = append(file.comments, bpparser.Comment{
				Pos:     file.pos,
				Comment: "//" + comment.Comment,
			})
		} else if assignment, ok := t.AsAssignment(); ok {
			handleAssignment(file, assignment)
		} else if directive, ok := t.AsDirective(); ok {
			switch directive.Name {
			case "include":
				val := directive.Args.Value(file.scope)
				switch val {
				case build_shared_library, build_static_library,
					build_executable, build_host_executable,
					build_prebuilt, build_host_static_library:
					makeModule(file, val)
				case clear_vars:
					resetModule(file)
				default:
					file.errorf(directive, "unsupported include '%s'", directive.Args.Value(file.scope))
					continue
				}
			}
		}
	}

	out, err := bpparser.Print(&bpparser.File{
		Defs:     file.defs,
		Comments: file.comments,
	})
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(string(out))
}

func handleAssignment(file *bpFile, assignment mkparser.Assignment) {
	if !assignment.Name.Const() {
		file.errorf(assignment, "unsupported non-const variable name")
		return
	}

	if assignment.Target != nil {
		file.errorf(assignment, "unsupported target assignment")
		return
	}

	name := assignment.Name.Value(nil)

	if prop, ok := stringProperties[name]; ok {
		setVariable(file, assignment, prop, bpparser.String, true)
	} else if prop, ok := listProperties[name]; ok {
		setVariable(file, assignment, prop, bpparser.List, true)
	} else if prop, ok := boolProperties[name]; ok {
		setVariable(file, assignment, prop, bpparser.Bool, true)
	} else {
		if name == "LOCAL_PATH" {
			// Nothing to do, except maybe avoid the "./" in paths?
		} else if strings.HasPrefix(name, "LOCAL_") {
			//setVariable(file, assignment, name, bpparser.String, true)
			switch name {
			case "LOCAL_ADDITIONAL_DEPENDENCIES":
				// TODO: check for only .mk files?
			default:
				file.errorf(assignment, "unsupported assignment to %s", name)
				return
			}
		} else {
			setVariable(file, assignment, name, bpparser.List, false)
		}
	}
}

func makeModule(file *bpFile, t string) {
	file.module.Type = bpparser.Ident{
		Name: t,
		Pos:  file.module.LbracePos,
	}
	file.module.RbracePos = file.pos
	file.defs = append(file.defs, file.module)
}

func resetModule(file *bpFile) {
	file.module = &bpparser.Module{}
	file.module.LbracePos = file.pos
	file.localAssignments = make(map[string]*bpparser.Value)
}

func setVariable(file *bpFile, assignment mkparser.Assignment, name string,
	typ bpparser.ValueType, local bool) {

	pos := file.pos
	val := assignment.Value

	var oldValue *bpparser.Value
	if local {
		oldValue = file.localAssignments[name]
	} else {
		oldValue = file.globalAssignments[name]
	}

	var exp *bpparser.Value
	var err error
	switch typ {
	case bpparser.List:
		exp, err = makeToListExpression(val)
	case bpparser.String:
		exp, err = makeToStringExpression(val)
	case bpparser.Bool:
		exp, err = makeToBoolExpression(val)
	default:
		panic("unknown type")
	}

	if err != nil {
		file.errorf(assignment, "unsupported expression: %s", err.Error())
		return
	}

	if oldValue != nil {
		if assignment.Type != "+=" {
			file.errorf(assignment, "assignment overwrites existing variable with %s", typ)
			return
		}

		val, err := addValues(oldValue, exp)
		if err != nil {
			file.errorf(assignment, "unsupported addition: %s", err.Error())
			return
		}
		val.Expression.Pos = pos
		*oldValue = *val
	} else {
		if local {
			prop := &bpparser.Property{
				Name:  bpparser.Ident{Name: name, Pos: pos},
				Pos:   pos,
				Value: *exp,
			}
			file.localAssignments[name] = &prop.Value
			file.module.Properties = append(file.module.Properties, prop)
		} else {
			a := &bpparser.Assignment{
				Name: bpparser.Ident{
					Name: name,
					Pos:  pos,
				},
				Value: *exp,
				Pos:   pos,
			}
			file.globalAssignments[name] = &a.Value
			file.defs = append(file.defs, a)
		}
	}
}

//func pushAssignment(defs *[]bpparser.Definition, name string, v *bpparser.Value,
//	pos scanner.Position) {
//	defs = append(defs, &bpparser.Assignment{
//		Name: bpparser.Ident{
//			Name: name,
//			Pos:  pos,
//		},
//		Value: *v,
//		Pos:   pos,
//	})
//}

//func pushProperty(module *bpparser.Module, name string, v *bpparser.Value, pos scanner.Position) {
//	module.Properties = append(module.Properties, newProp(name, pos, v))
//}
