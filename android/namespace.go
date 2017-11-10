// Copyright 2017 Google Inc. All rights reserved.
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
	"path/filepath"
	"strings"
	"sync"

	"github.com/google/blueprint"
)

// This file implements namespaces
const (
	namespacePrefix = "//"
	modulePrefix    = ":"
)

func init() {
	RegisterModuleType("soong_namespace", NamespaceFactory)
}

// A NameResolver implements blueprint.NameInterface, and implements the logic to
// find a module from namespaces based on a query string.
// A query string can be a module name or can be be "//namespace_path:module_path"
type NameResolver struct {
	rootNamespace *Namespace
	allNamespaces []*Namespace
	namespaceLock sync.Mutex

	namespacesByDir sync.Map // if generics were supported, this would be sync.Map[string]*Namespace

	namespaceExportFilter func(*Namespace) bool
}

func NewNameResolver(namespaceExportFilter func(*Namespace) bool) *NameResolver {
	namespacesByDir := sync.Map{}

	r := &NameResolver{
		namespacesByDir:       namespacesByDir,
		namespaceExportFilter: namespaceExportFilter,
	}
	r.rootNamespace = r.newNamespace(".")
	r.addNamespace(r.rootNamespace)

	return r
}

func (r *NameResolver) newNamespace(path string) *Namespace {
	namespace := NewNamespace(path)
	namespace.exportToMake = r.namespaceExportFilter(namespace)
	return namespace
}

func (r *NameResolver) addNewNamespaceForModule(module *NamespaceModule, dir string) error {
	namespace := r.newNamespace(dir)
	module.namespace = namespace
	module.resolver = r
	namespace.importedNamespaceNames = module.properties.Imports
	return r.addNamespace(namespace)
}

func (r *NameResolver) addNamespace(namespace *Namespace) (err error) {
	r.namespaceLock.Lock()
	defer r.namespaceLock.Unlock()

	r.allNamespaces = append(r.allNamespaces, namespace)
	existingNamespace, exists := r.namespaceAt(namespace.Path)
	if exists {
		if existingNamespace.Path == namespace.Path {
			return fmt.Errorf("namespace %v already exists", namespace.Path)
		} else {
			// It would probably confuse readers if namespaces were declared anywhere but
			// the top of the file, so we forbid declaring namespaces after anything else.
			return fmt.Errorf("a namespace must be the first module in the file")
		}
	}
	r.namespacesByDir.Store(namespace.Path, namespace)
	return nil
}

// nonrecursive check for namespace
func (r *NameResolver) namespaceAt(path string) (namespace *Namespace, found bool) {
	mapVal, found := r.namespacesByDir.Load(path)
	if !found {
		return nil, false
	}
	return mapVal.(*Namespace), true
}

// recursive search upwward for a namespace
func (r *NameResolver) findNamespace(path string) (namespace *Namespace) {
	namespace, found := r.namespaceAt(path)
	if found {
		return namespace
	}
	parentDir := filepath.Dir(path)
	if parentDir == path {
		return nil
	}
	namespace = r.findNamespace(parentDir)
	r.namespacesByDir.Store(path, namespace)
	return namespace
}

func (r *NameResolver) NewModule(ctx blueprint.NamespaceContext, moduleGroup blueprint.ModuleGroup, module blueprint.Module) (namespace blueprint.Namespace, errs []error) {
	// if this module is a namespace, then save it to our list of namespaces
	newNamespace, ok := module.(*NamespaceModule)
	if ok {
		err := r.addNewNamespaceForModule(newNamespace, ctx.ModuleDir())
		if err != nil {
			return blueprint.Namespace{}, []error{err}
		}
		return blueprint.Namespace{}, nil
	}

	// if this module is not a namespace, then save it into the appropriate namespace
	ns := r.findNamespaceFromCtx(ctx)

	_, errs = ns.moduleContainer.NewModule(ctx, moduleGroup, module)
	if len(errs) > 0 {
		return blueprint.Namespace{}, errs
	}

	return blueprint.Namespace{ns}, nil
}

func (r *NameResolver) AllModules() []blueprint.ModuleGroup {
	childLists := [][]blueprint.ModuleGroup{}
	totalCount := 0
	for _, namespace := range r.allNamespaces {
		newModules := namespace.moduleContainer.AllModules()
		totalCount += len(newModules)
		childLists = append(childLists, newModules)
	}

	allModules := make([]blueprint.ModuleGroup, 0, totalCount)
	for _, childList := range childLists {
		allModules = append(allModules, childList...)
	}
	return allModules
}

// parses an absolute path (like "//namespace_path:module_name") into a namespace name and modulename
func (r *NameResolver) parseAbs(name string) (namespaceName string, moduleName string, ok bool) {
	if !strings.HasPrefix(name, namespacePrefix) {
		return "", "", false
	}
	name = strings.Replace(name, namespacePrefix, "", 1)
	components := strings.Split(name, modulePrefix)
	if len(components) != 2 {
		return "", "", false
	}
	return components[0], components[1], true

}

func (r *NameResolver) ModuleFromName(name string, namespace blueprint.Namespace) (group blueprint.ModuleGroup, found bool) {
	// handle fully qualified references like "//namespace_path:module_name"
	nsName, moduleName, isAbs := r.parseAbs(name)
	if isAbs {
		namespace, found := r.namespaceAt(nsName)
		if !found {
			return blueprint.ModuleGroup{}, false
		}
		container := namespace.moduleContainer
		return container.ModuleFromName(moduleName, blueprint.Namespace{})
	}
	// try a lookup in the same namespace
	ns, valid := namespace.Impl.(*Namespace)
	if !valid {
		panic(fmt.Sprintf("Illegal namespace %#v (not of type *Namespace) given to search for module %v", namespace, name))
	}

	if ns != nil {
		// check the namespaces imported by the given namespace
		group, found = ns.moduleContainer.ModuleFromName(name, blueprint.Namespace{})
		if found {
			return group, true
		}
		// try a lookup in the imported namespace
		for _, importedNamespace := range ns.importedNamespaces {
			group, found := importedNamespace.moduleContainer.ModuleFromName(name, blueprint.Namespace{})
			if found {
				return group, true
			}
		}
	}
	// check the root namespace
	return r.rootNamespace.moduleContainer.ModuleFromName(name, blueprint.Namespace{})
}

func (r *NameResolver) Rename(oldName string, newName string, namespace blueprint.Namespace) []error {
	oldNs := r.findNamespace(oldName)
	newNs := r.findNamespace(newName)
	if oldNs != newNs {
		return []error{fmt.Errorf("cannot rename %v to %v because the destination is outside namespace %v", oldName, newName, oldNs.Path)}
	}

	oldName, err := filepath.Rel(oldNs.Path, oldName)
	if err != nil {
		panic(err)
	}
	newName, err = filepath.Rel(newNs.Path, newName)
	if err != nil {
		panic(err)
	}

	return oldNs.moduleContainer.Rename(oldName, newName, blueprint.Namespace{})
}

// resolve each element of namespace.importedNamespaceNames and put the result in namespace.importedNamespaces
func (r *NameResolver) FindNamespaceImports(namespace *Namespace) (err error) {
	namespace.importedNamespaces = make([]*Namespace, 0, len(namespace.importedNamespaceNames))
	for _, name := range namespace.importedNamespaceNames {
		imp, ok := r.namespaceAt(name)
		if !ok {
			return fmt.Errorf("namespace %v does not exist", name)
		}
		namespace.importedNamespaces = append(namespace.importedNamespaces, imp)
	}
	return nil
}

func (r *NameResolver) MissingDependencyError(depender string, depName string) (err error) {
	namespaces := []string{}
	for _, namespace := range r.allNamespaces {
		_, found := namespace.moduleContainer.ModuleFromName(depName, blueprint.Namespace{})
		if found {
			namespaces = append(namespaces, namespacePrefix+namespace.Path)
		}
	}
	text := fmt.Sprintf("%q depends on undefined module %q", depender, depName)
	if len(namespaces) > 0 {
		text += fmt.Sprintf("\nCan be found in these namespaces: %s", namespaces)
	}
	return fmt.Errorf(text)
}

func (r *NameResolver) GetNamespace(ctx blueprint.NamespaceContext) blueprint.Namespace {
	return blueprint.Namespace{r.findNamespaceFromCtx(ctx)}
}

func (r *NameResolver) findNamespaceFromCtx(ctx blueprint.NamespaceContext) *Namespace {
	return r.findNamespace(ctx.ModuleDir())
}

var _ blueprint.NameInterface = (*NameResolver)(nil)

type Namespace struct {
	Path                   string
	importedNamespaceNames []string
	importedNamespaces     []*Namespace

	exportToMake bool

	moduleContainer blueprint.NameInterface
}

func NewNamespace(path string) *Namespace {
	return &Namespace{Path: path, moduleContainer: blueprint.NewSimpleNameInterface()}
}

type NamespaceModule struct {
	ModuleBase

	namespace *Namespace
	resolver  *NameResolver

	properties struct {
		Imports []string
	}
}

func (n *NamespaceModule) DepsMutator(context BottomUpMutatorContext) {
}

func (n *NamespaceModule) GenerateAndroidBuildActions(ctx ModuleContext) {
}

func (n *NamespaceModule) GenerateBuildActions(ctx blueprint.ModuleContext) {
}

func (n *NamespaceModule) Name() (name string) {
	return *n.nameProperties.Name
}

func NamespaceFactory() Module {
	module := &NamespaceModule{}

	name := "soong_namespace"
	module.nameProperties.Name = &name

	module.AddProperties(&module.properties)
	return module
}

func RegisterNamespaceMutator(ctx RegisterMutatorsContext) {
	ctx.BottomUp("namespace_deps", namespaceDeps)
}

func namespaceDeps(ctx BottomUpMutatorContext) {
	module, ok := ctx.Module().(*NamespaceModule)
	if ok {
		err := module.resolver.FindNamespaceImports(module.namespace)
		if err != nil {
			ctx.ModuleErrorf(err.Error())
		}
	}
}
