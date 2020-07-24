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

// An API level, which may be a finalized (numbered) API, a preview (codenamed)
// API, or the future API level (10000). Can be parsed from a string with with
// ApiLevelFromUser or ApiLevelOrPanic.
//
// The different *types* of API levels are handled separately. Currently only
// Java has these, and they're managed with the sdkKind enum of the sdkSpec. A
// future cleanup should be to migrate sdkSpec to using ApiLevel instead of its
// sdkVersion int, and to move sdkSpec into this package.
type ApiLevel interface {
	// Returns the canonical name for this API level. For a finalized API level
	// this will be the API number as a string. For a preview API level this
	// will be the codename, or "current".
	String() string

	// Returns true if this is a non-final API level.
	IsPreview() bool

	// Returns true if this is the unfinalized "current" API level. This means
	// different things across Java and native. Java APIs do not use explicit
	// codenames, so all non-final codenames are grouped into "current". For
	// native explicit codenames are typically used, and current is the union of
	// all non-final APIs, including those that may not yet be in any codename.
	//
	// Note that in a build where the platform is final, "current" will not be a
	// preview API level but will instead be canonicalized to the final API
	// level.
	IsCurrent() bool

	// Returns -1 if the current API level is less than the argument, 0 if they
	// are equal, and 1 if it is greater than the argument.
	CompareTo(ApiLevel) int
	EqualTo(ApiLevel) bool
	GreaterThan(ApiLevel) bool
	GreaterThanOrEqualTo(ApiLevel) bool
	LessThan(ApiLevel) bool
	LessThanOrEqualTo(ApiLevel) bool

	// Returns either the final API number or the integer representing the
	// future API level. Should be removed eventually, but aids in the
	// transition from ints to ApiLevels.
	FinalOrFutureInt() int
}

type FinalApiLevel interface {
	ApiLevel

	AsInt() int
}

type PreviewApiLevel interface {
	ApiLevel
}

type finalApiLevel struct {
	value string
	asInt int
}

func (this finalApiLevel) String() string {
	return this.value
}

func (this finalApiLevel) IsPreview() bool {
	return false
}

func (this finalApiLevel) IsCurrent() bool {
	return false
}

func (this finalApiLevel) AsInt() int {
	return this.asInt
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

func (this finalApiLevel) EqualTo(other ApiLevel) bool {
	return this.CompareTo(other) == 0
}

func (this finalApiLevel) GreaterThan(other ApiLevel) bool {
	return this.CompareTo(other) > 0
}

func (this finalApiLevel) GreaterThanOrEqualTo(other ApiLevel) bool {
	return this.CompareTo(other) >= 0
}

func (this finalApiLevel) LessThan(other ApiLevel) bool {
	return this.CompareTo(other) < 0
}

func (this finalApiLevel) LessThanOrEqualTo(other ApiLevel) bool {
	return this.CompareTo(other) <= 0
}

func (this finalApiLevel) FinalOrFutureInt() int {
	return this.AsInt()
}

type previewApiLevel struct {
	value         string
	previewNumber int
}

func (this previewApiLevel) String() string {
	return this.value
}

func (this previewApiLevel) IsPreview() bool {
	return true
}

func (this previewApiLevel) IsCurrent() bool {
	return this.String() == "current"
}

func (this previewApiLevel) CompareTo(other ApiLevel) int {
	if !other.IsPreview() {
		return 1
	}

	otherValue := other.(previewApiLevel)
	if this.previewNumber < otherValue.previewNumber {
		return -1
	} else if this.previewNumber == otherValue.previewNumber {
		return 0
	} else {
		return 1
	}
}

func (this previewApiLevel) EqualTo(other ApiLevel) bool {
	return this.CompareTo(other) == 0
}

func (this previewApiLevel) GreaterThan(other ApiLevel) bool {
	return this.CompareTo(other) > 0
}

func (this previewApiLevel) GreaterThanOrEqualTo(other ApiLevel) bool {
	return this.CompareTo(other) >= 0
}

func (this previewApiLevel) LessThan(other ApiLevel) bool {
	return this.CompareTo(other) < 0
}

func (this previewApiLevel) LessThanOrEqualTo(other ApiLevel) bool {
	return this.CompareTo(other) <= 0
}

func (this previewApiLevel) FinalOrFutureInt() int {
	return FutureApiLevelInt
}

var _ FinalApiLevel = finalApiLevel{}
var _ PreviewApiLevel = previewApiLevel{}

func uncheckedFinalApiLevel(num int) finalApiLevel {
	return finalApiLevel{
		value: strconv.Itoa(num),
		asInt: num,
	}
}

// The first version that introduced 64-bit ABIs.
var FirstLp64Version = uncheckedFinalApiLevel(21)

// The first API level that does not require NDK code to link
// libandroid_support.
var FirstNonLibAndroidSupportVersion = uncheckedFinalApiLevel(21)

var FirstJava8Version = uncheckedFinalApiLevel(24)
var FirstJava9Version = uncheckedFinalApiLevel(30)

var FirstEmbeddedNativeLibsVersion = uncheckedFinalApiLevel(23)

// TODO: This probably shouldn't exist and should just be nil.
var NoneApiLevel = previewApiLevel{
	value:         "(no version)",
	previewNumber: 0,
}

func ReplaceFinalizedCodenames(ctx ConfigContext, raw string) string {
	num, ok := getFinalCodenamesMap(ctx.Config())[raw]
	if !ok {
		return raw
	}

	return strconv.Itoa(num)
}

func ApiLevelFromUser(ctx ConfigContext, raw string) (ApiLevel, error) {
	if raw == "" {
		panic("API level string must be non-empty")
	}

	if raw == "current" {
		return FutureApiLevel, nil
	}

	for i, codename := range ctx.Config().PlatformVersionActiveCodenames() {
		if codename == raw {
			return previewApiLevel{
				value:         raw,
				previewNumber: i,
			}, nil
		}
	}

	canonical := ReplaceFinalizedCodenames(ctx, raw)
	asInt, err := strconv.Atoi(canonical)
	if err != nil {
		return nil, fmt.Errorf("%q could not be parsed as an integer and is "+
			"not a recognized codename", canonical)
	}

	if asInt > ctx.Config().PlatformSdkVersion().AsInt() {
		// We don't want to allow this going forward, but there are already a
		// lot of build files that use 30 for R and 31 for S. We may want to
		// clean those up, but for now just allow them.
		//
		// Rather than just returning a previewApiLevel here, we call the
		// function again so we can be sure we get the right previewNumber.
		if asInt == 30 {
			return ApiLevelFromUser(ctx, "R")
		} else if asInt == 31 {
			return ApiLevelFromUser(ctx, "S")
		}
		return nil, fmt.Errorf("Non-final API levels must be specified by "+
			"their code name, not integers. %q (%d) is higher than the "+
			"maximum final API level %s", raw, asInt,
			ctx.Config().PlatformSdkVersion())
	}

	return uncheckedFinalApiLevel(asInt), nil
}

func ApiLevelOrPanic(ctx ConfigContext, raw string) ApiLevel {
	value, err := ApiLevelFromUser(ctx, raw)
	if err != nil {
		panic(err.Error())
	}
	return value
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

var finalCodenamesMapKey = NewOnceKey("FinalCodenamesMap")

func getFinalCodenamesMap(config Config) map[string]int {
	return config.Once(finalCodenamesMapKey, func() interface{} {
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

		// TODO: Differentiate "current" and "future".
		// The code base calls it FutureApiLevel, but the spelling is "current",
		// and these are really two different things. When defining APIs it
		// means the API has not yet been added to a specific release. When
		// choosing an API level to build for it means that the future API level
		// should be used, except in the case where the build is finalized in
		// which case the platform version should be used. This is *weird*,
		// because in the circumstance where API foo was added in R and bar was
		// added in S, both of these are usable when building for "current" when
		// neither R nor S are final, but the S APIs stop being available in a
		// final R build.
		if Bool(config.productVariables.Platform_sdk_final) {
			apiLevelsMap["current"] = config.PlatformSdkVersion().AsInt()
		}

		return apiLevelsMap
	}).(map[string]int)
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

func (a *apiLevelsSingleton) GenerateBuildActions(ctx SingletonContext) {
	apiLevelsMap := getApiLevelsMap(ctx.Config())
	apiLevelsJson := GetApiLevelsJson(ctx)
	createApiLevelsJson(ctx, apiLevelsJson, apiLevelsMap)
}
