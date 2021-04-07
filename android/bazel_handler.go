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
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"android/soong/bazel/cquery"

	"github.com/google/blueprint/bootstrap"

	"android/soong/bazel"
	"android/soong/shared"
)

// Map key to describe bazel cquery requests.
type cqueryKey struct {
	label       string
	requestType cquery.RequestType
	archType    ArchType
}

// A context object which tracks queued requests that need to be made to Bazel,
// and their results after the requests have been made.
type BazelContext struct {
	bazelCommandContext

	paths *bazelPaths

	// Build statements which should get registered to reflect Bazel's outputs.
	buildStatements []bazel.BuildStatement
}

type bazelCommandContext interface {
	bazelEnabled() bool

	issueBazelCommand(runName bazel.RunName, paths *bazelPaths, command string, labels []string, extraFlags ...string) (string, string, error)

	cquery(label string, requestType cquery.RequestType, archType ArchType) (string, bool)
	parseCqueryOutput(cqueryOutput, cqueryError string) error
	clearCqueryRequests()

	mainBzlFileContents() string
	mainBuildFileContents() string
	cqueryStarlarkFileContents() string

	makeDirectoryIfNotExists(path string) error
	writeFile(path, contents string) error
}

type noopBazelCommandContext struct{}

func (n *noopBazelCommandContext) bazelEnabled() bool {
	return false
}

func (n *noopBazelCommandContext) issueBazelCommand(runName bazel.RunName, paths *bazelPaths, command string, labels []string, extraFlags ...string) (string, string, error) {
	panic("unimplemented")
}

func (n *noopBazelCommandContext) cquery(label string, requestType cquery.RequestType, archType ArchType) (string, bool) {
	panic("unimplemented")
}
func (n *noopBazelCommandContext) parseCqueryOutput(cqueryOutput, cqueryError string) error {
	panic("unimplemented")
}
func (n *noopBazelCommandContext) clearCqueryRequests() { panic("unimplemented") }

func (n *noopBazelCommandContext) mainBzlFileContents() string        { panic("unimplemented") }
func (n *noopBazelCommandContext) mainBuildFileContents() string      { panic("unimplemented") }
func (n *noopBazelCommandContext) cqueryStarlarkFileContents() string { panic("unimplemented") }

func (n *noopBazelCommandContext) makeDirectoryIfNotExists(_ string) error { panic("unimplemented") }
func (n *noopBazelCommandContext) writeFile(_, _ string) error             { panic("unimplemented") }

type MockBazelCommandContext struct {
	parsedCquery  bool
	cqueryResults map[string]string
}

func (m *MockBazelCommandContext) bazelEnabled() bool {
	return true
}

func (m *MockBazelCommandContext) issueBazelCommand(runName bazel.RunName, paths *bazelPaths, command string, labels []string, extraFlags ...string) (string, string, error) {
	return "", "", nil
}

func (m *MockBazelCommandContext) cquery(label string, requestType cquery.RequestType, archType ArchType) (string, bool) {
	if !m.parsedCquery {
		return "", false
	}
	res, ok := m.cqueryResults[label]
	return res, ok
}

func (m *MockBazelCommandContext) parseCqueryOutput(cqueryOutput, cqueryError string) error {
	m.parsedCquery = true
	return nil
}

func (m *MockBazelCommandContext) clearCqueryRequests() {}

func (m *MockBazelCommandContext) mainBzlFileContents() string        { return "" }
func (m *MockBazelCommandContext) mainBuildFileContents() string      { return "" }
func (m *MockBazelCommandContext) cqueryStarlarkFileContents() string { return "" }

func (m *MockBazelCommandContext) makeDirectoryIfNotExists(_ string) error { return nil }
func (m *MockBazelCommandContext) writeFile(_, _ string) error             { return nil }

func (bazelCtx *BazelContext) GetOutputFiles(label string, archType ArchType) ([]string, bool) {
	rawString, ok := bazelCtx.cquery(label, cquery.GetOutputFiles, archType)
	var ret []string
	if ok {
		bazelOutput := strings.TrimSpace(rawString)
		ret = cquery.GetOutputFiles.ParseResult(bazelOutput).([]string)
	}
	return ret, ok
}

func (bazelCtx *BazelContext) GetOutputFilesAndCcObjectFiles(label string, archType ArchType) ([]string, []string, bool) {
	var outputFiles []string
	var ccObjects []string

	result, ok := bazelCtx.cquery(label, cquery.GetOutputFilesAndCcObjectFiles, archType)
	if ok {
		bazelOutput := strings.TrimSpace(result)
		returnResult := cquery.GetOutputFilesAndCcObjectFiles.ParseResult(bazelOutput).(cquery.GetOutputFilesAndCcObjectFiles_Result)
		outputFiles = returnResult.OutputFiles
		ccObjects = returnResult.CcObjectFiles
	}

	return outputFiles, ccObjects, ok
}

func noopBazelContext() *BazelContext {
	return &BazelContext{bazelCommandContext: &noopBazelCommandContext{}}
}

func MockBazelContext(cqueryResults map[string]string, outputBase string) *BazelContext {
	return &BazelContext{
		bazelCommandContext: &MockBazelCommandContext{
			cqueryResults: cqueryResults,
		},
		paths: &bazelPaths{outputBase: outputBase},
	}
}

func NewBazelContext(c *config) (*BazelContext, error) {
	// TODO(cparsons): Assess USE_BAZEL=1 instead once "mixed Soong/Bazel builds"
	// are production ready.
	if c.Getenv("USE_BAZEL_ANALYSIS") == "1" {
		return noopBazelContext(), nil
	}

	paths, err := bazelPathsFromConfig(c)
	if err != nil {
		return nil, err
	}

	return &BazelContext{
		bazelCommandContext: &bazelContext{requests: make(map[cqueryKey]bool)},
		paths:               paths,
	}, nil
}

func bazelPathsFromConfig(c *config) (*bazelPaths, error) {
	paths := bazelPaths{
		buildDir: c.buildDir,
	}
	missingEnvVars := []string{}
	if len(c.Getenv("BAZEL_HOME")) > 1 {
		paths.homeDir = c.Getenv("BAZEL_HOME")
	} else {
		missingEnvVars = append(missingEnvVars, "BAZEL_HOME")
	}
	if len(c.Getenv("BAZEL_PATH")) > 1 {
		paths.bazelPath = c.Getenv("BAZEL_PATH")
	} else {
		missingEnvVars = append(missingEnvVars, "BAZEL_PATH")
	}
	if len(c.Getenv("BAZEL_OUTPUT_BASE")) > 1 {
		paths.outputBase = c.Getenv("BAZEL_OUTPUT_BASE")
	} else {
		missingEnvVars = append(missingEnvVars, "BAZEL_OUTPUT_BASE")
	}
	if len(c.Getenv("BAZEL_WORKSPACE")) > 1 {
		paths.workspaceDir = c.Getenv("BAZEL_WORKSPACE")
	} else {
		missingEnvVars = append(missingEnvVars, "BAZEL_WORKSPACE")
	}
	if len(c.Getenv("BAZEL_METRICS_DIR")) > 1 {
		paths.metricsDir = c.Getenv("BAZEL_METRICS_DIR")
	} else {
		missingEnvVars = append(missingEnvVars, "BAZEL_METRICS_DIR")
	}
	if len(missingEnvVars) > 0 {
		return nil, errors.New(fmt.Sprintf("missing required env vars to use bazel: %s", missingEnvVars))
	} else {
		return &paths, nil
	}
}

type bazelPaths struct {
	homeDir      string
	bazelPath    string
	outputBase   string
	workspaceDir string
	buildDir     string
	metricsDir   string
}

// BazelMetricsDir provides Bazel-specific metrics directory  for use in shared paths
func (p *bazelPaths) BazelMetricsDir() string {
	return p.metricsDir
}

// Returns a workspace-relative path containing build-related metadata required
// for interfacing with Bazel. Example: out/soong/bazel.
func (p *bazelPaths) intermediatesDir() string {
	return filepath.Join(p.buildDir, "bazel")
}

func (context *BazelContext) BazelEnabled() bool {
	return context.bazelEnabled()
}

type bazelContext struct {
	requests     map[cqueryKey]bool // cquery requests that have not yet been issued to Bazel
	requestMutex sync.Mutex         // requests can be written in parallel

	results map[cqueryKey]string // Results of cquery requests after Bazel invocations
}

func (context *bazelContext) bazelEnabled() bool {
	return true
}

// Adds a cquery request to the Bazel request queue, to be later invoked, or
// returns the result of the given request if the request was already made.
// If the given request was already made (and the results are available), then
// returns (result, true). If the request is queued but no results are available,
// then returns ("", false).
func (context *bazelContext) cquery(label string, requestType cquery.RequestType,
	archType ArchType) (string, bool) {
	key := cqueryKey{label, requestType, archType}
	if result, ok := context.results[key]; ok {
		return result, true
	} else {
		context.requestMutex.Lock()
		defer context.requestMutex.Unlock()
		context.requests[key] = true
		return "", false
	}
}

func (context *bazelContext) parseCqueryOutput(cqueryOutput, cqueryError string) error {
	context.results = make(map[cqueryKey]string)

	cqueryResults := map[string]string{}
	for _, outputLine := range strings.Split(cqueryOutput, "\n") {
		if strings.Contains(outputLine, ">>") {
			splitLine := strings.SplitN(outputLine, ">>", 2)
			cqueryResults[splitLine[0]] = splitLine[1]
		}
	}

	for val, _ := range context.requests {
		if cqueryResult, ok := cqueryResults[getCqueryId(val)]; ok {
			context.results[val] = string(cqueryResult)
		} else {
			return fmt.Errorf("missing result for bazel target %s. query output: [%s], cquery err: [%s]",
				getCqueryId(val), cqueryOutput, cqueryError)
		}
	}
	return nil
}

func (c *bazelContext) clearCqueryRequests() {
	c.requests = make(map[cqueryKey]bool)
}

func pwdPrefix() string {
	// Darwin doesn't have /proc
	if runtime.GOOS != "darwin" {
		return "PWD=/proc/self/cwd"
	}
	return ""
}

// Issues the given bazel command with given build label and additional flags.
// Returns (stdout, stderr, error). The first and second return values are strings
// containing the stdout and stderr of the run command, and an error is returned if
// the invocation returned an error code.
func (context *bazelContext) issueBazelCommand(runName bazel.RunName, paths *bazelPaths, command string, labels []string,
	extraFlags ...string) (string, string, error) {

	cmdFlags := []string{"--output_base=" + paths.outputBase, command}
	cmdFlags = append(cmdFlags, labels...)
	cmdFlags = append(cmdFlags, "--package_path=%workspace%/"+paths.intermediatesDir())
	cmdFlags = append(cmdFlags, "--profile="+shared.BazelMetricsFilename(paths, runName))

	// Set default platforms to canonicalized values for mixed builds requests.
	// If these are set in the bazelrc, they will have values that are
	// non-canonicalized to @sourceroot labels, and thus be invalid when
	// referenced from the buildroot.
	//
	// The actual platform values here may be overridden by configuration
	// transitions from the buildroot.
	cmdFlags = append(cmdFlags,
		fmt.Sprintf("--platforms=%s", canonicalizeLabel("//build/bazel/platforms:android_x86_64")))
	cmdFlags = append(cmdFlags,
		fmt.Sprintf("--extra_toolchains=%s", canonicalizeLabel("//prebuilts/clang/host/linux-x86:all")))
	// This should be parameterized on the host OS, but let's restrict to linux
	// to keep things simple for now.
	cmdFlags = append(cmdFlags,
		fmt.Sprintf("--host_platform=%s", canonicalizeLabel("//build/bazel/platforms:linux_x86_64")))

	// Explicitly disable downloading rules (such as canonical C++ and Java rules) from the network.
	cmdFlags = append(cmdFlags, "--experimental_repository_disable_download")
	cmdFlags = append(cmdFlags, extraFlags...)

	bazelCmd := exec.Command(paths.bazelPath, cmdFlags...)
	bazelCmd.Dir = paths.workspaceDir
	bazelCmd.Env = append(os.Environ(), "HOME="+paths.homeDir, pwdPrefix(),
		// Disables local host detection of gcc; toolchain information is defined
		// explicitly in BUILD files.
		"BAZEL_DO_NOT_DETECT_CPP_TOOLCHAIN=1")
	stderr := &bytes.Buffer{}
	bazelCmd.Stderr = stderr

	if output, err := bazelCmd.Output(); err != nil {
		return "", string(stderr.Bytes()),
			fmt.Errorf("bazel command failed. command: [%s], env: [%s], error [%s]", bazelCmd, bazelCmd.Env, stderr)
	} else {
		return string(output), string(stderr.Bytes()), nil
	}
}

// Returns the string contents of a workspace file that should be output
// adjacent to the main bzl file and build file.
// This workspace file allows, via local_repository rule, sourcetree-level
// BUILD targets to be referenced via @sourceroot.
func (context *BazelContext) workspaceFileContents() string {
	formatString := `
# This file is generated by soong_build. Do not edit.
local_repository(
    name = "sourceroot",
    path = "%s",
)

local_repository(
    name = "rules_cc",
    path = "%s/build/bazel/rules_cc",
)
`
	return fmt.Sprintf(formatString, context.paths.workspaceDir, context.paths.workspaceDir)
}

func (context *bazelContext) mainBzlFileContents() string {
	// TODO(cparsons): Define configuration transitions programmatically based
	// on available archs.
	contents := `
#####################################################
# This file is generated by soong_build. Do not edit.
#####################################################

def _config_node_transition_impl(settings, attr):
    return {
        "//command_line_option:platforms": "@sourceroot//build/bazel/platforms:android_%s" % attr.arch,
    }

_config_node_transition = transition(
    implementation = _config_node_transition_impl,
    inputs = [],
    outputs = [
        "//command_line_option:platforms",
    ],
)

def _passthrough_rule_impl(ctx):
    return [DefaultInfo(files = depset(ctx.files.deps))]

config_node = rule(
    implementation = _passthrough_rule_impl,
    attrs = {
        "arch" : attr.string(mandatory = True),
        "deps" : attr.label_list(cfg = _config_node_transition),
        "_allowlist_function_transition": attr.label(default = "@bazel_tools//tools/allowlists/function_transition_allowlist"),
    },
)


# Rule representing the root of the build, to depend on all Bazel targets that
# are required for the build. Building this target will build the entire Bazel
# build tree.
mixed_build_root = rule(
    implementation = _passthrough_rule_impl,
    attrs = {
        "deps" : attr.label_list(),
    },
)

def _phony_root_impl(ctx):
    return []

# Rule to depend on other targets but build nothing.
# This is useful as follows: building a target of this rule will generate
# symlink forests for all dependencies of the target, without executing any
# actions of the build.
phony_root = rule(
    implementation = _phony_root_impl,
    attrs = {"deps" : attr.label_list()},
)
`
	return contents
}

// Returns a "canonicalized" corresponding to the given sourcetree-level label.
// This abstraction is required because a sourcetree label such as //foo/bar:baz
// must be referenced via the local repository prefix, such as
// @sourceroot//foo/bar:baz.
func canonicalizeLabel(label string) string {
	if strings.HasPrefix(label, "//") {
		return "@sourceroot" + label
	} else {
		return "@sourceroot//" + label
	}
}

func (context *bazelContext) mainBuildFileContents() string {
	// TODO(cparsons): Map label to attribute programmatically; don't use hard-coded
	// architecture mapping.
	formatString := `
# This file is generated by soong_build. Do not edit.
load(":main.bzl", "config_node", "mixed_build_root", "phony_root")

%s

mixed_build_root(name = "buildroot",
    deps = [%s],
)

phony_root(name = "phonyroot",
    deps = [":buildroot"],
)
`
	configNodeFormatString := `
config_node(name = "%s",
    arch = "%s",
    deps = [%s],
)
`

	configNodesSection := ""

	labelsByArch := map[string][]string{}
	for val, _ := range context.requests {
		labelString := fmt.Sprintf("\"%s\"", canonicalizeLabel(val.label))
		archString := getArchString(val)
		labelsByArch[archString] = append(labelsByArch[archString], labelString)
	}

	configNodeLabels := []string{}
	for archString, labels := range labelsByArch {
		configNodeLabels = append(configNodeLabels, fmt.Sprintf("\":%s\"", archString))
		labelsString := strings.Join(labels, ",\n            ")
		configNodesSection += fmt.Sprintf(configNodeFormatString, archString, archString, labelsString)
	}

	return fmt.Sprintf(formatString, configNodesSection, strings.Join(configNodeLabels, ",\n            "))
}

func indent(original string) string {
	result := ""
	for _, line := range strings.Split(original, "\n") {
		result += "  " + line + "\n"
	}
	return result
}

// Returns the file contents of the buildroot.cquery file that should be used for the cquery
// expression in order to obtain information about buildroot and its dependencies.
// The contents of this file depend on the bazelContext's requests; requests are enumerated
// and grouped by their request type. The data retrieved for each label depends on its
// request type.
func (context *bazelContext) cqueryStarlarkFileContents() string {
	requestTypeToCqueryIdEntries := map[cquery.RequestType][]string{}
	for val, _ := range context.requests {
		cqueryId := getCqueryId(val)
		mapEntryString := fmt.Sprintf("%q : True", cqueryId)
		requestTypeToCqueryIdEntries[val.requestType] =
			append(requestTypeToCqueryIdEntries[val.requestType], mapEntryString)
	}
	labelRegistrationMapSection := ""
	functionDefSection := ""
	mainSwitchSection := ""

	mapDeclarationFormatString := `
%s = {
  %s
}
`
	functionDefFormatString := `
def %s(target):
%s
`
	mainSwitchSectionFormatString := `
  if id_string in %s:
    return id_string + ">>" + %s(target)
`

	for requestType, _ := range requestTypeToCqueryIdEntries {
		labelMapName := requestType.Name() + "_Labels"
		functionName := requestType.Name() + "_Fn"
		labelRegistrationMapSection += fmt.Sprintf(mapDeclarationFormatString,
			labelMapName,
			strings.Join(requestTypeToCqueryIdEntries[requestType], ",\n  "))
		functionDefSection += fmt.Sprintf(functionDefFormatString,
			functionName,
			indent(requestType.StarlarkFunctionBody()))
		mainSwitchSection += fmt.Sprintf(mainSwitchSectionFormatString,
			labelMapName, functionName)
	}

	formatString := `
# This file is generated by soong_build. Do not edit.

# Label Map Section
%s

# Function Def Section
%s

def get_arch(target):
  buildoptions = build_options(target)
  platforms = build_options(target)["//command_line_option:platforms"]
  if len(platforms) != 1:
    # An individual configured target should have only one platform architecture.
    # Note that it's fine for there to be multiple architectures for the same label,
    # but each is its own configured target.
    fail("expected exactly 1 platform for " + str(target.label) + " but got " + str(platforms))
  platform_name = build_options(target)["//command_line_option:platforms"][0].name
  if platform_name == "host":
    return "HOST"
  elif not platform_name.startswith("android_"):
    fail("expected platform name of the form 'android_<arch>', but was " + str(platforms))
    return "UNKNOWN"
  return platform_name[len("android_"):]

def format(target):
  id_string = str(target.label) + "|" + get_arch(target)

  # Main switch section
  %s
  # This target was not requested via cquery, and thus must be a dependency
  # of a requested target.
  return id_string + ">>NONE"
`

	return fmt.Sprintf(formatString, labelRegistrationMapSection, functionDefSection, mainSwitchSection)
}

func (c *bazelContext) makeDirectoryIfNotExists(dir string) error {
	var err error
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return os.Mkdir(dir, 0777)
	}
	return err
}

func (c *bazelContext) writeFile(path, contents string) error {
	return ioutil.WriteFile(path, []byte(contents), 0666)
}

// Issues commands to Bazel to receive results for all cquery requests
// queued in the BazelContext.
func (context *BazelContext) InvokeBazel() error {
	var cqueryOutput string
	var cqueryErr string
	var err error

	// TODO(eakammer): handle writing during tests
	intermediatesDirPath := absolutePath(context.paths.intermediatesDir())
	err = context.makeDirectoryIfNotExists(intermediatesDirPath)
	if err != nil {
		return err
	}
	err = context.writeFile(filepath.Join(intermediatesDirPath, "main.bzl"), context.mainBzlFileContents())
	if err != nil {
		return err
	}
	err = context.writeFile(filepath.Join(intermediatesDirPath, "BUILD.bazel"), context.mainBuildFileContents())
	if err != nil {
		return err
	}
	cqueryFileRelpath := filepath.Join(context.paths.intermediatesDir(), "buildroot.cquery")
	err = context.writeFile(absolutePath(cqueryFileRelpath), context.cqueryStarlarkFileContents())
	if err != nil {
		return err
	}
	workspaceFileRelpath := filepath.Join(context.paths.intermediatesDir(), "WORKSPACE.bazel")
	err = context.writeFile(absolutePath(workspaceFileRelpath), context.workspaceFileContents())
	if err != nil {
		return err
	}
	buildrootLabel := "//:buildroot"
	cqueryOutput, cqueryErr, err = context.issueBazelCommand(bazel.CqueryBuildRootRunName, context.paths, "cquery",
		[]string{fmt.Sprintf("kind(rule, deps(%s))", buildrootLabel)},
		"--output=starlark",
		"--starlark:file="+cqueryFileRelpath)
	if err != nil {
		return err
	}
	err = context.writeFile(filepath.Join(intermediatesDirPath, "cquery.out"), cqueryOutput)
	if err != nil {
		return err
	}

	err = context.parseCqueryOutput(cqueryOutput, cqueryErr)
	if err != nil {
		return err
	}

	// Issue an aquery command to retrieve action information about the bazel build tree.
	//
	// TODO(cparsons): Use --target_pattern_file to avoid command line limits.
	var aqueryOutput string
	aqueryOutput, _, err = context.issueBazelCommand(bazel.AqueryBuildRootRunName, context.paths, "aquery",
		[]string{fmt.Sprintf("deps(%s)", buildrootLabel),
			// Use jsonproto instead of proto; actual proto parsing would require a dependency on Bazel's
			// proto sources, which would add a number of unnecessary dependencies.
			"--output=jsonproto"})

	if err != nil {
		return err
	}

	context.buildStatements, err = bazel.AqueryBuildStatements([]byte(aqueryOutput))
	if err != nil {
		return err
	}

	// Issue a build command of the phony root to generate symlink forests for dependencies of the
	// Bazel build. This is necessary because aquery invocations do not generate this symlink forest,
	// but some of symlinks may be required to resolve source dependencies of the build.
	_, _, err = context.issueBazelCommand(bazel.BazelBuildPhonyRootRunName, context.paths, "build",
		[]string{"//:phonyroot"})

	if err != nil {
		return err
	}

	// Clear requests.
	context.clearCqueryRequests()
	return nil
}

func (context *BazelContext) BuildStatementsToRegister() []bazel.BuildStatement {
	return context.buildStatements
}

func (context *BazelContext) OutputBase() string {
	return context.paths.outputBase
}

// Singleton used for registering BUILD file ninja dependencies (needed
// for correctness of builds which use Bazel.
func BazelSingleton() Singleton {
	return &bazelSingleton{}
}

type bazelSingleton struct{}

func (c *bazelSingleton) GenerateBuildActions(ctx SingletonContext) {
	// bazelSingleton is a no-op if mixed-soong-bazel-builds are disabled.
	if !ctx.Config().BazelContext.BazelEnabled() {
		return
	}

	// Add ninja file dependencies for files which all bazel invocations require.
	bazelBuildList := absolutePath(filepath.Join(
		filepath.Dir(bootstrap.CmdlineModuleListFile()), "bazel.list"))
	ctx.AddNinjaFileDeps(bazelBuildList)

	data, err := ioutil.ReadFile(bazelBuildList)
	if err != nil {
		ctx.Errorf(err.Error())
	}
	files := strings.Split(strings.TrimSpace(string(data)), "\n")
	for _, file := range files {
		ctx.AddNinjaFileDeps(file)
	}

	// Register bazel-owned build statements (obtained from the aquery invocation).
	for index, buildStatement := range ctx.Config().BazelContext.BuildStatementsToRegister() {
		if len(buildStatement.Command) < 1 {
			panic(fmt.Sprintf("unhandled build statement: %v", buildStatement))
		}
		rule := NewRuleBuilder(pctx, ctx)
		cmd := rule.Command()
		cmd.Text(fmt.Sprintf("cd %s/execroot/__main__ && %s",
			ctx.Config().BazelContext.OutputBase(), buildStatement.Command))

		for _, outputPath := range buildStatement.OutputPaths {
			cmd.ImplicitOutput(PathForBazelOut(ctx, outputPath))
		}
		for _, inputPath := range buildStatement.InputPaths {
			cmd.Implicit(PathForBazelOut(ctx, inputPath))
		}

		if depfile := buildStatement.Depfile; depfile != nil {
			cmd.ImplicitDepFile(PathForBazelOut(ctx, *depfile))
		}

		// This is required to silence warnings pertaining to unexpected timestamps. Particularly,
		// some Bazel builtins (such as files in the bazel_tools directory) have far-future
		// timestamps. Without restat, Ninja would emit warnings that the input files of a
		// build statement have later timestamps than the outputs.
		rule.Restat()

		rule.Build(fmt.Sprintf("bazel %d", index), buildStatement.Mnemonic)
	}
}

func getCqueryId(key cqueryKey) string {
	return canonicalizeLabel(key.label) + "|" + getArchString(key)
}

func getArchString(key cqueryKey) string {
	arch := key.archType.Name
	if len(arch) > 0 {
		return arch
	} else {
		return "x86_64"
	}
}
