// Copyright 2023 Google Inc. All rights reserved.
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
	"io"
	"maps"
	"reflect"

	"github.com/google/blueprint"
)

var (
	mergeAconfigFilesRule = pctx.AndroidStaticRule("mergeAconfigFilesRule",
		blueprint.RuleParams{
			Command:     `${aconfig} dump --dedup --format protobuf --out $out $flags`,
			CommandDeps: []string{"${aconfig}"},
		}, "flags")
	_ = pctx.HostBinToolVariable("aconfig", "aconfig")
)

// Provider published by aconfig_value_set
type AconfigDeclarationsProviderData struct {
	Package                     string
	Container                   string
	Exportable                  bool
	IntermediateCacheOutputPath WritablePath
	IntermediateDumpOutputPath  WritablePath
}

var AconfigDeclarationsProviderKey = blueprint.NewProvider[AconfigDeclarationsProviderData]()

type AconfigReleaseDeclarationsProviderData map[string]AconfigDeclarationsProviderData

var AconfigReleaseDeclarationsProviderKey = blueprint.NewProvider[AconfigReleaseDeclarationsProviderData]()

type ModeInfo struct {
	Container string
	Mode      string
}
type CodegenInfo struct {
	// AconfigDeclarations is the name of the aconfig_declarations modules that
	// the codegen module is associated with
	AconfigDeclarations []string

	// Paths to the cache files of the associated aconfig_declaration modules
	IntermediateCacheOutputPaths Paths

	// Paths to the srcjar files generated from the java_aconfig_library modules
	Srcjars Paths

	ModeInfos map[string]ModeInfo
}

var CodegenInfoProvider = blueprint.NewProvider[CodegenInfo]()

func propagateModeInfos(ctx ModuleContext, module Module, to, from map[string]ModeInfo) {
	if len(from) > 0 {
		depTag := ctx.OtherModuleDependencyTag(module)
		if tag, ok := depTag.(PropagateAconfigValidationDependencyTag); ok && tag.PropagateAconfigValidation() {
			maps.Copy(to, from)
		}
	}
}

type aconfigPropagatingDeclarationsInfo struct {
	AconfigFiles map[string]Paths
	ModeInfos    map[string]ModeInfo
}

var AconfigPropagatingProviderKey = blueprint.NewProvider[aconfigPropagatingDeclarationsInfo]()

func VerifyAconfigBuildMode(ctx ModuleContext, container string, module blueprint.Module, asError bool) {
	if dep, ok := OtherModuleProvider(ctx, module, AconfigPropagatingProviderKey); ok {
		for k, v := range dep.ModeInfos {
			msg := fmt.Sprintf("%s/%s depends on %s/%s/%s across containers\n",
				module.Name(), container, k, v.Container, v.Mode)
			if v.Container != container && v.Mode != "exported" && v.Mode != "force-read-only" {
				if asError {
					ctx.ModuleErrorf(msg)
				} else {
					fmt.Print("WARNING: " + msg)
				}
			} else {
				if !asError {
					fmt.Print("PASSED: " + msg)
				}
			}
		}
	}
}

func aconfigUpdateAndroidBuildActions(ctx ModuleContext) {
	mergedAconfigFiles := make(map[string]Paths)
	mergedModeInfos := make(map[string]ModeInfo)

	ctx.VisitDirectDepsProxy(func(module ModuleProxy) {
		if aconfig_dep, ok := OtherModuleProvider(ctx, module, CodegenInfoProvider); ok && len(aconfig_dep.ModeInfos) > 0 {
			maps.Copy(mergedModeInfos, aconfig_dep.ModeInfos)
		}

		// If any of our dependencies have aconfig declarations (directly or propagated), then merge those and provide them.
		if dep, ok := OtherModuleProvider(ctx, module, AconfigDeclarationsProviderKey); ok {
			mergedAconfigFiles[dep.Container] = append(mergedAconfigFiles[dep.Container], dep.IntermediateCacheOutputPath)
		}
		// If we were generating on-device artifacts for other release configs, we would need to add code here to propagate
		// those artifacts as well.  See also b/298444886.
		if dep, ok := OtherModuleProvider(ctx, module, AconfigPropagatingProviderKey); ok {
			for container, v := range dep.AconfigFiles {
				mergedAconfigFiles[container] = append(mergedAconfigFiles[container], v...)
			}
			propagateModeInfos(ctx, module, mergedModeInfos, dep.ModeInfos)
		}
	})
	// We only need to set the provider if we have aconfig files.
	if len(mergedAconfigFiles) > 0 {
		for _, container := range SortedKeys(mergedAconfigFiles) {
			aconfigFiles := mergedAconfigFiles[container]
			mergedAconfigFiles[container] = mergeAconfigFiles(ctx, container, aconfigFiles, true)
		}

		SetProvider(ctx, AconfigPropagatingProviderKey, aconfigPropagatingDeclarationsInfo{
			AconfigFiles: mergedAconfigFiles,
			ModeInfos:    mergedModeInfos,
		})
		ctx.setAconfigPaths(getAconfigFilePaths(getContainer(ctx.Module()), mergedAconfigFiles))
	}
}

func aconfigUpdateAndroidMkData(ctx fillInEntriesContext, mod Module, data *AndroidMkData) {
	info, ok := OtherModuleProvider(ctx, mod, AconfigPropagatingProviderKey)
	// If there is no aconfigPropagatingProvider, or there are no AconfigFiles, then we are done.
	if !ok || len(info.AconfigFiles) == 0 {
		return
	}
	data.Extra = append(data.Extra, func(w io.Writer, outputFile Path) {
		AndroidMkEmitAssignList(w, "LOCAL_ACONFIG_FILES", getAconfigFilePaths(
			getContainerUsingProviders(ctx, mod), info.AconfigFiles).Strings())
	})
	// If there is a Custom writer, it needs to support this provider.
	if data.Custom != nil {
		switch reflect.TypeOf(mod).String() {
		case "*aidl.aidlApi": // writes non-custom before adding .phony
		case "*android_sdk.sdkRepoHost": // doesn't go through base_rules
		case "*apex.apexBundle": // aconfig_file properties written
		case "*bpf.bpf": // properties written (both for module and objs)
		case "*genrule.Module": // writes non-custom before adding .phony
		case "*java.SystemModules": // doesn't go through base_rules
		case "*phony.phony": // properties written
		case "*phony.PhonyRule": // writes phony deps and acts like `.PHONY`
		case "*sysprop.syspropLibrary": // properties written
		default:
			panic(fmt.Errorf("custom make rules do not handle aconfig files for %q (%q) module %q", ctx.ModuleType(mod), reflect.TypeOf(mod), mod))
		}
	}
}

func aconfigUpdateAndroidMkEntries(ctx fillInEntriesContext, mod Module, entries *[]AndroidMkEntries) {
	// If there are no entries, then we can ignore this module, even if it has aconfig files.
	if len(*entries) == 0 {
		return
	}
	info, ok := OtherModuleProvider(ctx, mod, AconfigPropagatingProviderKey)
	if !ok || len(info.AconfigFiles) == 0 {
		return
	}
	// All of the files in the module potentially depend on the aconfig flag values.
	for idx, _ := range *entries {
		(*entries)[idx].ExtraEntries = append((*entries)[idx].ExtraEntries,
			func(_ AndroidMkExtraEntriesContext, entries *AndroidMkEntries) {
				entries.AddPaths("LOCAL_ACONFIG_FILES", getAconfigFilePaths(
					getContainerUsingProviders(ctx, mod), info.AconfigFiles))
			},
		)

	}
}

// TODO(b/397766191): Change the signature to take ModuleProxy
// Please only access the module's internal data through providers.
func aconfigUpdateAndroidMkInfos(ctx fillInEntriesContext, mod Module, infos *AndroidMkProviderInfo) {
	info, ok := OtherModuleProvider(ctx, mod, AconfigPropagatingProviderKey)
	if !ok || len(info.AconfigFiles) == 0 {
		return
	}
	// All of the files in the module potentially depend on the aconfig flag values.
	infos.PrimaryInfo.AddPaths("LOCAL_ACONFIG_FILES", getAconfigFilePaths(
		getContainerUsingProviders(ctx, mod), info.AconfigFiles))
	if len(infos.ExtraInfo) > 0 {
		for _, ei := range (*infos).ExtraInfo {
			ei.AddPaths("LOCAL_ACONFIG_FILES", getAconfigFilePaths(
				getContainerUsingProviders(ctx, mod), info.AconfigFiles))
		}
	}
}

func mergeAconfigFiles(ctx ModuleContext, container string, inputs Paths, generateRule bool) Paths {
	inputs = SortedUniquePaths(inputs)
	if len(inputs) == 1 {
		return Paths{inputs[0]}
	}

	output := PathForModuleOut(ctx, container, "aconfig_merged.pb")

	if generateRule {
		ctx.Build(pctx, BuildParams{
			Rule:        mergeAconfigFilesRule,
			Description: "merge aconfig files",
			Inputs:      inputs,
			Output:      output,
			Args: map[string]string{
				"flags": JoinWithPrefix(inputs.Strings(), "--cache "),
			},
		})
	}

	return Paths{output}
}

func getContainer(m Module) string {
	container := "system"
	base := m.base()
	if base.SocSpecific() {
		container = "vendor"
	} else if base.ProductSpecific() {
		container = "product"
	} else if base.SystemExtSpecific() {
		container = "system_ext"
	}

	return container
}

// TODO(b/397766191): Change the signature to take ModuleProxy
// Please only access the module's internal data through providers.
func getContainerUsingProviders(ctx OtherModuleProviderContext, m Module) string {
	container := "system"
	commonInfo := OtherModulePointerProviderOrDefault(ctx, m, CommonModuleInfoProvider)
	if commonInfo.Vendor || commonInfo.Proprietary || commonInfo.SocSpecific {
		container = "vendor"
	} else if commonInfo.ProductSpecific {
		container = "product"
	} else if commonInfo.SystemExtSpecific {
		container = "system_ext"
	}

	return container
}

func getAconfigFilePaths(container string, aconfigFiles map[string]Paths) (paths Paths) {
	paths = append(paths, aconfigFiles[container]...)
	if container == "system" {
		// TODO(b/311155208): Once the default container is system, we can drop this.
		paths = append(paths, aconfigFiles[""]...)
	}
	if container != "system" {
		if len(aconfigFiles[container]) == 0 && len(aconfigFiles[""]) > 0 {
			// TODO(b/308625757): Either we guessed the container wrong, or the flag is misdeclared.
			// For now, just include the system (aka "") container if we get here.
			//fmt.Printf("container_mismatch: module=%v container=%v files=%v\n", m, container, aconfigFiles)
		}
		paths = append(paths, aconfigFiles[""]...)
	}
	return
}
void Global::Weapon::SetStartDamage(void * Player, void * ObjectPoolCallbackBase) {
        if (Player){
                uintptr_t (*func)(void *,void *);
                        func =  (uintptr_t (*)(void *,void *))(offsets.weapon.SetStartDamage);
                                func(Player,ObjectPoolCallbackBase);
                                    }
                                    }

                                    void * Global::Weapon::GetWeaponOnHand(uintptr_t Player) {
                                        if (Player){
                                                void * (*func)(uintptr_t );
                                                        func =  (void * (*)(uintptr_t ))(offsets.weapon.GetWeaponOnHand);
                                                                return func(Player);
                                                                    }
                                                                        return nullptr;
                                                                        }
                                                                        int Global::Weapon::TakeDamage(uintptr_t  _this, int baseDamage, COW_GamePlay_IHAAMHPPLMG_o Damage, void * DamageInfo, int weaponDataID, Vector3 firePos, Vector3 hitPos, monoList<float *> checkParams, void * damagerWeaponDynamicInfo, int damagerVehicleID) {
                                                                            if (_this){
                                                                                    int (*func)(uintptr_t ,int,COW_GamePlay_IHAAMHPPLMG_o,void *,int,Vector3,Vector3,monoList<float *>,void *,uint);
                                                                                            func =  (int (*)(uintptr_t ,int,COW_GamePlay_IHAAMHPPLMG_o,void *,int,Vector3,Vector3,monoList<float *>,void *,uint))(offsets.weapon.TakeDamage);
                                                                                                    return func(_this,baseDamage,Damage,DamageInfo,weaponDataID,firePos,hitPos,checkParams,damagerWeaponDynamicInfo,damagerVehicleID);
                                                                                                        }
                                                                                                            return 0;
                                                                                                            }

                                                                                                            void Global::Weapon::StartWholeBodyFiring(uintptr_t  Player, void * ObjectPoolCallbackBase) {
                                                                                                                if (Player){
                                                                                                                        uintptr_t (*func)(uintptr_t ,void *);
                                                                                                                                func =  (uintptr_t (*)(uintptr_t ,void *))(offsets.weapon.StartWholeBodyFiring);
                                                                                                                                        func(Player,ObjectPoolCallbackBase);
                                                                                                                                            }
                                                                                                                                            }

                                                                                                                                            void Global::Weapon::StopFire(uintptr_t  Player, void * ObjectPoolCallbackBase) {
                                                                                                                                                if (Player){
                                                                                                                                                        uintptr_t (*func)(uintptr_t ,void *);
                                                                                                                                                                func =  (uintptr_t (*)(uintptr_t ,void *))(offsets.weapon.StopFire);
                                                                                                                                                                        func(Player,ObjectPoolCallbackBase);
                                                                                                                                                                            }
                                                                                                                                                                            }

                                                                                                                                                                            int Global::Weapon::GetDamage(void * Player) {
                                                                                                                                                                                if (Player){
                                                                                                                                                                                        int (*func)(void *);
                                                                                                                                                                                                func =  (int (*)(void *))(offsets.weapon.GetDamage);
                                                                                                                                                                                                        return func(Player);
                                                                                                                                                                                                            }
                                                                                                                                                                                                                return 0;
                                                                                                                                                                                                                }
                                                                                                                                                                                                                int Global::Weapon::GetTypeWeapon(void * Player) {
                                                                                                                                                                                                                    if (Player){
                                                                                                                                                                                                                            int (*func)(void *);
                                                                                                                                                                                                                                    func =  (int (*)(void *))(offsets.weapon.GetTypeWeapon);
                                                                                                                                                                                                                                            return func(Player);
                                                                                                                                                                                                                                                }
                                                                                                                                                                                                                                                    return 0;
                                                                                                                                                                                                                                                    }

                                                                                                                                                                                                                                                    int Global::Weapon::GetWeaponID(void * Player) {
                                                                                                                                                                                                                                                        if (Player){
                                                                                                                                                                                                                                                                int (*func)(void *);
                                                                                                                                                                                                                                                                        func =  (int (*)(void *))(offsets.weapon.GetWeaponID);
                                                                                                                                                                                                                                                                                return func(Player);
                                                                                                                                                                                                                                                                                    }
                                                                                                                                                                                                                                                                                        return 0;
                                                                                                                                                                                                                                                                                        }
                                                                                                                                                                                                                                                                                        float Global::Timer::get_time() {
                                                                                                                                                                                                                                                                                            if (offsets.timer.get_time){
                                                                                                                                                                                                                                                                                                    float (*func)();
                                                                                                                                                                                                                                                                                                            func =  (float (*)())(offsets.timer.get_time);
                                                                                                                                                                                                                                                                                                                    return func();
                                                                                                                                                                                                                                                                                                                        }
                                                                                                                                                                                                                                                                                                                            return 0;
                                                                                                                                                                                                                                                                                                                            }
}
# include <iostream>


//usage 
if ( this->config.aimkill){
  if (enemyLocation != Vector3::Zero()) {
      aimkill.Start(localPlayer,closestEnemy);
        }
        }

        //offsets 

        using namespace Il2Cpp;
        offsets.weapon.SetStartDamage= (uintptr_t) GetMethodOffset(YK("Assembly-CSharp.dll"), YK("COW.GamePlay"), YK("GPBDEDFKJNA"),YK("BLAGCMCGEJG"),1);
        offsets.weapon.GetWeaponOnHand= (uintptr_t) GetMethodOffset(YK("Assembly-CSharp.dll"), YK("COW.GamePlay"), YK("Player"),YK("GetWeaponOnHand"));
        offsets.weapon.TakeDamage= (uintptr_t) GetMethodOffset(YK("Assembly-CSharp.dll"), YK("COW.GamePlay"), YK("PlayerNetwork"),YK("TakeDamage"),9);
        offsets.weapon.StartWholeBodyFiring= (uintptr_t) GetMethodOffset(YK("Assembly-CSharp.dll"), YK("COW.GamePlay"), YK("PlayerNetwork"),YK("StartWholeBodyFiring"),1);
        offsets.weapon.StopFire= (uintptr_t) GetMethodOffset(YK("Assembly-CSharp.dll"), YK("COW.GamePlay"), YK("PlayerNetwork"),YK("StopFire"),1);
        offsets.weapon.GetWeaponID= (uintptr_t) GetMethodOffset(YK("Assembly-CSharp.dll"), YK("COW.GamePlay"), YK("KOGBJLFDJHC"),YK("IDOGDPOPGAI"),0);
        offsets.weapon.GetDamage= (uintptr_t) GetMethodOffset(YK("Assembly-CSharp.dll"), YK("COW.GamePlay"), YK("GPBDEDFKJNA"),YK("MEMAEFCDOFL"),0);
        offsets.weapon.GetTypeWeapon= (uintptr_t) GetMethodOffset(YK("Assembly-CSharp.dll"), YK("COW.GamePlay"), YK("GPBDEDFKJNA"),YK("GJCHEHNJIAD"),0);

        offsets.weapon.GEGFCFDGGGP = (uintptr_t) GetFieldOffset(YK("Assembly-CSharp.dll"), YK("COW.GamePlay"),YK("Player"), YK("GEGFCFDGGGP"));
        offsets.weapon.KFMGKCJMCAM = (uintptr_t) GetFieldOffset(YK("Assembly-CSharp.dll"), YK("COW.GamePlay"),YK("Player"), YK("KFMGKCJMCAM"));
        offsets.weapon.PIGOIHAOJGH = (uintptr_t) GetFieldOffset(YK("Assembly-CSharp.dll"), YK("COW.GamePlay"),YK("GPBDEDFKJNA"), YK("PIGOIHAOJGH"));

        offsets.weapon.DBLBLKADCNP = (uintptr_t) GetFieldOffset(YK("Assembly-CSharp.dll"), YK("COW.GamePlay"),YK("ELMGJKHIIAA"), YK("DHGCIEKPBFA"));
        offsets.weapon.DHGCIEKPBFA = (uintptr_t) GetFieldOffset(YK("Assembly-CSharp.dll"), YK("COW.GamePlay"),YK("ELMGJKHIIAA"), YK("DHGCIEKPBFA"));
        offsets.weapon.KENBMOOEHBG = (uintptr_t) GetFieldOffset(YK("Assembly-CSharp.dll"), YK("COW.GamePlay"),YK("ELMGJKHIIAA"), YK("KENBMOOEHBG"));
        offsets.weapon.NNNADMOFPIE = (uintptr_t) GetFieldOffset(YK("Assembly-CSharp.dll"), YK("COW.GamePlay"),YK("ELMGJKHIIAA"), YK("NNNADMOFPIE"));
        offsets.weapon.MJIHLDJNHLF = (uintptr_t) GetFieldOffset(YK("Assembly-CSharp.dll"), YK("COW.GamePlay"),YK("ELMGJKHIIAA"), YK("MJIHLDJNHLF"));
        offsets.weapon.GPBDEDFKJNA = (uintptr_t) GetFieldOffset(YK("Assembly-CSharp.dll"), YK("COW.GamePlay"),YK("ELMGJKHIIAA"), YK("GPBDEDFKJNA"));
        offsets.weapon.CNEICNJFGLM = (uintptr_t) GetFieldOffset(YK("Assembly-CSharp.dll"), YK("COW.GamePlay"),YK("ELMGJKHIIAA"), YK("CNEICNJFGLM"));
        offsets.weapon.HECJHKEDFEB = (uintptr_t) GetFieldOffset(YK("Assembly-CSharp.dll"), YK("COW.GamePlay"),YK("ELMGJKHIIAA"), YK("HECJHKEDFEB"));
        offsets.weapon.LAEMLAPIAFD = (uintptr_t) GetFieldOffset(YK("Assembly-CSharp.dll"), YK("COW.GamePlay"),YK("GPBDEDFKJNA"), YK("LAEMLAPIAFD"));
        offsets.weapon.EFGDILOKKDP = (uintptr_t) GetFieldOffset(YK("Assembly-CSharp.dll"), YK("COW.GamePlay"),YK("GPBDEDFKJNA"), YK("EFGDILOKKDP"));
        offsets.weapon.PIAMIOFEBKF = (uintptr_t) GetFieldOffset(YK("Assembly-CSharp.dll"), YK("COW.GamePlay"),YK("ELMGJKHIIAA"), YK("PIAMIOFEBKF"));
        offsets.weapon.FHLFLAHCIBN = (uintptr_t) GetFieldOffset(YK("Assembly-CSharp.dll"), YK("COW.GamePlay"),YK("ELMGJKHIIAA"), YK("FHLFLAHCIBN"));
        offsets.weapon.JNLGFLFLBHO = (uintptr_t) GetFieldOffset(YK("Assembly-CSharp.dll"), YK("COW.GamePlay"),YK("ELMGJKHIIAA"), YK("JNLGFLFLBHO"));
        offsets.weapon.ACAKHEABPEJ = (uintptr_t) GetFieldOffset(YK("Assembly-CSharp.dll"), YK("COW.GamePlay"),YK("ELMGJKHIIAA"), YK("ACAKHEABPEJ"));
        offsets.weapon.POBGKMDJMDC = (uintptr_t) GetFieldOffset(YK("Assembly-CSharp.dll"), YK("COW.GamePlay"),YK("OOIPMACFIFL"), YK("POBGKMDJMDC"));

                    