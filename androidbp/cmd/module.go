package main

import (
	"fmt"
	"strings"

	bpparser "github.com/google/blueprint/parser"
)

func copyBPValue(v *bpparser.Value) (ret bpparser.Value) {
	ret = *v
	ret.MapValue = copyBPProperties(v.MapValue)
	ret.ListValue = nil
	for i := range v.ListValue {
		ret.ListValue = append(ret.ListValue, copyBPValue(&v.ListValue[i]))
	}
	if v.Expression != nil {
		ret.Expression = copyBPExpression(v.Expression)
	}
	return
}

func copyBPExpression(e *bpparser.Expression) (ret *bpparser.Expression) {
	ret = &bpparser.Expression{}
	*ret = *e
	ret.Args[0] = copyBPValue(&e.Args[0])
	ret.Args[1] = copyBPValue(&e.Args[1])
	return
}

func copyBPProperty(p *bpparser.Property) (ret *bpparser.Property) {
	ret = &bpparser.Property{}
	*ret = *p
	ret.Value = copyBPValue(&p.Value)
	return
}

func copyBPProperties(props []*bpparser.Property) (ret []*bpparser.Property) {
	for _, prop := range props {
		ret = append(ret, copyBPProperty(prop))
	}
	return
}

func copyBPModule(mod *bpparser.Module) (ret *bpparser.Module) {
	ret = &bpparser.Module{}
	*ret = *mod
	ret.Properties = copyBPProperties(ret.Properties)
	return
}

type Module struct {
	bpmod      *bpparser.Module
	bpname     string
	mkname     string
	isHostRule bool
}

func newModule(mod *bpparser.Module) *Module {
	return &Module{
		bpmod:  copyBPModule(mod),
		bpname: mod.Type.Name,
	}
}

func (m *Module) translateRuleName() error {
	var name string
	if translation, ok := moduleTypeToRule[m.bpname]; ok {
		name = translation
	} else {
		return fmt.Errorf("Unknown module type %q", m.bpname)
	}

	if m.isHostRule {
		if trans, ok := targetToHostModuleRule[name]; ok {
			name = trans
		} else {
			return fmt.Errorf("No corresponding host rule for %q", name)
		}
	} else {
		m.isHostRule = strings.Contains(name, "HOST")
	}

	m.mkname = name

	return nil
}

func (m *Module) Properties() Properties {
	return Properties{&m.bpmod.Properties}
}

func (m *Module) PropBool(name string) bool {
	if prop, ok := m.Properties().Prop(name); ok {
		return prop.Value.BoolValue
	}
	return false
}

func (m *Module) IterateArchPropertiesWithName(name string, f func(Properties, *bpparser.Property)) {
	if p, ok := m.Properties().Prop(name); ok {
		f(m.Properties(), p)
	}
	for _, prop := range m.bpmod.Properties {
		switch prop.Name.Name {
		case "arch", "multilib", "target":
			for _, sub := range prop.Value.MapValue {
				props := Properties{&sub.Value.MapValue}
				if p, ok := props.Prop(name); ok {
					f(props, p)
				}
			}
		}
	}
}

type Properties struct {
	props *[]*bpparser.Property
}

func (p Properties) Prop(name string) (*bpparser.Property, bool) {
	for _, prop := range *p.props {
		if name == prop.Name.Name {
			return prop, true
		}
	}
	return nil, false
}

func (p Properties) AppendToProp(name string, src *bpparser.Property) {
	if d, ok := p.Prop(name); ok {
		d.Value = appendValueToValue(d.Value, src.Value)
	} else {
		prop := copyBPProperty(src)
		prop.Name.Name = name
		*p.props = append(*p.props, prop)
	}
}

func (p Properties) DeleteProp(name string) {
	for i, prop := range *p.props {
		if prop.Name.Name == name {
			*p.props = append((*p.props)[0:i], (*p.props)[i+1:]...)
			return
		}
	}
}
