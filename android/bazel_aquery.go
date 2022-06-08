// Copyright 2020 Google Inc. All rights reserved.
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
	"encoding/json"
	"fmt"
	"log"
	os2 "os"
	"path/filepath"
	"strings"
)

/*
 * Unmarshalled aquery --output=jsonproto structs that we process.
 */

// actionGraphContainer contains relevant portions of Bazel's aquery proto, ActionGraphContainer.
// An aquery response from Bazel contains a single ActionGraphContainer proto.
type actionGraphContainer struct {
	Artifacts     []artifact
	Actions       []action
	DepSetOfFiles []depSetOfFiles
	PathFragments []pathFragment
}

type artifactId int
type depsetId int
type pathFragmentId int

// artifact contains relevant portions of Bazel's aquery proto, Artifact.
// Represents a single artifact, whether it's a source file or a derived output file.
type artifact struct {
	Id             artifactId
	PathFragmentId pathFragmentId
}

type pathFragment struct {
	Id       pathFragmentId
	Label    string
	ParentId pathFragmentId
}

// keyValuePair represents Bazel's aquery proto, keyValuePair.
type keyValuePair struct {
	Key   string
	Value string
}

// depSetOfFiles contains relevant portions of Bazel's aquery proto, DepSetOfFiles.
// Represents a data structure containing one or more files. Depsets in Bazel are an efficient
// data structure for storing large numbers of file paths.
type depSetOfFiles struct {
	Id                  depsetId
	DirectArtifactIds   []artifactId
	TransitiveDepSetIds []depsetId
}

// action contains relevant portions of Bazel's aquery proto, Action.
// Represents a single command line invocation in the Bazel build graph.
type action struct {
	Arguments            []string
	EnvironmentVariables []keyValuePair
	InputDepSetIds       []depsetId
	Mnemonic             string
	OutputIds            []artifactId
	TemplateContent      string
	Substitutions        []keyValuePair
	FileContents         string
}

/*
 * Deserialized bazel actions.
 */

type bazelPath struct {
	fragment string
	parent   *bazelPath
}

func (bp bazelPath) String() string {
	if bp.parent == nil {
		return bp.fragment
	}
	return filepath.Join(bp.parent.String(), bp.fragment)
}

type bazelArtifact interface {
}

type bazelFileArtifact struct {
	file *bazelPath
}

func (fa bazelFileArtifact) String() string {
	if fa.file == nil {
		return ""
	}
	return fa.file.String()
}

type bazelSetArtifact struct {
	items []bazelArtifact
}

func (fs bazelSetArtifact) String() string {
	var buf strings.Builder
	sep := "{"
	for _, item := range fs.items {
		buf.WriteString(sep)
		buf.WriteString(fmt.Sprintf("%s", item))
		sep = " "
	}
	buf.WriteString("}")
	return buf.String()
}

type bazelBadArtifact struct {
	id depsetId
}

func (ba bazelBadArtifact) String() string {
	return fmt.Sprintf("bad artifact #%d", ba.id)
}

type bazelAction interface {
	emitSoongRule(ctx SingletonContext)
}

type bazelSymlinkAction struct {
	from bazelArtifact
	to   bazelArtifact
}

func (a bazelSymlinkAction) String() string {
	return fmt.Sprintf("Symlink %s: %s", a.to, a.from)
}

func (a bazelSymlinkAction) emitSoongRule(ctx SingletonContext) {
	//TODO implement me
	panic("implement me")
}

type bazelTemplateExpandAction struct {
	to            *bazelFileArtifact
	template      string
	substitutions []keyValuePair
}

func (a bazelTemplateExpandAction) emitSoongRule(ctx SingletonContext) {
	//TODO implement me
	panic("implement me")
}

func (a bazelTemplateExpandAction) String() string {
	return fmt.Sprintf("TemplateExpand %s: %q[%v]", a.to, a.template, a.substitutions)
}

type bazelFileWriteAction struct {
	to       *bazelFileArtifact
	contents string
}

func (a bazelFileWriteAction) emitSoongRule(ctx SingletonContext) {
	//TODO implement me
	panic("implement me")
}

func (a bazelFileWriteAction) String() string {
	return fmt.Sprintf("FileWrite %s: %q", a.to, a.contents)
}

type bazelPythonZipperAction struct {
	outs      []*bazelFileArtifact
	ins       bazelSetArtifact
	arguments []string
	env       []keyValuePair
}

func (a bazelPythonZipperAction) emitSoongRule(ctx SingletonContext) {
	//TODO implement me
	panic("implement me")
}

func (a bazelPythonZipperAction) String() string {
	return fmt.Sprintf("PythonZipper %s : %s\n\tcmd: %s\n\tenv: %s",
		a.outs, a.ins, a.arguments, a.env)
}

type bazelGenericAction struct {
	arguments []string
	outs      []*bazelFileArtifact
	ins       bazelSetArtifact
	env       []keyValuePair
	mnenonic  string
}

func (a bazelGenericAction) emitSoongRule(ctx SingletonContext) {
	//TODO implement me
	panic("implement me")
}

func (a bazelGenericAction) String() string {
	return fmt.Sprintf("GenericAction(%s) %s: %s\n\tcmd: %s\n\tenv: %s",
		a.mnenonic, a.outs, a.ins, a.arguments, a.env)
}

type bazelActions []bazelAction

func (ba *bazelActions) fixPythonZipperActions() {
	// TODO
}

type bazelActionBuilder struct {
	paths            []bazelPath
	artifacts        []bazelFileArtifact
	middlemanDepsets [][]depsetId
	artifactSets     []bazelSetArtifact
}

func (bab *bazelActionBuilder) buildPaths(pathFragments []pathFragment) {
	// Find out the largest pathFragmentId, allocate array of bazelPath's
	var maxFragmentId pathFragmentId = 0
	for _, pf := range pathFragments {
		if pf.Id > maxFragmentId {
			maxFragmentId = pf.Id
		}
	}
	bab.paths = make([]bazelPath, maxFragmentId+1)
	// Create all bazelPaths
	for _, pf := range pathFragments {
		bab.paths[pf.Id] = bazelPath{fragment: pf.Label}
	}
	for _, pf := range pathFragments {
		if pf.ParentId == 0 {
			continue
		}
		if int(pf.ParentId) >= len(bab.paths) {
			bab.paths[pf.Id] = bazelPath{fragment: fmt.Sprintf("'bad path #%d'", pf.ParentId)}
			continue
		}
		bab.paths[pf.Id].parent = &bab.paths[pf.ParentId]
	}
}

// The file name of py3wrapper.sh, which is used by py_binary targets.
const py3wrapperFileName = "/py3wrapper.sh"

func (bab *bazelActionBuilder) buildFileArtifacts(artifacts []artifact) {
	// Find the largest artifactId, allocate array of bazelFileArtifacts
	var maxArtifactid artifactId = 0
	for _, a := range artifacts {
		if a.Id > maxArtifactid {
			maxArtifactid = a.Id
		}
	}
	bab.artifacts = make([]bazelFileArtifact, maxArtifactid+1)
	// Create bazelLFileArtifacts
	for _, a := range artifacts {
		bab.artifacts[a.Id] = bazelFileArtifact{file: &bab.paths[a.PathFragmentId]}
	}
}

func (bab bazelActionBuilder) fileArtifact(id artifactId) *bazelFileArtifact {
	if id > 0 && int(id) < len(bab.artifacts) {
		return &bab.artifacts[id]
	}
	return &bazelFileArtifact{file: nil}
}

func (bab *bazelActionBuilder) buildArtifactSets(depsets []depSetOfFiles) {
	// Allocate array of bazelSetArtifacts
	var maxDepsetId depsetId = 0
	for _, d := range depsets {
		if d.Id > maxDepsetId {
			maxDepsetId = d.Id
		}
	}
	bab.artifactSets = make([]bazelSetArtifact, maxDepsetId+1)

	// Create bazelSetArtifacts. depsets will be copied, so allocate at least that much
	for _, d := range depsets {
		bab.artifactSets[d.Id] = bazelSetArtifact{items: make([]bazelArtifact, len(d.TransitiveDepSetIds))}
	}

	for _, d := range depsets {
		i := 0
		for _, setId := range d.TransitiveDepSetIds {
			bab.artifactSets[d.Id].items[i] = bab.setArtifact(setId)
			i++
		}
		// Middleman artifacts are treated as "substitute" artifacts for mixed builds. For example,
		// if we find a middleman action which has outputs [foo, bar], and output [baz_middleman], then,
		// for each other action which has input [baz_middleman], we add [foo, bar] to the inputs for
		// that action instead.
		for _, fid := range d.DirectArtifactIds {
			if fid <= 0 || int(fid) >= len(bab.artifacts) {
				continue
			}
			if len(bab.middlemanDepsets[fid]) > 0 {
				for _, mmid := range bab.middlemanDepsets[fid] {
					bab.artifactSets[d.Id].items = append(bab.artifactSets[d.Id].items, bab.setArtifact(mmid))
				}
			} else {
				fa := bab.fileArtifact(fid)
				// See go/python-binary-host-mixed-build for more details.
				// 1) For py3wrapper.sh, there is no action for creating py3wrapper.sh in the aquery output of
				// Bazel py_binary targets, so there is no Ninja build statements generated for creating it.
				// 2) For MANIFEST file, SourceSymlinkManifest action is in aquery output of Bazel py_binary targets,
				// but it doesn't contain sufficient information so no Ninja build statements are generated
				// for creating it.
				// So in mixed build mode, when these two are used as input of some Ninja build statement,
				// since there is no build statement to create them, they should be removed from input paths.
				// TODO(b/197135294): Clean up this custom runfiles handling logic when
				// SourceSymlinkManifest and SymlinkTree actions are supported.
				if strings.HasSuffix(fa.file.fragment, py3wrapperFileName) {
					continue
				}
				// Match manifestFilePattern
				if fa.file.fragment == "MANIFEST" {
					continue
				}
				if fa.file.parent != nil && strings.HasSuffix(fa.file.parent.fragment, ".runfiles") {
					continue
				}
				bab.artifactSets[d.Id].items = append(bab.artifactSets[d.Id].items, fa)
			}
		}
	}
}

func (bab *bazelActionBuilder) buildMiddlemanMap(actions []action) {
	// We will need to know all the 'middleman' outputs in order to replace each such output
	// with inputs it depends on.
	bab.middlemanDepsets = make([][]depsetId, len(bab.artifacts))
	for _, a := range actions {
		if a.Mnemonic == "Middleman" {
			for _, id := range a.OutputIds {
				bab.middlemanDepsets[id] = a.InputDepSetIds
			}
		}
	}
}

func (bab bazelActionBuilder) oneOut(a action) *bazelFileArtifact {
	if len(a.OutputIds) != 1 {
		// TODO(asmundak): once tests is fixed, panic
		// panic(fmt.Errorf("expected single output for the %s action, got %d", a.Mnemonic, len(a.OutputIds)))
		return nil
	}
	return bab.fileArtifact(a.OutputIds[0])
}

func (bab bazelActionBuilder) allOuts(a action) []*bazelFileArtifact {
	outs := make([]*bazelFileArtifact, len(a.OutputIds))
	for i, id := range a.OutputIds {
		outs[i] = bab.fileArtifact(id)
	}
	return outs
}

func (bab bazelActionBuilder) oneIn(a action) bazelArtifact {
	if len(a.InputDepSetIds) != 1 {
		panic(fmt.Errorf("expected 1 input, %d", len(a.InputDepSetIds)))
	}
	return bab.setArtifact(a.InputDepSetIds[0])
}

func (bab bazelActionBuilder) allIns(a action) bazelSetArtifact {
	ins := bazelSetArtifact{items: make([]bazelArtifact, len(a.InputDepSetIds))}
	for i, id := range a.InputDepSetIds {
		ins.items[i] = bab.setArtifact(id)
	}
	return ins
}

func (bab bazelActionBuilder) setArtifact(id depsetId) bazelArtifact {
	if id > 0 && int(id) < len(bab.artifactSets) {
		return bab.artifactSets[id]
	}
	return &bazelBadArtifact{id: id}
}

func (bab bazelActionBuilder) unmarshalAction(a action) bazelAction {
	switch a.Mnemonic {
	case "Symlink", "SolibSymlink", "ExecutableSymlink":
		return bazelSymlinkAction{from: bab.oneIn(a), to: bab.oneOut(a)}
	case "TemplateExpand":
		return bazelTemplateExpandAction{to: bab.oneOut(a), template: a.TemplateContent, substitutions: a.Substitutions}
	case "PythonZipper":
		return bazelPythonZipperAction{
			outs:      bab.allOuts(a),
			ins:       bab.allIns(a),
			arguments: a.Arguments,
			env:       a.EnvironmentVariables,
		}
	case "FileWrite":
		return bazelFileWriteAction{to: bab.oneOut(a), contents: a.FileContents}
	case "SymlinkTree", "SourceSymlinkManifest":
		return nil
	case "Middleman":
		// Handled in buildMiddleman+buildArtifactSets
		return nil
	case "Fail":
		return nil
	default:
		return bazelGenericAction{
			arguments: a.Arguments,
			outs:      bab.allOuts(a),
			ins:       bab.allIns(a),
			env:       a.EnvironmentVariables,
			mnenonic:  a.Mnemonic,
		}
	}
}
func buildActions(aqueryJsonProto []byte) (bazelActions, error) {
	var aqueryResult actionGraphContainer
	if err := json.Unmarshal(aqueryJsonProto, &aqueryResult); err != nil {
		return nil, err
	}
	return aqueryResult.buildActions(), nil
}

func (ac actionGraphContainer) buildActions() bazelActions {
	var builder bazelActionBuilder
	builder.buildPaths(ac.PathFragments)
	builder.buildFileArtifacts(ac.Artifacts)
	builder.buildMiddlemanMap(ac.Actions)
	builder.buildArtifactSets(ac.DepSetOfFiles)

	// Finally, deserialize actions
	var bazelActions bazelActions
	for _, a := range ac.Actions {
		ba := builder.unmarshalAction(a)
		if ba != nil {
			bazelActions = append(bazelActions, ba)
		}
	}
	bazelActions.fixPythonZipperActions()
	if f, err := os2.CreateTemp("", "bzout"); err == nil {
		for _, ba := range bazelActions {
			fmt.Fprintln(f, ba)
		}
		for i, fa := range builder.artifacts[1:] {
			fmt.Fprintln(f, i+1, fa)
		}
		_ = f.Close()
		log.Printf("saved bazelActions in %s\n", f.Name())
	}
	return bazelActions
}
