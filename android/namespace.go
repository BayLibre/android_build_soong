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
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/google/blueprint"
)

func init() {
	registerNamespaceBuildComponents(InitRegistrationContext)
}

func registerNamespaceBuildComponents(ctx RegistrationContext) {
	ctx.RegisterModuleType("soong_namespace", NamespaceFactory)
}

type ExportModulesFromNamespace interface {
	ExportModulesFromNamespace() []string
}

func namespaceBoundaryEnforcerSingleton() Singleton {
	return &namespaceBoundaryEnforcer{}
}

type namespaceBoundaryEnforcer struct{}

const ENFORCEMENT_ENV = "SOONG_API_BOUNDARY_ENFORCEMENT"

func getApiBoundaryEnforcement(config Config) (string, error) {
	enforcement := config.GetenvWithDefault(ENFORCEMENT_ENV, "warn")
	enforcement = strings.ToLower(enforcement)
	switch enforcement {
	case "off", "warn", "error":
		return enforcement, nil
	}

	return enforcement, fmt.Errorf("Environment variable %q has invalid value %q", ENFORCEMENT_ENV, enforcement)
}

func EnableApiBoundaryEnforcement(config Config) bool {
	enforcement, err := getApiBoundaryEnforcement(config)
	if err != nil {
		panic(err)
	}
	return enforcement != "off"
}

func (*namespaceBoundaryEnforcer) GenerateBuildActions(ctx SingletonContext) {
	// Check the environment variable to see what  enforcement should be done.
	enforcement, err := getApiBoundaryEnforcement(ctx.Config())
	if err != nil {
		ctx.Errorf("%s", err)
		return
	}

	var treatViolationsAsError bool
	switch enforcement {
	case "off":
		return

	case "warn":
		treatViolationsAsError = false

	case "error":
		treatViolationsAsError = true
	}

	r := getNameResolver(ctx.Config())

	getNamespace := func(module Module) *Namespace {
		// Get the namespace for the module. If no such namespace exists then it is an error.
		dir := ctx.ModuleDir(module)
		return r.findNamespace(dir)
	}

	// First populate all namespaces with the set of modules they export.
	ctx.VisitAllModules(func(module Module) {
		if exporter, ok := module.(ExportModulesFromNamespace); ok {
			// Get the namespace for the module. If no such namespace exists then it is an error.
			namespace := getNamespace(module)
			if namespace == nil {
				return
			}

			if !namespace.protected {
				// Ignore unprotected namespaces.
				return
			}

			if !namespace.active {
				// Ignore inactive namespaces as they only export modules explicitly listed.
				return
			}

			if namespace == r.rootNamespace {
				// Ignore exports in the root namespace as everything is exported from there.
				return
			}

			exported := exporter.ExportModulesFromNamespace()

			// Add exports to the namespace.
			addExportsToNamespace := func(namespace *Namespace, exported []string) {
				// Get a map of exported names for the namespace, creating one if necessary.
				namespaceExports := namespace.exportedModules
				if namespaceExports == nil {
					namespaceExports = map[string]bool{}
					namespace.exportedModules = namespaceExports
				}

				// Add all the exported names to the namespace's exports.
				for _, export := range exported {
					namespaceExports[export] = true
				}
			}

			// Add exports to the namespaces.
			addExportsToNamespace(namespace, exported)
		}
	})

	qualifiedPath := func(module Module) string {
		return fmt.Sprintf("//%s:%s", ctx.ModuleDir(module), ctx.ModuleName(module))
	}

	describeModule := func(module Module, namespace *Namespace) string {
		var namespaceString string
		if namespace == r.rootNamespace {
			namespaceString = "root namespace"
		} else {
			namespaceString = fmt.Sprintf("namespace(%s)", namespace)
		}
		return fmt.Sprintf("%s - %s", qualifiedPath(module), namespaceString)
	}

	// The violations for a specific unexported module. Consisting of a map from the unexported module
	// to the modules that depend upon it.
	type violations map[string]map[string]bool
	violationsByNamespace := map[*Namespace]violations{}

	// Now check all dependencies to see if they violate a namespace boundary.
	// * Visit all modules
	// * For each module
	//   * Visit each direct dependency.
	//   * For each direct dependency
	//     * Check whether it crosses a namespace boundary.
	//     * If it does then check whether the destination namespace is protected, i.e. has exports.
	//     * If it does then check to make sure that the dependency module is exported.
	//     * If it is not the a violation has occurred.
	ctx.VisitAllModules(func(module Module) {
		fromNamespace := getNamespace(module)
		if fromNamespace == nil {
			return
		}

		ctx.VisitDirectDeps(module, func(dep Module) {
			toNamespace := getNamespace(dep)
			if toNamespace == nil {
				return
			}

			// If the dependencies is within a namespace then the dependency is not violating the
			// boundary.
			if fromNamespace == toNamespace {
				return
			}

			// TODO: Add support for accessing the dep tag and use that directly.
			// Ignore dependencies from sources onto prebuilts.
			if module.Name() == RemoveOptionalPrebuiltPrefix(dep.Name()) {
				if _, ok := dep.(PrebuiltInterface); ok {
					return
				}
			}

			// Ignore dependencies from a prebuilt java_sdk_library_import to the source
			// java_sdk_library's implementation library. It only matters when the prebuilt is preferred
			// and doesn't expose any more unexported modules than are already exposed by the source
			// modules.
			if module.Name()+".impl" == PrebuiltNameFromSource(dep.Name()) {
				return
			}

			if exports := toNamespace.exportedModules; exports != nil {
				if exports[ctx.ModuleName(dep)] {
					// The module is exported so ignore it.
					return
				}

				// Get the map of violations for the target namespace, creating one if necessary.
				violationsOfNamespace := violationsByNamespace[toNamespace]
				if violationsOfNamespace == nil {
					violationsOfNamespace = violations{}
					violationsByNamespace[toNamespace] = violationsOfNamespace
				}

				toModule := qualifiedPath(dep)

				violationsOfProtectedModule := violationsOfNamespace[toModule]
				if violationsOfProtectedModule == nil {
					violationsOfProtectedModule = map[string]bool{}
					violationsOfNamespace[toModule] = violationsOfProtectedModule
				}

				fromModule := describeModule(module, fromNamespace)
				violationsOfProtectedModule[fromModule] = true
			}
		})
	})

	if len(violationsByNamespace) > 0 {
		// Iterate over all the namespaces and report violations.
		for _, namespace := range r.sortedNamespaces.sortedItems() {
			violationsOfNamespace := violationsByNamespace[namespace]
			if violationsOfNamespace == nil {
				continue
			}

			fmt.Printf("Namespace %q\n", namespace.Path)
			for _, additionalPath := range namespace.additionalPaths {
				fmt.Printf("  including %q\n", additionalPath)
			}
			fmt.Printf("\n")

			fmt.Printf("  Exports\n")
			exports := namespace.exportedModules
			for _, export := range SortedStringKeys(exports) {
				fmt.Printf("    %s\n", export)
			}
			fmt.Printf("\n")

			fmt.Printf("  Violations:\n")
			for _, protectedModule := range SortedStringKeys(violationsOfNamespace) {
				fmt.Printf("    %s\n", protectedModule)
				violationsOfProtectedModule := violationsOfNamespace[protectedModule]
				for _, violatingModule := range SortedStringKeys(violationsOfProtectedModule) {
					fmt.Printf("      ^-- %s\n", violatingModule)
				}
				fmt.Printf("\n")
			}
			fmt.Printf("\n")
		}

		if treatViolationsAsError {
			ctx.Errorf("API boundary violations detected, please refer to previous output for details")
		}
	}
}

func getNameResolver(config Config) *NameResolver {
	r := config.Once(nameInterfaceKey, func() interface{} {
		panic("name interface not configured")
	}).(*NameResolver)
	return r
}

// threadsafe sorted list
type sortedNamespaces struct {
	lock   sync.Mutex
	items  []*Namespace
	sorted bool
}

func (s *sortedNamespaces) add(namespace *Namespace) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.sorted {
		panic("It is not supported to call sortedNamespaces.add() after sortedNamespaces.sortedItems()")
	}
	s.items = append(s.items, namespace)
}

func (s *sortedNamespaces) sortedItems() []*Namespace {
	s.lock.Lock()
	defer s.lock.Unlock()
	if !s.sorted {
		less := func(i int, j int) bool {
			return s.items[i].Path < s.items[j].Path
		}
		sort.Slice(s.items, less)
		s.sorted = true

		// Mark each namespace with a unique id.
		for i, ns := range s.items {
			ns.id = strconv.Itoa(i)
		}
	}
	return s.items
}

func (s *sortedNamespaces) index(namespace *Namespace) int {
	for i, candidate := range s.sortedItems() {
		if namespace == candidate {
			return i
		}
	}
	return -1
}

// A NameResolver implements blueprint.NameInterface, and implements the logic to
// find a module from namespaces based on a query string.
// A query string can be a module name or can be "//namespace_path:module_path"
type NameResolver struct {
	rootNamespace *Namespace

	// id counter for atomic.AddInt32
	nextNamespaceId int32

	// All namespaces, without duplicates.
	sortedNamespaces sortedNamespaces

	// Map from dir to namespace. Will have duplicates if two dirs are part of the same namespace.
	namespacesByDir sync.Map // if generics were supported, this would be sync.Map[string]*Namespace

	// func telling whether to export a namespace to Kati
	namespaceExportFilter func(*Namespace) bool
}

type ProtectedNamespaceConfig struct {
	// The root directories belonging to the namespace.
	Paths []string

	// True if this namespace is active or not.
	//
	// An active namespace will export all modules, both modules implicitly exported by modules within
	// the namespace that implement ExportModulesFromNamespace, and modules explicitly listed in
	// AdditionalMakeExports and AdditionalSoongExports.
	//
	// An inactive namespace will only export modules explicitly listed in AdditionalMakeExports and
	// AdditionalSoongExports.
	Active bool

	// Additional exports that have to be exported from the namespace to other Soong namespaces
	// and Make.
	AdditionalSoongExports []string

	// Additional exports that have to be exported from the namespace to make.
	AdditionalMakeExports []string

	ExcludedNamespaces []*ProtectedNamespaceConfig
}

func (c *ProtectedNamespaceConfig) Exclude(other *ProtectedNamespaceConfig) {
	c.ExcludedNamespaces = append(c.ExcludedNamespaces, other)
}

func NewNameResolver(namespaceExportFilter func(*Namespace) bool, protectedNamespaces []*ProtectedNamespaceConfig) *NameResolver {
	r := &NameResolver{
		namespacesByDir:       sync.Map{},
		namespaceExportFilter: namespaceExportFilter,
	}
	rootNamespace := r.newNamespace(".")
	r.rootNamespace = rootNamespace
	rootNamespace.visibleNamespaces = []*Namespace{rootNamespace}
	if err := r.addNamespace(rootNamespace); err != nil {
		panic(fmt.Sprintf("Could not add root namespace: %s", err))
	}

	allNamespaces := []*Namespace{}
	namespacesByConfig := map[*ProtectedNamespaceConfig]*Namespace{}
	for _, protectedNamespace := range protectedNamespaces {
		paths := protectedNamespace.Paths
		namespace := r.newNamespace(paths[0], paths[1:]...)

		namespacesByConfig[protectedNamespace] = namespace

		namespace.protected = true
		namespace.active = protectedNamespace.Active
		namespace.exportedModules = map[string]bool{}
		for _, additionalExport := range protectedNamespace.AdditionalSoongExports {
			namespace.exportedModules[additionalExport] = true
		}
		for _, additionalExport := range protectedNamespace.AdditionalMakeExports {
			namespace.exportedModules[additionalExport] = true
		}
		allNamespaces = append(allNamespaces, namespace)

		if err := r.addNamespace(namespace); err != nil {
			panic(err)
		}

		// Resolve the imports in the namespaces.
		if err := r.FindNamespaceImports(namespace); err != nil {
			panic(err)
		}

		// Add the selected namespaces to the root namespace so that they will be visible to every
		// namespace.
		rootNamespace.visibleNamespaces = append(rootNamespace.visibleNamespaces, namespace)

		// Export the selected namespaces to Make.
		namespace.exportToKati = true
	}

	for _, protectedNamespace := range protectedNamespaces {
		ns := namespacesByConfig[protectedNamespace]
		ns.ignoreNamespacesVisibleThroughRoot = map[*Namespace]struct{}{}
		for _, excludedProtectedNamespace := range protectedNamespace.ExcludedNamespaces {
			excludedNs := namespacesByConfig[excludedProtectedNamespace]
			ns.ignoreNamespaceVisibleToRoot(excludedNs)
		}
	}

	return r
}

func (r *NameResolver) newNamespace(path string, additionalPaths ...string) *Namespace {
	namespace := NewNamespace(path, additionalPaths...)

	namespace.exportToKati = r.namespaceExportFilter(namespace)

	return namespace
}

func (r *NameResolver) addNewNamespaceForModule(module *NamespaceModule, path string) error {
	fileName := filepath.Base(path)
	if fileName != "Android.bp" {
		return errors.New("A namespace may only be declared in a file named Android.bp")
	}
	dir := filepath.Dir(path)

	namespace := r.newNamespace(dir)
	module.namespace = namespace
	module.resolver = r
	namespace.importedNamespaceNames = module.properties.Imports
	return r.addNamespace(namespace)
}

func (r *NameResolver) checkExistingNamespace(path string) error {
	existingNamespace, exists := r.namespaceAt(path)
	if exists {
		if existingNamespace.Path == path || InList(path, existingNamespace.additionalPaths) {
			return fmt.Errorf("namespace %v already exists", existingNamespace)
		} else {
			// It would probably confuse readers if namespaces were declared anywhere but
			// the top of the file, so we forbid declaring namespaces after anything else.
			return fmt.Errorf("a namespace must be the first module in the file")
		}
	}
	return nil
}

func (r *NameResolver) addNamespace(namespace *Namespace) (err error) {
	if err := r.checkExistingNamespace(namespace.Path); err != nil {
		return err
	}
	r.namespacesByDir.Store(namespace.Path, namespace)

	for _, additionalPath := range namespace.additionalPaths {
		if err := r.checkExistingNamespace(additionalPath); err != nil {
			return err
		}
		r.namespacesByDir.Store(additionalPath, namespace)
	}

	r.sortedNamespaces.add(namespace)

	return nil
}

// non-recursive check for namespace
func (r *NameResolver) namespaceAt(path string) (namespace *Namespace, found bool) {
	mapVal, found := r.namespacesByDir.Load(path)
	if !found {
		return nil, false
	}
	return mapVal.(*Namespace), true
}

// recursive search upward for a namespace
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

// A NamelessModule can never be looked up by name.  It must still implement Name(), but the return
// value doesn't have to be unique.
type NamelessModule interface {
	Nameless()
}

func (r *NameResolver) NewModule(ctx blueprint.NamespaceContext, moduleGroup blueprint.ModuleGroup, module blueprint.Module) (namespace blueprint.Namespace, errs []error) {
	// if this module is a namespace, then save it to our list of namespaces
	newNamespace, ok := module.(*NamespaceModule)
	if ok {
		err := r.addNewNamespaceForModule(newNamespace, ctx.ModulePath())
		if err != nil {
			return nil, []error{err}
		}
		return nil, nil
	}

	if _, ok := module.(NamelessModule); ok {
		return nil, nil
	}

	// if this module is not a namespace, then save it into the appropriate namespace
	ns := r.findNamespaceFromCtx(ctx)

	_, errs = ns.moduleContainer.NewModule(ctx, moduleGroup, module)
	if len(errs) > 0 {
		return nil, errs
	}

	amod, ok := module.(Module)
	if ok {
		// inform the module whether its namespace is one that we want to export to Make
		amod.base().commonProperties.NamespaceExportedToMake = ns.exportToKati
		amod.base().commonProperties.DebugName = module.Name()
	}

	return ns, nil
}

func (r *NameResolver) AllModules() []blueprint.ModuleGroup {
	childLists := [][]blueprint.ModuleGroup{}
	totalCount := 0
	for _, namespace := range r.sortedNamespaces.sortedItems() {
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

// parses a fully-qualified path (like "//namespace_path:module_name") into a namespace name and a
// module name
func (r *NameResolver) parseFullyQualifiedName(name string) (namespaceName string, moduleName string, ok bool) {
	if !strings.HasPrefix(name, "//") {
		return "", "", false
	}
	name = strings.TrimPrefix(name, "//")
	components := strings.Split(name, ":")
	if len(components) != 2 {
		return "", "", false
	}
	return components[0], components[1], true

}

func (r *NameResolver) getNamespacesToSearchForModule(sourceNamespace blueprint.Namespace) (searchOrder []*Namespace) {
	ns, ok := sourceNamespace.(*Namespace)
	if !ok || ns.visibleNamespaces == nil {
		// When handling dependencies before namespaceMutator, assume they are non-Soong Blueprint modules and give
		// access to all namespaces.
		return r.sortedNamespaces.sortedItems()
	}
	return ns.visibleNamespaces
}

func (r *NameResolver) ModuleFromName(name string, namespace blueprint.Namespace) (group blueprint.ModuleGroup, found bool) {
	fromNs := namespace.(*Namespace)

	// handle fully qualified references like "//namespace_path:module_name"
	nsName, moduleName, isAbs := r.parseFullyQualifiedName(name)
	if isAbs {
		namespace, found := r.namespaceAt(nsName)
		if !found {
			return blueprint.ModuleGroup{}, false
		}
		return namespace.findModuleByName(moduleName, fromNs)
	}
	for _, candidate := range r.getNamespacesToSearchForModule(namespace) {
		group, found = candidate.findModuleByName(name, fromNs)
		if found {
			return group, true
		}
	}

	if namespace != r.rootNamespace {
		// search the root namespace and all namespaces that are visible to it last.
		for _, candidate := range r.rootNamespace.visibleNamespaces {
			if _, ok := fromNs.ignoreNamespacesVisibleThroughRoot[candidate]; ok {
				continue
			}
			group, found = candidate.findModuleByName(name, fromNs)
			if found {
				return group, true
			}
		}
	}

	return blueprint.ModuleGroup{}, false

}

func (r *NameResolver) Rename(oldName string, newName string, namespace blueprint.Namespace) []error {
	return namespace.(*Namespace).moduleContainer.Rename(oldName, newName, namespace)
}

// resolve each element of namespace.importedNamespaceNames and put the result in namespace.visibleNamespaces
func (r *NameResolver) FindNamespaceImports(namespace *Namespace) (err error) {
	namespace.visibleNamespaces = make([]*Namespace, 0, 2+len(namespace.importedNamespaceNames))
	// search itself first
	namespace.visibleNamespaces = append(namespace.visibleNamespaces, namespace)
	// search its imports next
	for _, name := range namespace.importedNamespaceNames {
		imp, ok := r.namespaceAt(name)
		if !ok {
			return fmt.Errorf("namespace %v does not exist", name)
		}
		namespace.ignoreNamespaceVisibleToRoot(imp)
		namespace.visibleNamespaces = append(namespace.visibleNamespaces, imp)
	}

	return nil
}

func (r *NameResolver) chooseId(_ *Namespace) {
	// Make sure that all namespaces have an id.
	r.sortedNamespaces.sortedItems()
}

func (r *NameResolver) MissingDependencyError(depender string, dependerNamespace blueprint.Namespace, depName string) (err error) {
	text := fmt.Sprintf("%q depends on undefined module %q", depender, depName)

	_, _, isAbs := r.parseFullyQualifiedName(depName)
	if isAbs {
		// if the user gave a fully-qualified name, we don't need to look for other
		// modules that they might have been referring to
		return fmt.Errorf(text)
	}

	// determine which namespaces the module can be found in
	foundInNamespaces := []string{}
	for _, namespace := range r.sortedNamespaces.sortedItems() {
		_, found := namespace.moduleContainer.ModuleFromName(depName, nil)
		if found {
			foundInNamespaces = append(foundInNamespaces, namespace.String())
		}
	}
	if len(foundInNamespaces) > 0 {
		// determine which namespaces are visible to dependerNamespace
		dependerNs := dependerNamespace.(*Namespace)
		importedNames := r.getNamesOfVisibleNamespaces(dependerNs)
		text += fmt.Sprintf("\nModule %q is defined in namespace %q which can read these %v namespaces: %q", depender, dependerNs.Path, len(importedNames), importedNames)
		text += fmt.Sprintf("\nModule %q can be found in these namespaces: %q", depName, foundInNamespaces)
	}

	return fmt.Errorf(text)
}

func (r *NameResolver) getNamesOfVisibleNamespaces(namespace *Namespace) []string {
	importedNames := []string{}
	for _, ns := range namespace.visibleNamespaces {
		importedNames = append(importedNames, ns.String())
	}
	// Every namespace can see the root namespace and all the namespaces visible to it. Except those
	// that is specifically excludes.
	for _, ns := range r.rootNamespace.visibleNamespaces {
		if _, ok := namespace.ignoreNamespacesVisibleThroughRoot[ns]; ok {
			continue
		}
		importedNames = append(importedNames, ns.String())
	}
	return importedNames
}

func (r *NameResolver) GetNamespace(ctx blueprint.NamespaceContext) blueprint.Namespace {
	return r.findNamespaceFromCtx(ctx)
}

func (r *NameResolver) findNamespaceFromCtx(ctx blueprint.NamespaceContext) *Namespace {
	return r.findNamespace(filepath.Dir(ctx.ModulePath()))
}

func (r *NameResolver) UniqueName(ctx blueprint.NamespaceContext, name string) (unique string) {
	prefix := r.findNamespaceFromCtx(ctx).id
	if prefix != "" {
		prefix = prefix + "-"
	}
	return prefix + name
}

var _ blueprint.NameInterface = (*NameResolver)(nil)

type Namespace struct {
	blueprint.NamespaceMarker
	Path string

	// names of namespaces listed as imports by this namespace
	importedNamespaceNames []string

	// The namespaces that should be searched when a module in this namespace declares a dependency
	// - does not include the root namespace as that is always searched.
	// - does include itself.
	visibleNamespaces []*Namespace

	// The namespaces that are visible to the root namespace but which must not be searched when
	// resolving a dependency in this namespace. This includes namespaces that have been explicitly
	// imported (to avoid searching them twice) as well as namespaces that have been explicitly
	// excluded.
	ignoreNamespacesVisibleThroughRoot map[*Namespace]struct{}

	id string

	exportToKati bool

	moduleContainer blueprint.NameInterface

	// The additional paths that are part of this namespace.
	additionalPaths []string

	exportedModules map[string]bool

	protected bool
	active    bool
}

func NewNamespace(path string, additionalPaths ...string) *Namespace {
	return &Namespace{
		Path:            path,
		additionalPaths: additionalPaths,
		moduleContainer: blueprint.NewSimpleNameInterface(),
	}
}

var _ blueprint.Namespace = (*Namespace)(nil)

func (n *Namespace) ignoreNamespaceVisibleToRoot(other *Namespace) {
	if n.ignoreNamespacesVisibleThroughRoot == nil {
		n.ignoreNamespacesVisibleThroughRoot = map[*Namespace]struct{}{}
	}
	n.ignoreNamespacesVisibleThroughRoot[other] = struct{}{}
}

func (n *Namespace) couldExportModule(name string, fromNamespace *Namespace) bool {
	return fromNamespace == n || !n.protected || n.active || n.exportedModules[name]
}

func (n *Namespace) findModuleByName(name string, fromNamespace *Namespace) (blueprint.ModuleGroup, bool) {
	if !n.couldExportModule(name, fromNamespace) {
		return blueprint.ModuleGroup{}, false
	}
	container := n.moduleContainer
	return container.ModuleFromName(name, nil)
}

func (n *Namespace) String() string {
	if len(n.additionalPaths) == 0 {
		return n.Path
	}

	return fmt.Sprintf("%s(+%s)", n.Path, strings.Join(n.additionalPaths, ","))
}

type namespaceProperties struct {
	// a list of namespaces that contain modules that will be referenced
	// by modules in this namespace.
	Imports []string `android:"path"`
}

type NamespaceModule struct {
	ModuleBase

	namespace *Namespace
	resolver  *NameResolver

	properties namespaceProperties
}

func (n *NamespaceModule) GenerateAndroidBuildActions(ctx ModuleContext) {
}

func (n *NamespaceModule) GenerateBuildActions(ctx blueprint.ModuleContext) {
}

func (n *NamespaceModule) Name() (name string) {
	return *n.nameProperties.Name
}

// soong_namespace provides a scope to modules in an Android.bp file to prevent
// module name conflicts with other defined modules in different Android.bp
// files. Once soong_namespace has been defined in an Android.bp file, the
// namespacing is applied to all modules that follow the soong_namespace in
// the current Android.bp file, as well as modules defined in Android.bp files
// in subdirectories. An Android.bp file in a subdirectory can define its own
// soong_namespace which is applied to all its modules and as well as modules
// defined in subdirectories Android.bp files. Modules in a soong_namespace are
// visible to Make by listing the namespace path in PRODUCT_SOONG_NAMESPACES
// make variable in a makefile.
func NamespaceFactory() Module {
	module := &NamespaceModule{}

	name := "soong_namespace"
	module.nameProperties.Name = &name

	module.AddProperties(&module.properties)
	return module
}

func RegisterNamespaceMutator(ctx RegisterMutatorsContext) {
	ctx.BottomUp("namespace_deps", namespaceMutator).Parallel()
}

func namespaceMutator(ctx BottomUpMutatorContext) {
	module, ok := ctx.Module().(*NamespaceModule)
	if ok {
		err := module.resolver.FindNamespaceImports(module.namespace)
		if err != nil {
			ctx.ModuleErrorf(err.Error())
		}

		module.resolver.chooseId(module.namespace)
	}
}
