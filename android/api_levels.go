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
	"encoding/json"
	"fmt"
	"strconv"
)

func init() {
	RegisterSingletonType("api_levels", ApiLevelsSingleton)
}

// Should only be used with DeserializeApiLevel. Must be public (and have public
// members) because all transitive members of a properties struct must be
// public.
type SerializedApiLevel struct {
	OriginalSpelling string
	CanonicalForm    string
	IsPreview        bool
	Value            int
}

// TODO: How do we deal with flavors like system_27 or APEX versions?
type ApiLevel interface {
	OriginalSpelling() string
	Canonical() string
	IsPreview() bool
	CompareTo(ApiLevel) int
	IsEqualTo(ApiLevel) bool
	Serialize() SerializedApiLevel
}

type FinalApiLevel interface {
	ApiLevel

	AsInt() int
}

type PreviewApiLevel interface {
	ApiLevel
}

type finalApiLevel struct {
	originalSpelling string
	canonicalForm    string
	value            int
}

func (this finalApiLevel) OriginalSpelling() string {
	return this.originalSpelling
}

func (this finalApiLevel) Canonical() string {
	return this.canonicalForm
}

func (this finalApiLevel) IsPreview() bool {
	return false
}

func (this finalApiLevel) AsInt() int {
	return this.value
}

func (this finalApiLevel) CompareTo(other ApiLevel) int {
	if other.IsPreview() {
		return -1
	}

	otherValue := other.(finalApiLevel).AsInt()
	if this.AsInt() < otherValue {
		return -1
	} else if this.AsInt() == otherValue {
		return 0
	} else {
		return 1
	}
}

func (this finalApiLevel) IsEqualTo(other ApiLevel) bool {
	return this.CompareTo(other) == 0
}

func (this finalApiLevel) Serialize() SerializedApiLevel {
	return SerializedApiLevel{
		OriginalSpelling: this.OriginalSpelling(),
		CanonicalForm:    this.Canonical(),
		IsPreview:        false,
		Value:            this.AsInt(),
	}
}

type previewApiLevel struct {
	originalSpelling string
	canonicalForm    string
}

func (this previewApiLevel) OriginalSpelling() string {
	return this.originalSpelling
}

func (this previewApiLevel) Canonical() string {
	return this.canonicalForm
}

func (this previewApiLevel) IsPreview() bool {
	return true
}

func (this previewApiLevel) CompareTo(other ApiLevel) int {
	if !other.IsPreview() {
		return 1
	}

	if this.Canonical() == other.Canonical() {
		return 0
	} else if other.Canonical() == "current" {
		return -1
	} else if this.Canonical() == "current" {
		return 1
	} else {
		// TODO: Should we impose ordering on actual previews?
		return 0
	}
}

func (this previewApiLevel) IsEqualTo(other ApiLevel) bool {
	return this.CompareTo(other) == 0
}

func (this previewApiLevel) Serialize() SerializedApiLevel {
	return SerializedApiLevel{
		OriginalSpelling: this.OriginalSpelling(),
		CanonicalForm:    this.Canonical(),
		IsPreview:        true,
		Value:            0,
	}
}

var _ FinalApiLevel = finalApiLevel{}
var _ PreviewApiLevel = previewApiLevel{}

// TODO: Merge with FutureApiLevel
var CurrentApiLevel = previewApiLevel{
	originalSpelling: "current",
	canonicalForm:    "current",
}

type ApiLevelCanonicalizer interface {
	ReplaceAliases(string) string
	AdjustFinalApiLevel(level int) int
}

func ReplaceFinalizedCodenames(ctx BaseModuleContext, raw string) string {
	num, ok := getApiLevelsMap(ctx.Config())[raw]
	if !ok {
		return raw
	}

	// TODO: Get a different map that returns the string and doesn't deal
	// with preview codenames.
	if num < 9000 {
		return strconv.Itoa(num)
	}

	return raw
}

func ApiLevelFromUser(ctx BaseModuleContext, raw string,
	canonicalizer ApiLevelCanonicalizer) ApiLevel {

	canonical := canonicalizer.ReplaceAliases(raw)
	asInt, err := strconv.Atoi(raw)
	if err != nil {
		return previewApiLevel{
			originalSpelling: raw,
			canonicalForm:    canonical,
		}
	}

	adjusted := canonicalizer.AdjustFinalApiLevel(asInt)
	if adjusted > ctx.Config().PlatformSdkVersionInt() {
		panic(fmt.Sprintf("Non-final API levels must be specified by their "+
			"code name, not integers. %q (adjusted to %d) is higher than "+
			"the maximum final API level %d", raw, adjusted,
			ctx.Config().PlatformSdkVersionInt()))
	}

	return finalApiLevel{
		originalSpelling: raw,
		canonicalForm:    strconv.Itoa(adjusted),
		value:            adjusted,
	}
}

// Deserializes a SerializedApiLevel which is assumed to have  already been
// through ApiLevelFromUser. No bounds checking, alias replacement, or
// canonicalization is done.
func DeserializeApiLevel(value SerializedApiLevel) ApiLevel {
	if value.IsPreview {
		return previewApiLevel{
			originalSpelling: value.OriginalSpelling,
			canonicalForm:    value.CanonicalForm,
		}
	}
	return finalApiLevel{
		originalSpelling: value.OriginalSpelling,
		canonicalForm:    value.CanonicalForm,
		value:            value.Value,
	}
}

func ApiLevelsSingleton() Singleton {
	return &apiLevelsSingleton{}
}

type apiLevelsSingleton struct{}

func createApiLevelsJson(ctx SingletonContext, file WritablePath,
	apiLevelsMap map[string]int) {

	jsonStr, err := json.Marshal(apiLevelsMap)
	if err != nil {
		ctx.Errorf(err.Error())
	}

	ctx.Build(pctx, BuildParams{
		Rule:        WriteFile,
		Description: "generate " + file.Base(),
		Output:      file,
		Args: map[string]string{
			"content": string(jsonStr[:]),
		},
	})
}

func GetApiLevelsJson(ctx PathContext) WritablePath {
	return PathForOutput(ctx, "api_levels.json")
}

var apiLevelsMapKey = NewOnceKey("ApiLevelsMap")

func getApiLevelsMap(config Config) map[string]int {
	return config.Once(apiLevelsMapKey, func() interface{} {
		baseApiLevel := 9000
		apiLevelsMap := map[string]int{
			"G":     9,
			"I":     14,
			"J":     16,
			"J-MR1": 17,
			"J-MR2": 18,
			"K":     19,
			"L":     21,
			"L-MR1": 22,
			"M":     23,
			"N":     24,
			"N-MR1": 25,
			"O":     26,
			"O-MR1": 27,
			"P":     28,
			"Q":     29,
		}
		for i, codename := range config.PlatformVersionActiveCodenames() {
			apiLevelsMap[codename] = baseApiLevel + i
		}

		return apiLevelsMap
	}).(map[string]int)
}

// Converts an API level string into its numeric form.
// * Codenames are decoded.
// * Numeric API levels are simply converted.
// * "current" is mapped to FutureApiLevel(10000)
// * "minimum" is NDK specific and not handled with this. (refer normalizeNdkApiLevel in cc.go)
func ApiStrToNum(ctx BaseModuleContext, apiLevel string) (int, error) {
	if apiLevel == "current" {
		return FutureApiLevel, nil
	}
	if num, ok := getApiLevelsMap(ctx.Config())[apiLevel]; ok {
		return num, nil
	}
	if num, err := strconv.Atoi(apiLevel); err == nil {
		return num, nil
	}
	return 0, fmt.Errorf("SDK version should be one of \"current\", <number> or <codename>: %q", apiLevel)
}

func (a *apiLevelsSingleton) GenerateBuildActions(ctx SingletonContext) {
	apiLevelsMap := getApiLevelsMap(ctx.Config())
	apiLevelsJson := GetApiLevelsJson(ctx)
	createApiLevelsJson(ctx, apiLevelsJson, apiLevelsMap)
}
