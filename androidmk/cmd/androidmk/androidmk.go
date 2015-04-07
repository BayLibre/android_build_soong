package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"text/scanner"

	mkparser "android/soong/androidmk/parser"

	bpparser "github.com/google/blueprint/parser"
)

// TODO: non-expanded variables with expressions

type bpFile struct {
	comments          []bpparser.Comment
	defs              []bpparser.Definition
	localAssignments  map[string]*bpparser.Property
	globalAssignments map[string]*bpparser.Value
	scope             mkparser.Scope
	module            *bpparser.Module

	pos            scanner.Position
	prevLine, line int
}

func (f *bpFile) errorf(thing mkparser.MakeThing, s string, args ...interface{}) {
	orig := thing.Dump()
	s = fmt.Sprintf(s, args...)
	f.comments = append(f.comments, bpparser.Comment{
		Comment: []string{fmt.Sprintf("// ANDROIDMK TRANSLATION ERROR: %s", s)},
		Pos:     f.pos,
	})
	lines := strings.Split(orig, "\n")
	for _, l := range lines {
		f.incPos()
		f.comments = append(f.comments, bpparser.Comment{
			Comment: []string{"// " + l},
			Pos:     f.pos,
		})
	}
}

func (f *bpFile) setPos(pos, endPos scanner.Position) {
	f.pos = pos

	f.line++
	if f.pos.Line > f.prevLine+1 {
		f.line++
	}

	f.pos.Line = f.line
	f.prevLine = endPos.Line
}

func (f *bpFile) incPos() {
	f.pos.Line++
	f.line++
	f.prevLine++
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
		localAssignments:  make(map[string]*bpparser.Property),
		globalAssignments: make(map[string]*bpparser.Value),
	}

	var conds []*conditional
	var cond *conditional

	for _, t := range things {
		file.setPos(t.Pos(), t.EndPos())

		if comment, ok := t.AsComment(); ok {
			file.comments = append(file.comments, bpparser.Comment{
				Pos:     file.pos,
				Comment: []string{"//" + comment.Comment},
			})
		} else if assignment, ok := t.AsAssignment(); ok {
			handleAssignment(file, assignment, cond)
		} else if directive, ok := t.AsDirective(); ok {
			switch directive.Name {
			case "include":
				val := directive.Args.Value(file.scope)
				switch {
				case soongModuleTypes[val]:
					handleModuleConditionals(file, directive, cond)
					makeModule(file, val)
				case val == clear_vars:
					resetModule(file)
				default:
					file.errorf(directive, "unsupported include")
					continue
				}
			case "ifeq", "ifneq", "ifdef", "ifndef":
				args := directive.Args.Dump()
				eq := directive.Name == "ifeq" || directive.Name == "ifdef"
				if _, ok := conditionalTranslations[args]; ok {
					newCond := conditional{args, eq}
					conds = append(conds, &newCond)
					if cond == nil {
						cond = &newCond
					} else {
						file.errorf(directive, "unsupported nested conditional")
					}
				} else {
					file.errorf(directive, "unsupported conditional")
					conds = append(conds, nil)
					continue
				}
			case "else":
				if len(conds) == 0 {
					file.errorf(directive, "missing if before else")
					continue
				} else if conds[len(conds)-1] == nil {
					file.errorf(directive, "else from unsupported contitional")
					continue
				}
				cond.eq = !cond.eq
			case "endif":
				if len(conds) == 0 {
					file.errorf(directive, "missing if before endif")
					continue
				} else if conds[len(conds)-1] == nil {
					file.errorf(directive, "endif from unsupported contitional")
					conds = conds[:len(conds)-1]
				} else {
					if cond == conds[len(conds)-1] {
						cond = nil
					}
					conds = conds[:len(conds)-1]
				}
			default:
				file.errorf(directive, "unsupported directive")
				continue
			}
		} else {
			file.errorf(t, "unsupported line")
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

	fmt.Print(string(out))
}

func handleAssignment(file *bpFile, assignment mkparser.Assignment, c *conditional) {
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
		for k, v := range propertyPrefixes {
			if strings.HasSuffix(name, "_"+k) {
				name = strings.TrimSuffix(name, "_"+k)
				prefix = v
				break
			}
		}

		if c != nil {
			if prefix != "" {
				file.errorf(assignment, "prefix assignment inside conditional, skipping conditional")
			} else {
				cond := c.cond
				if !c.eq {
					cond = "!" + cond
				}
				var ok bool
				if prefix, ok = conditionalTranslations[cond]; !ok {
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

	var err error
	if prop, ok := stringProperties[name]; ok {
		err = setVariable(file, assignment.Value, assignment.Type == "+=", prefix, prop, bpparser.String, true)
	} else if prop, ok := listProperties[name]; ok {
		err = setVariable(file, assignment.Value, assignment.Type == "+=", prefix, prop, bpparser.List, true)
	} else if prop, ok := boolProperties[name]; ok {
		err = setVariable(file, assignment.Value, assignment.Type == "+=", prefix, prop, bpparser.Bool, true)
	} else if _, ok := deleteProperties[name]; ok {
		return
	} else {
		if name == "LOCAL_PATH" {
			// Nothing to do, except maybe avoid the "./" in paths?
		} else if name == "LOCAL_ARM_MODE" {
			// This is a hack to get the LOCAL_ARM_MODE value inside
			// of an arch: { arm: {} } block.
			armModeAssign := assignment
			armModeAssign.Name = mkparser.SimpleMakeString("LOCAL_ARM_MODE_HACK_arm", assignment.Name.Pos)
			handleAssignment(file, armModeAssign, c)
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
			err = setVariable(file, assignment.Value, assignment.Type == "+=", prefix, name, bpparser.List, false)
		}
	}
	if err != nil {
		file.errorf(assignment, err.Error())
	}
}

func handleModuleConditionals(file *bpFile, directive mkparser.Directive, c *conditional) {
	if c == nil {
		return
	}

	cond := c.cond
	inverseCond := "!" + c.cond
	if !c.eq {
		cond, inverseCond = inverseCond, cond
	}

	var prefix, disabledPrefix string
	var ok bool
	if prefix, ok = conditionalTranslations[cond]; !ok {
		panic("unknown conditional " + cond)
	}
	if disabledPrefix, ok = conditionalTranslations[inverseCond]; !ok {
		panic("unknown conditional " + inverseCond)
	}

	names := strings.Split(prefix, ".")
	if len(names) != 2 {
		panic("expected class.type")
	}
	class := names[0]
	typ := names[1]
	classProp := file.localAssignments[class]

	// Hoist all properties inside the condtional up to the top level
	file.module.Properties = file.localAssignments[prefix].Value.MapValue
	file.module.Properties = append(file.module.Properties, classProp)
	file.localAssignments[prefix].Value.MapValue = nil
	for i := range classProp.Value.MapValue {
		if classProp.Value.MapValue[i].Name.Name == typ {
			classProp.Value.MapValue = append(classProp.Value.MapValue[:i], classProp.Value.MapValue[i+1:]...)
		}
	}

	// Create a fake assignment with enabled = false
	err := setVariable(file, mkparser.SimpleMakeString("true", file.pos), false,
		disabledPrefix, "disabled", bpparser.Bool, true)
	if err != nil {
		file.errorf(directive, err.Error())
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
	file.localAssignments = make(map[string]*bpparser.Property)
}

func setVariable(file *bpFile, val *mkparser.MakeString, plusequals bool, prefix, name string,
	typ bpparser.ValueType, local bool) error {

	if prefix != "" {
		name = prefix + "." + name
	}

	pos := file.pos

	var oldValue *bpparser.Value
	if local {
		oldProp := file.localAssignments[name]
		if oldProp != nil {
			oldValue = &oldProp.Value
		}
	} else {
		oldValue = file.globalAssignments[name]
	}

	var exp *bpparser.Value
	var err error
	switch typ {
	case bpparser.List:
		exp, err = makeToListExpression(val, file.scope)
	case bpparser.String:
		exp, err = makeToStringExpression(val, file.scope)
	case bpparser.Bool:
		exp, err = makeToBoolExpression(val)
	default:
		panic("unknown type")
	}

	if err != nil {
		return err
	}

	if local {
		if oldValue != nil && plusequals {
			val, err := addValues(oldValue, exp)
			if err != nil {
				return fmt.Errorf("unsupported addition: %s", err.Error())
			}
			val.Expression.Pos = pos
			*oldValue = *val
		} else {
			names := strings.Split(name, ".")
			container := &file.module.Properties

			for i, n := range names[:len(names)-1] {
				fqn := strings.Join(names[0:i+1], ".")
				prop := file.localAssignments[fqn]
				if prop == nil {
					prop = &bpparser.Property{
						Name: bpparser.Ident{Name: n, Pos: pos},
						Pos:  pos,
						Value: bpparser.Value{
							Type:     bpparser.Map,
							MapValue: []*bpparser.Property{},
						},
					}
					file.localAssignments[fqn] = prop
					*container = append(*container, prop)
				}
				container = &prop.Value.MapValue
			}

			prop := &bpparser.Property{
				Name:  bpparser.Ident{Name: names[len(names)-1], Pos: pos},
				Pos:   pos,
				Value: *exp,
			}
			file.localAssignments[name] = prop
			*container = append(*container, prop)
		}
	} else {
		if oldValue != nil && plusequals {
			a := &bpparser.Assignment{
				Name: bpparser.Ident{
					Name: name,
					Pos:  pos,
				},
				Value:     *exp,
				OrigValue: *exp,
				Pos:       pos,
				Assigner:  "+=",
			}
			file.defs = append(file.defs, a)
		} else {
			a := &bpparser.Assignment{
				Name: bpparser.Ident{
					Name: name,
					Pos:  pos,
				},
				Value:     *exp,
				OrigValue: *exp,
				Pos:       pos,
				Assigner:  "=",
			}
			file.globalAssignments[name] = &a.Value
			file.defs = append(file.defs, a)
		}
	}

	return nil
}
