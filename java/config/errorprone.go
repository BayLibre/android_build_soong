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

package config

import (
	"path/filepath"
	"strings"
)

func m2(group, artifact, version string) string {
	return filepath.Join("prebuilts/tools/common/m2/repository",
		strings.Replace(group, ".", "/", -1),
		artifact,
		version,
		artifact+"-"+version+".jar")
}

const (
	errorProneVersion          = "2.0.18"
	errorProneGuavaVersion     = "21.0"
	errorProneTruthVersion     = "0.30"
	errorProneAutovalueVersion = "1.3"
)

var (
	errorProneJavacJar = m2("com.google.errorprone", "javac", "9-dev-r3297-4")
	errorProneJar      = m2("com.google.errorprone", "error_prone_core", errorProneVersion)
)

func init() {
	// TODO: Verify source path for all m2 jars?
	pctx.SourcePathVariable("ErrorProneJavacJar", errorProneJavacJar)
	pctx.SourcePathVariable("ErrorProneJar", errorProneJar)
	pctx.SourcePathsVariable("ErrorProneClasspath", ":",
		m2("com.google.errorprone", "error_prone_core", errorProneVersion),
		m2("com.google.errorprone", "error_prone_annotation", errorProneVersion),
		m2("com.google.errorprone", "error_prone_check_api", errorProneVersion),
		m2("com.google.errorprone", "error_prone_test_helpers", errorProneVersion),
		m2("com.github.stephenc.jcip", "jcip-annotations", "1.0-1"),
		m2("org.pcollections", "pcollections", "2.1.2"),
		m2("com.google.guava", "guava", errorProneGuavaVersion),
		m2("com.google.auto", "auto-common", "0.7"),
		m2("com.google.code.findbugs", "jFormatString", "3.0.0"),
		m2("com.google.code.findbugs", "jsr305", "3.0.0"),
		m2("org.checkerframework", "dataflow", "1.8.10"),
		m2("com.google.auto.value", "auto-value", errorProneAutovalueVersion),
		m2("com.google.errorprone", "error_prone_annotations", errorProneVersion),
		m2("junit", "junit", "4.13-SNAPSHOT"),
		m2("org.hamcrest", "hamcrest-core", "1.3"),
		m2("org.hamcrest", "hamcrest-library", "1.3"),
		m2("com.google.truth", "truth", errorProneTruthVersion),
		m2("com.google.inject", "guice", "4.0-beta5"),
		m2("com.google.inject.extensions", "guice-assistedinject", "4.0-beta5"),
		m2("com.google.inject.extensions", "guice-servlet", "4.0-beta5"),
		m2("com.google.gwt.inject", "gin", "2.1.2"),
		m2("org.mockito", "mockito-core", "2.0.3-beta"),
		m2("org.jmock", "jmock", "2.8.0"),
		m2("org.jmock", "jmock-junit4", "2.8.0"),
		m2("com.google.protobuf", "protobuf-java", "3.0.0-beta-4"),
		m2("com.google.dagger", "dagger", "2.5"),
		m2("com.google.dagger", "dagger-producers", "2.5"),
		m2("com.google.auto.factory", "auto-factory", "1.0-beta3"),
		m2("com.google.guava", "guava-testlib", errorProneGuavaVersion),
		m2("com.google.testing.compile", "compile-testing", "0.9"),
		m2("com.ibm.icu", "icu4j", "56.1"),
		m2("com.google.android", "android", "4.1.1.4"),
		m2("com.google.android", "support-v4", "r6"),
		m2("com.google.auto.service", "auto-service", "1.0-rc2"),
		m2("org.checkerframework", "javacutil", "1.8.10"))

	// The checks that are fatal to the build.
	pctx.StaticVariable("ErrorProneChecksError", strings.Join([]string{
		"-Xep:AsyncCallableReturnsNull:ERROR",
		"-Xep:AsyncFunctionReturnsNull:ERROR",
		"-Xep:BundleDeserializationCast:ERROR",
		"-Xep:CompatibleWithAnnotationMisuse:ERROR",
		"-Xep:CompileTimeConstant:ERROR",
		"-Xep:DaggerProvidesNull:ERROR",
		"-Xep:DoNotCall:ERROR",
		"-Xep:ForOverride:ERROR",
		"-Xep:FunctionalInterfaceMethodChanged:ERROR",
		"-Xep:FuturesGetCheckedIllegalExceptionType:ERROR",
		"-Xep:GuiceAssistedInjectScoping:ERROR",
		"-Xep:GuiceAssistedParameters:ERROR",
		"-Xep:GuiceInjectOnFinalField:ERROR",
		"-Xep:Immutable:ERROR",
		"-Xep:ImmutableModification:ERROR",
		"-Xep:IncompatibleArgumentType:ERROR",
		"-Xep:IndexOfChar:ERROR",
		"-Xep:InjectMoreThanOneScopeAnnotationOnClass:ERROR",
		"-Xep:JavaxInjectOnAbstractMethod:ERROR",
		"-Xep:JUnit4SetUpNotRun:ERROR",
		"-Xep:JUnit4TearDownNotRun:ERROR",
		"-Xep:JUnit4TestNotRun:ERROR",
		"-Xep:JUnitAssertSameCheck:ERROR",
		"-Xep:LiteByteStringUtf8:ERROR",
		"-Xep:LoopConditionChecker:ERROR",
		"-Xep:MockitoCast:ERROR",
		"-Xep:MockitoUsage:ERROR",
		"-Xep:MoreThanOneInjectableConstructor:ERROR",
		"-Xep:MustBeClosedChecker:ERROR",
		"-Xep:NonCanonicalStaticImport:ERROR",
		"-Xep:NonFinalCompileTimeConstant:ERROR",
		"-Xep:OptionalEquality:ERROR",
		"-Xep:OverlappingQualifierAndScopeAnnotation:ERROR",
		"-Xep:PackageInfo:ERROR",
		"-Xep:PreconditionsCheckNotNull:ERROR",
		"-Xep:PreconditionsCheckNotNullPrimitive:ERROR",
		"-Xep:ProtoFieldNullComparison:ERROR",
		"-Xep:ProvidesMethodOutsideOfModule:ERROR",
		"-Xep:RestrictedApiChecker:ERROR",
		"-Xep:SelfAssignment:ERROR",
		"-Xep:StreamToString:ERROR",
		"-Xep:SuppressWarningsDeprecated:ERROR",
		"-Xep:ThrowIfUncheckedKnownChecked:ERROR",
		"-Xep:ThrowNull:ERROR",
		"-Xep:TypeParameterQualifier:ERROR",
		"-Xep:UnnecessaryTypeArgument:ERROR",
		"-Xep:UnusedAnonymousClass:ERROR",
	}, " "))

	pctx.StaticVariable("ErrorProneFlags", strings.Join([]string{
		"com.google.errorprone.ErrorProneCompiler",
		"-Xdiags:verbose",
		"-XDcompilePolicy=simple",
		"-XDallowBetterNullChecks=false",
		"-XDusePolyAttribution=true",
		"-XDuseStrictMethodClashCheck=true",
		"-XDuseStructuralMostSpecificResolution=true",
		"-XDuseGraphInference=true",
		"-Xmaxwarns 100000",
		"-XDandroidCompatible=true",
		"-XepAllErrorsAsWarnings",
	}, " "))

	pctx.StaticVariable("ErrorProneCmd",
		"${JavaCmd} -Xbootclasspath/p:${ErrorProneJavacJar} -cp ${ErrorProneClasspath} ${ErrorProneFlags} ${ErrorProneChecksError}")
}
