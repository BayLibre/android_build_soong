// Copyright 2016 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package android

import (
	"fmt"
	"reflect"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"
)

// This file implements hooks that external module types can use to inject logic into existing
// module types.  Each hook takes an interface as a parameter so that new methods can be added
// to the interface without breaking existing module types.

// Load hooks are run after the module's properties have been filled from the blueprint file, but
// before the module has been split into architecture variants, and before defaults modules have
// been applied.
type LoadHookContext interface {
	EarlyModuleContext

	AppendProperties(...interface{})
	PrependProperties(...interface{})

	// Set each module property that matches one of the supplied properties to the matching property's
	// value. If the module property has already been set (i.e. is anything other than the default value
	// for the property type) from the module definition then fail with an error.
	FixProperties(...interface{})
	CreateModule(ModuleFactory, ...interface{}) Module

	registerScopedModuleType(name string, factory blueprint.ModuleFactory)
	moduleFactories() map[string]blueprint.ModuleFactory
}

func AddLoadHook(m blueprint.Module, hook func(LoadHookContext)) {
	blueprint.AddLoadHook(m, func(ctx blueprint.LoadHookContext) {
		actx := &loadHookContext{
			earlyModuleContext: m.(Module).base().earlyModuleContextFactory(ctx),
			bp:                 ctx,
		}
		hook(actx)
	})
}

type loadHookContext struct {
	earlyModuleContext
	bp     blueprint.LoadHookContext
	module Module
}

func (l *loadHookContext) moduleFactories() map[string]blueprint.ModuleFactory {
	return l.bp.ModuleFactories()
}

func (l *loadHookContext) extendMatchingProperties(props []interface{}, filter proptools.ExtendPropertyFilterFunc, order proptools.ExtendPropertyOrderFunc) {
	for _, p := range props {
		err := proptools.ExtendMatchingProperties(l.Module().base().customizableProperties,
			p, filter, order)
		if err != nil {
			if propertyErr, ok := err.(*proptools.ExtendPropertyError); ok {
				l.PropertyErrorf(propertyErr.Property, "%s", propertyErr.Err.Error())
			} else {
				panic(err)
			}
		}
	}
}

func (l *loadHookContext) AppendProperties(props ...interface{}) {
	l.extendMatchingProperties(props, nil, proptools.OrderAppend)
}

func (l *loadHookContext) PrependProperties(props ...interface{}) {
	l.extendMatchingProperties(props, nil, proptools.OrderPrepend)
}

func checkDestinationNotSet(property string, dstField, srcField reflect.StructField, dstValue, srcValue interface{}) (bool, error) {
	// If the destination matches the zero value then it is fine.
	zeroValue := reflect.Zero(dstField.Type).Interface()
	if reflect.DeepEqual(dstValue, zeroValue) {
		return true, nil
	}

	if reflect.DeepEqual(dstValue, srcValue) {
		srcValue = referencePossiblePointer(srcValue)
		return false, fmt.Errorf("is fixed to %q so setting it to the same value is redundant", srcValue)
	} else {
		srcValue = referencePossiblePointer(srcValue)
		dstValue = referencePossiblePointer(dstValue)
		return false, fmt.Errorf("is fixed to %q and cannot be set to %q", srcValue, dstValue)
	}
}

func referencePossiblePointer(possiblePtr interface{}) interface{} {
	value := reflect.ValueOf(possiblePtr)
	if value.Kind() == reflect.Ptr {
		if value.IsNil() {
			// Cannot deference a nil pointer so use the zero value instead.
			return reflect.Zero(value.Type().Elem()).Interface()
		}
		value = value.Elem()
		return value.Interface()
	}

	return possiblePtr
}

func (l *loadHookContext) FixProperties(props ...interface{}) {
	l.extendMatchingProperties(props, checkDestinationNotSet, proptools.OrderReplace)
}

func (l *loadHookContext) CreateModule(factory ModuleFactory, props ...interface{}) Module {
	inherited := []interface{}{&l.Module().base().commonProperties}
	module := l.bp.CreateModule(ModuleFactoryAdaptor(factory), append(inherited, props...)...).(Module)

	if l.Module().base().variableProperties != nil && module.base().variableProperties != nil {
		src := l.Module().base().variableProperties
		dst := []interface{}{
			module.base().variableProperties,
			// Put an empty copy of the src properties into dst so that properties in src that are not in dst
			// don't cause a "failed to find property to extend" error.
			proptools.CloneEmptyProperties(reflect.ValueOf(src)).Interface(),
		}
		err := proptools.AppendMatchingProperties(dst, src, nil)
		if err != nil {
			panic(err)
		}
	}

	return module
}

func (l *loadHookContext) registerScopedModuleType(name string, factory blueprint.ModuleFactory) {
	l.bp.RegisterScopedModuleType(name, factory)
}

type InstallHookContext interface {
	ModuleContext
	Path() InstallPath
	Symlink() bool
}

// Install hooks are run after a module creates a rule to install a file or symlink.
// The installed path is available from InstallHookContext.Path(), and
// InstallHookContext.Symlink() will be true if it was a symlink.
func AddInstallHook(m blueprint.Module, hook func(InstallHookContext)) {
	h := &m.(Module).base().hooks
	h.install = append(h.install, hook)
}

type installHookContext struct {
	ModuleContext
	path    InstallPath
	symlink bool
}

func (x *installHookContext) Path() InstallPath {
	return x.path
}

func (x *installHookContext) Symlink() bool {
	return x.symlink
}

func (x *hooks) runInstallHooks(ctx ModuleContext, path InstallPath, symlink bool) {
	if len(x.install) > 0 {
		mctx := &installHookContext{
			ModuleContext: ctx,
			path:          path,
			symlink:       symlink,
		}
		for _, x := range x.install {
			x(mctx)
			if mctx.Failed() {
				return
			}
		}
	}
}

type hooks struct {
	install []func(InstallHookContext)
}
