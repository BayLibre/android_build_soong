package cquery

import (
	"strings"
)

var (
	GetOutputFiles   RequestType = &getOutputFilesRequestType{}
	GetCcObjectFiles RequestType = &getCcObjectFilesRequestType{}
	GetCcLibraryInfo RequestType = &getCcLibraryInfoType{}
)

type GetCcLibraryInfo_Result struct {
	OutputFiles    []string
	CcObjectFiles  []string
	Includes       []string
	SystemIncludes []string
}

var RequestTypes []RequestType = []RequestType{
	GetOutputFiles, GetCcObjectFiles, GetCcLibraryInfo}

type RequestType interface {
	// Name returns a string name for this request type. Such request type names must be unique,
	// and must only consist of alphanumeric characters.
	Name() string

	// StarlarkFunctionBody returns a straark function body to process this request type.
	// The returned string is the body of a Starlark function which obtains
	// all request-relevant information about a target and returns a string containing
	// this information.
	// The function should have the following properties:
	//   - `target` is the only parameter to this function (a configured target).
	//   - The return value must be a string.
	//   - The function body should not be indented outside of its own scope.
	StarlarkFunctionBody() string

	// ParseResult returns a value obtained by parsing the result of the request's Starlark function.
	// The given rawString must correspond to the string output which was created by evaluating the
	// Starlark given in StarlarkFunctionBody.
	// The type of this value depends on the request type; it is up to the caller to
	// cast to the correct type.
	ParseResult(rawString string) interface{}
}

type getOutputFilesRequestType struct{}

func (g getOutputFilesRequestType) Name() string {
	return "getOutputFiles"
}

func (g getOutputFilesRequestType) StarlarkFunctionBody() string {
	return "return ', '.join([f.path for f in target.files.to_list()])"
}

func (g getOutputFilesRequestType) ParseResult(rawString string) interface{} {
	return strings.Split(rawString, ", ")
}

type getCcObjectFilesRequestType struct{}

func (g getCcObjectFilesRequestType) Name() string {
	return "getCcObjectFiles"
}

func (g getCcObjectFilesRequestType) StarlarkFunctionBody() string {
	return `
result = []
linker_inputs = providers(target)["CcInfo"].linking_context.linker_inputs.to_list()

for linker_input in linker_inputs:
  for library in linker_input.libraries:
    for object in library.objects:
      result += [object.path]
return ', '.join(result)`
}

func (g getCcObjectFilesRequestType) ParseResult(rawString string) interface{} {
	return strings.Split(rawString, ", ")
}

type getCcLibraryInfoType struct{}

func (g getCcLibraryInfoType) Name() string {
	return "getOutputFilesAndCcObjectFiles"
}

func (g getCcLibraryInfoType) StarlarkFunctionBody() string {
	return `
outputFiles = [f.path for f in target.files.to_list()]

ccObjectFiles = []
linker_inputs = providers(target)["CcInfo"].linking_context.linker_inputs.to_list()

includes = providers(target)["CcInfo"].compilation_context.includes.to_list()
system_includes = providers(target)["CcInfo"].compilation_context.system_includes.to_list()

for linker_input in linker_inputs:
  for library in linker_input.libraries:
    for object in library.objects:
      ccObjectFiles += [object.path]

ret = ', '.join(outputFiles)
ret += "|" + ', '.join(ccObjectFiles)
ret += "|" + ', '.join(includes)
ret += "|" + ', '.join(system_includes)

return ret`
}

func (g getCcLibraryInfoType) ParseResult(rawString string) interface{} {
	splitString := strings.Split(rawString, "|")
	outputFiles := strings.Split(splitString[0], ", ")
	ccObjects := strings.Split(splitString[1], ", ")
	includes := strings.Split(splitString[2], ", ")
	systemIncludes := strings.Split(splitString[3], ", ")

	return GetCcLibraryInfo_Result{
		outputFiles, ccObjects,
		includes, systemIncludes}
}
