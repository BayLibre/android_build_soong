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

// Copies all APKs matching the target configuration from the given APK set
// into a zip file.
package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/golang/protobuf/proto"

	"io"
	"log"
	"os"

	"android/soong/cmd/extract_apks/bundle_proto"
	"android/soong/third_party/zip"
)

type TargetConfig struct {
	sdkVersion int32
	screenDpi  map[android_bundle_proto.ScreenDensity_DensityAlias]bool
	abis       map[android_bundle_proto.Abi_AbiAlias]bool
}

var (
	outputZip    = flag.String("o", "", "output zip containing extracted APKs")
	targetConfig = TargetConfig{
		screenDpi: map[android_bundle_proto.ScreenDensity_DensityAlias]bool{},
		abis:      map[android_bundle_proto.Abi_AbiAlias]bool{},
	}
)

type Toc *android_bundle_proto.BuildApksResult

type ApkSet struct {
	path    string
	reader  *zip.ReadCloser
	entries map[string]int
}

func newApkSet(path string) (*ApkSet, error) {
	apkSet := &ApkSet{path: path, entries: make(map[string]int)}
	var err error
	if apkSet.reader, err = zip.OpenReader(apkSet.path); err != nil {
		return nil, err
	}
	for i, f := range apkSet.reader.File {
		apkSet.entries[f.Name] = i
	}
	return apkSet, nil
}

func (apkSet *ApkSet) getToc() (Toc, error) {
	var err error
	tocIndex, ok := apkSet.entries["toc.pb"]
	if !ok {
		return nil, fmt.Errorf("%s: APK set should have toc.pb entry", apkSet.path)
	}
	tocFile := apkSet.reader.File[tocIndex]
	rc, err := tocFile.Open()
	if err != nil {
		return nil, err
	}
	bytes := make([]byte, tocFile.FileHeader.UncompressedSize64)
	if _, err := rc.Read(bytes); err != io.EOF {
		return nil, err
	}
	rc.Close()
	buildApksResult := new(android_bundle_proto.BuildApksResult)
	if err = proto.Unmarshal(bytes, buildApksResult); err != nil {
		return nil, err
	}
	return buildApksResult, nil
}

func (apkSet *ApkSet) close() {
	apkSet.reader.Close()
}

type bpAbiTargeting struct {
	*android_bundle_proto.AbiTargeting
}

func (t bpAbiTargeting) matches(config TargetConfig) bool {
	if t.AbiTargeting == nil {
		return true
	}
	if _, ok := config.abis[android_bundle_proto.Abi_UNSPECIFIED_CPU_ARCHITECTURE]; ok {
		return true
	}
	for _, v := range t.GetValue() {
		if _, ok := config.abis[v.Alias]; ok {
			return true
		}
	}
	return false
}

type bpApkDescription struct {
	*android_bundle_proto.ApkDescription
}

func (m bpApkDescription) matches(config TargetConfig) bool {
	return m.ApkDescription == nil || (bpApkTargeting{m.Targeting}).matches(config)
}

type bpApkTargeting struct {
	*android_bundle_proto.ApkTargeting
}

func (t bpApkTargeting) matches(config TargetConfig) bool {
	return t.ApkTargeting == nil ||
		(bpAbiTargeting{t.AbiTargeting}.matches(config) &&
			bpLanguageTargeting{t.LanguageTargeting}.matches(config) &&
			bpScreenDensityTargeting{t.ScreenDensityTargeting}.matches(config) &&
			bpSdkVersionTargeting{t.SdkVersionTargeting}.matches(config) &&
			bpMultiAbiTargeting{t.MultiAbiTargeting}.matches(config))
}

type bpLanguageTargeting struct {
	*android_bundle_proto.LanguageTargeting
}

func (t bpLanguageTargeting) matches(_ TargetConfig) bool {
	if t.LanguageTargeting == nil {
		return true
	}
	panic("Not implemented")

}

type bpModuleMetadata struct {
	*android_bundle_proto.ModuleMetadata
}

func (m bpModuleMetadata) matches(config TargetConfig) bool {
	return m.ModuleMetadata == nil ||
		(m.GetDeliveryType() == android_bundle_proto.DeliveryType_INSTALL_TIME &&
			bpModuleTargeting{m.Targeting}.matches(config) &&
			!m.IsInstant)
}

type bpModuleTargeting struct {
	*android_bundle_proto.ModuleTargeting
}

func (t bpModuleTargeting) matches(config TargetConfig) bool {
	return t.ModuleTargeting == nil ||
		(bpSdkVersionTargeting{t.SdkVersionTargeting}.matches(config) &&
			bpUserCountriesTargeting{t.UserCountriesTargeting}.matches(config))
}

type bpMultiAbiTargeting struct {
	*android_bundle_proto.MultiAbiTargeting
}

func (t bpMultiAbiTargeting) matches(_ TargetConfig) bool {
	if t.MultiAbiTargeting == nil {
		return true
	}
	// TODO(asmundak)
	panic("Not implemented")
}

type bpScreenDensityTargeting struct {
	*android_bundle_proto.ScreenDensityTargeting
}

func (t bpScreenDensityTargeting) matches(config TargetConfig) bool {
	if t.ScreenDensityTargeting == nil {
		return true
	}
	// TODO(asmundak)
	if _, ok := config.screenDpi[android_bundle_proto.ScreenDensity_DENSITY_UNSPECIFIED]; ok {
		return true
	}
	for _, v := range t.GetValue() {
		switch x := v.GetDensityOneof().(type) {
		case *android_bundle_proto.ScreenDensity_DensityAlias_:
			if _, ok := config.screenDpi[x.DensityAlias]; ok {
				return true
			}
		default:
			panic("Not implemented")
		}
	}
	return false
}

type bpSdkVersionTargeting struct {
	*android_bundle_proto.SdkVersionTargeting
}

func (t bpSdkVersionTargeting) matches(config TargetConfig) bool {
	if t.SdkVersionTargeting == nil {
		return true
	}
	if len(t.Value) > 1 {
		log.Fatal(fmt.Sprintf("sdk_version_targeting should not have multiple values:%#v", t.Value))
	}
	// We don't care about the better alternatives as we will select all
	// matching APKs
	return t.Value[0].Min == nil || t.Value[0].Min.Value <= config.sdkVersion
}

type bpTextureCompressionFormatTargeting struct {
	*android_bundle_proto.TextureCompressionFormatTargeting
}

func (t bpTextureCompressionFormatTargeting) matches(_ TargetConfig) bool {
	if t.TextureCompressionFormatTargeting == nil {
		return true
	}
	// TODO(asmundak)
	panic("Not implemented")
}

type bpUserCountriesTargeting struct {
	*android_bundle_proto.UserCountriesTargeting
}

func (t bpUserCountriesTargeting) matches(_ TargetConfig) bool {
	if t.UserCountriesTargeting == nil {
		return true
	}
	panic("Not implemented")
}

type bpVariantTargeting struct {
	*android_bundle_proto.VariantTargeting
}

func (m bpVariantTargeting) matches(config TargetConfig) bool {
	if m.VariantTargeting == nil {
		return true
	}
	return bpSdkVersionTargeting{m.SdkVersionTargeting}.matches(config) &&
		bpAbiTargeting{m.AbiTargeting}.matches(config) &&
		bpMultiAbiTargeting{m.MultiAbiTargeting}.matches(config) &&
		bpScreenDensityTargeting{m.ScreenDensityTargeting}.matches(config) &&
		bpTextureCompressionFormatTargeting{m.TextureCompressionFormatTargeting}.matches(config)
}

func selectApks(toc Toc, targetConfig TargetConfig) ([]string, error) {
	apks := []string{}
	for _, variant := range (*toc).GetVariant() {
		if !(bpVariantTargeting{variant.GetTargeting()}.matches(targetConfig)) {
			continue
		}
		for _, as := range variant.GetApkSet() {
			if !(bpModuleMetadata{as.ModuleMetadata}.matches(targetConfig)) {
				continue
			}
			for _, apkdesc := range as.GetApkDescription() {
				if (bpApkDescription{apkdesc}).matches(targetConfig) {
					apks = append(apks, apkdesc.GetPath())
				}
			}
		}
	}
	return apks, nil
}

func (apkSet *ApkSet) writeApks(apks []string, outPath string) error {
	if len(apks) == 0 {
		return fmt.Errorf("there are no APKs for the target configuration: %#v", targetConfig)
	}
	var writer *zip.Writer
	outFile, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer outFile.Close()
	writer = zip.NewWriter(outFile)
	defer func() {
		if err := writer.Close(); err != nil {
			log.Fatal(err)
		}
	}()
	for _, apk := range apks {
		i, ok := apkSet.entries[apk]
		if !ok {
			return fmt.Errorf("TOC refers to APK %s which does not exist", apk)
		}
		apkFile := apkSet.reader.File[i]
		if err := writer.CopyFrom(apkFile, apkFile.Name); err != nil {
			return err
		}
	}
	return nil
}

type abiValueIf struct {
	targetConfig *TargetConfig
}

func (a abiValueIf) String() string {
	return "all"
}

func (a abiValueIf) Set(abi_list string) error {
	if abi_list == "none" {
		return nil
	}
	if abi_list == "all" {
		targetConfig.abis[android_bundle_proto.Abi_UNSPECIFIED_CPU_ARCHITECTURE] = true
		return nil
	}
	for _, abi := range strings.Split(abi_list, ",") {
		v, ok := android_bundle_proto.Abi_AbiAlias_value[abi]
		if !ok {
			return fmt.Errorf("bad ABI value: %q", v)
		}
		targetConfig.abis[android_bundle_proto.Abi_AbiAlias(v)] = true
	}
	return nil
}

type screenDensityValueIf struct {
	targetConfig *TargetConfig
}

func (s screenDensityValueIf) String() string {
	return "none"
}

func (s screenDensityValueIf) Set(density_list string) error {
	if density_list == "none" {
		return nil
	}
	if density_list == "all" {
		targetConfig.screenDpi[android_bundle_proto.ScreenDensity_DENSITY_UNSPECIFIED] = true
		return nil
	}
	for _, density := range strings.Split(density_list, ",") {
		v, found := android_bundle_proto.ScreenDensity_DensityAlias_value[density]
		if !found {
			return fmt.Errorf("bad screen density value: %q", density)
		}
		targetConfig.screenDpi[android_bundle_proto.ScreenDensity_DensityAlias(v)] = true
	}
	return nil
}

func processArgs() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, `usage: extract_apks -o <output-zip> -sdk-version value -abis value -screen-densities value  <APK set>`)
		flag.PrintDefaults()
	}
	version := flag.Uint("sdk-version", 0, "SDK version")
	flag.Var(abiValueIf{&targetConfig}, "abis",
		"'all' or comma-separated ABIs list of ARMEABI ARMEABI_V7A ARM64_V8A X86 X86_64 MIPS MIPS64")
	flag.Var(screenDensityValueIf{&targetConfig}, "screen-densities",
		"'all' or comma-separated list of screen density names (NODPI LDPI MDPI TVDPI HDPI XHDPI XXHDPI XXXHDPI)")
	flag.Parse()
	if (*outputZip == "") || len(flag.Args()) != 1 || *version == 0 {
		flag.Usage()
	}
	targetConfig.sdkVersion = int32(*version)

}

func main() {
	processArgs()
	var toc Toc
	apkSet, err := newApkSet(flag.Arg(0))
	if err == nil {
		defer apkSet.close()
		toc, err = apkSet.getToc()
	}
	if err != nil {
		log.Fatal(err)
	}
	var apks []string
	if apks, err = selectApks(toc, targetConfig); err == nil {
		err = apkSet.writeApks(apks, *outputZip)
	}
	if err != nil {
		log.Fatal(err)
	}
}
