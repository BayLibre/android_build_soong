package cquery

import (
	"strings"
)

type GetOutputFilesAndCcObjectFiles_Result struct {
	OutputFiles   []string
	CcObjectFiles []string
}

var Requests []*Request = []*Request{
	GetOutputFiles,
	GetOutputFilesAndCcObjectFiles,
}

// Request stores all information for a cquery request
type Request struct {
	name                 string
	starlarkFunctionBody string
	parseResult          func(string) interface{}
}

// Name returns a string name for this request type. Such request type names must be unique,
// and must only consist of alphanumeric characters.
func (r *Request) Name() string {
	return r.name
}

// StarlarkFunctionBody returns a straark function body to process this request type.
// The returned string is the body of a Starlark function which obtains
// all request-relevant information about a target and returns a string containing
// this information.
// The function should have the following properties:
//   - `target` is the only parameter to this function (a configured target).
//   - The return value must be a string.
//   - The function body should not be indented outside of its own scope.
func (r *Request) StarlarkFunctionBody() string {
	return r.starlarkFunctionBody
}

// ParseResult returns a value obtained by parsing the result of the request's Starlark function.
// The given rawString must correspond to the string output which was created by evaluating the
// Starlark given in StarlarkFunctionBody.
// The type of this value depends on the request type; it is up to the caller to
// cast to the correct type.
func (r Request) ParseResult(rawString string) interface{} {
	return r.parseResult(rawString)
}

// GetOutputFiles cquery request to get paths for output files from a Bazel target.
var GetOutputFiles = &Request{
	name: "getOutputFiles",

	starlarkFunctionBody: "return ', '.join([f.path for f in target.files.to_list()])",

	parseResult: func(rawString string) interface{} {
		return strings.Split(rawString, ", ")
	},
}

// GetOutputFilesAndCcObjectFiles cquery request to get paths for output files and cc object files
// from a cc Bazel target.
var GetOutputFilesAndCcObjectFiles = &Request{
	name: "getOutputFilesAndCcObjectFiles",

	starlarkFunctionBody: `
outputFiles = [f.path for f in target.files.to_list()]

ccObjectFiles = []
linker_inputs = providers(target)["CcInfo"].linking_context.linker_inputs.to_list()

for linker_input in linker_inputs:
  for library in linker_input.libraries:
    for object in library.objects:
      ccObjectFiles += [object.path]
return ', '.join(outputFiles) + "|" + ', '.join(ccObjectFiles)`,

	parseResult: func(rawString string) interface{} {
		var outputFiles []string
		var ccObjects []string

		splitString := strings.Split(rawString, "|")
		outputFilesString := splitString[0]
		ccObjectsString := splitString[1]
		outputFiles = strings.Split(outputFilesString, ", ")
		ccObjects = strings.Split(ccObjectsString, ", ")
		return GetOutputFilesAndCcObjectFiles_Result{outputFiles, ccObjects}
	},
}
