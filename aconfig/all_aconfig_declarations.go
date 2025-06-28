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

package aconfig

import (
	"fmt"
	"slices"

	"android/soong/android"

	"github.com/google/blueprint"
	"github.com/google/blueprint/proptools"
)

// A singleton module that collects all of the aconfig flags declared in the
// tree into a single combined file for export to the external flag setting
// server (inside Google it's Gantry).
//
// Note that this is ALL aconfig_declarations modules present in the tree, not just
// ones that are relevant to the product currently being built, so that that infra
// doesn't need to pull from multiple builds and merge them.
func AllAconfigDeclarationsFactory() android.SingletonModule {
	module := &allAconfigDeclarationsSingleton{releaseMap: make(map[string]allAconfigReleaseDeclarationsSingleton)}
	module.AddProperties(&module.properties)
	android.InitAndroidArchModule(module, android.DeviceSupported, android.MultilibCommon)
	return module
}

type allAconfigDeclarationsInfo struct {
	parsedFlagsFile android.Path
}

var allAconfigDeclarationsInfoProvider = blueprint.NewProvider[allAconfigDeclarationsInfo]()

type allAconfigReleaseDeclarationsSingleton struct {
	intermediateBinaryProtoPath android.OutputPath
	intermediateTextProtoPath   android.OutputPath
}

type ApiSurfaceContributorProperties struct {
	Api_signature_files  proptools.Configurable[[]string] `android:"arch_variant,path"`
	Finalized_flags_file string                           `android:"arch_variant,path"`
}

type allAconfigDeclarationsSingleton struct {
	android.SingletonModuleBase

	releaseMap map[string]allAconfigReleaseDeclarationsSingleton
	properties ApiSurfaceContributorProperties

	finalizedFlags android.OutputPath
}

func (this *allAconfigDeclarationsSingleton) sortedConfigNames() []string {
	var names []string
	for k := range this.releaseMap {
		names = append(names, k)
	}
	slices.Sort(names)
	return names
}

func GenerateFinalizedFlagsForApiSurface(ctx android.ModuleContext, outputPath android.WritablePath,
	parsedFlagsFile android.Path, apiSurface ApiSurfaceContributorProperties) {

	apiSignatureFiles := android.Paths{}
	for _, apiSignatureFile := range apiSurface.Api_signature_files.GetOrDefault(ctx, nil) {
		if path := android.PathForModuleSrc(ctx, apiSignatureFile); path != nil {
			apiSignatureFiles = append(apiSignatureFiles, path)
		}
	}
	finalizedFlagsFile := android.PathForModuleSrc(ctx, apiSurface.Finalized_flags_file)

	ctx.Build(pctx, android.BuildParams{
		Rule:   RecordFinalizedFlagsRule,
		Inputs: append(apiSignatureFiles, finalizedFlagsFile, parsedFlagsFile),
		Output: outputPath,
		Args: map[string]string{
			"api_signature_files":  android.JoinPathsWithPrefix(apiSignatureFiles, "--api-signature-file "),
			"finalized_flags_file": "--finalized-flags-file " + finalizedFlagsFile.String(),
			"parsed_flags_file":    "--parsed-flags-file " + parsedFlagsFile.String(),
		},
	})
}

func GenerateExportedFlagCheck(ctx android.ModuleContext, outputPath android.WritablePath,
	parsedFlagsFile android.Path, apiSurface ApiSurfaceContributorProperties) {

	apiSignatureFiles := android.Paths{}
	for _, apiSignatureFile := range apiSurface.Api_signature_files.GetOrDefault(ctx, nil) {
		if path := android.PathForModuleSrc(ctx, apiSignatureFile); path != nil {
			apiSignatureFiles = append(apiSignatureFiles, path)
		}
	}
	finalizedFlagsFile := android.PathForModuleSrc(ctx, apiSurface.Finalized_flags_file)

	ctx.Build(pctx, android.BuildParams{
		Rule:   ExportedFlagCheckRule,
		Inputs: append(apiSignatureFiles, finalizedFlagsFile, parsedFlagsFile),
		Output: outputPath,
		Args: map[string]string{
			"api_signature_files":  android.JoinPathsWithPrefix(apiSignatureFiles, "--api-signature-file "),
			"finalized_flags_file": "--finalized-flags-file " + finalizedFlagsFile.String(),
			"parsed_flags_file":    "--parsed-flags-file " + parsedFlagsFile.String(),
		},
	})
}

func (this *allAconfigDeclarationsSingleton) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	parsedFlagsFile := android.PathForIntermediates(ctx, "all_aconfig_declarations.pb")
	this.finalizedFlags = android.PathForIntermediates(ctx, "finalized-flags.txt")
	GenerateFinalizedFlagsForApiSurface(ctx, this.finalizedFlags, parsedFlagsFile, this.properties)

	depsFiles := android.Paths{this.finalizedFlags}
	if checkExportedFlag, ok := ctx.Config().GetBuildFlag("RELEASE_EXPORTED_FLAG_CHECK"); ok {
		if checkExportedFlag == "true" {
			invalidExportedFlags := android.PathForIntermediates(ctx, "invalid_exported_flags.txt")
			GenerateExportedFlagCheck(ctx, invalidExportedFlags, parsedFlagsFile, this.properties)
			depsFiles = append(depsFiles, invalidExportedFlags)
			ctx.Phony("droidcore", invalidExportedFlags)
		}
	}

	ctx.Phony("all_aconfig_declarations", depsFiles...)

	android.SetProvider(ctx, allAconfigDeclarationsInfoProvider, allAconfigDeclarationsInfo{
		parsedFlagsFile: parsedFlagsFile,
	})
}

func (this *allAconfigDeclarationsSingleton) GenerateSingletonBuildActions(ctx android.SingletonContext) {
	for _, rcName := range append([]string{""}, ctx.Config().ReleaseAconfigExtraReleaseConfigs()...) {
		// Find all of the aconfig_declarations modules
		var packages = make(map[string]int)
		var cacheFiles android.Paths
		ctx.VisitAllModuleProxies(func(module android.ModuleProxy) {
			decl, ok := android.OtherModuleProvider(ctx, module, android.AconfigReleaseDeclarationsProviderKey)
			if !ok {
				return
			}
			cacheFiles = append(cacheFiles, decl[rcName].IntermediateCacheOutputPath)
			packages[decl[rcName].Package]++
		})

		var numOffendingPkg = 0
		offendingPkgsMessage := ""
		for pkg, cnt := range packages {
			if cnt > 1 {
				offendingPkgsMessage += fmt.Sprintf("%d aconfig_declarations found for package %s\n", cnt, pkg)
				numOffendingPkg++
			}
		}

		if numOffendingPkg > 0 {
			panic("Only one aconfig_declarations allowed for each package.\n" + offendingPkgsMessage)
		}

		// Generate build action for aconfig (binary proto output)
		paths := allAconfigReleaseDeclarationsSingleton{
			intermediateBinaryProtoPath: android.PathForIntermediates(ctx, assembleFileName(rcName, "all_aconfig_declarations.pb")),
			intermediateTextProtoPath:   android.PathForIntermediates(ctx, assembleFileName(rcName, "all_aconfig_declarations.textproto")),
		}
		this.releaseMap[rcName] = paths
		ctx.Build(pctx, android.BuildParams{
			Rule:        AllDeclarationsRule,
			Inputs:      cacheFiles,
			Output:      this.releaseMap[rcName].intermediateBinaryProtoPath,
			Description: "all_aconfig_declarations",
			Args: map[string]string{
				"cache_files": android.JoinPathsWithPrefix(cacheFiles, "--cache "),
			},
		})
		ctx.Phony("all_aconfig_declarations", this.releaseMap[rcName].intermediateBinaryProtoPath)

		// Generate build action for aconfig (text proto output)
		ctx.Build(pctx, android.BuildParams{
			Rule:        AllDeclarationsRuleTextProto,
			Inputs:      cacheFiles,
			Output:      this.releaseMap[rcName].intermediateTextProtoPath,
			Description: "all_aconfig_declarations_textproto",
			Args: map[string]string{
				"cache_files": android.JoinPathsWithPrefix(cacheFiles, "--cache "),
			},
		})
		ctx.Phony("all_aconfig_declarations_textproto", this.releaseMap[rcName].intermediateTextProtoPath)
	}

	for _, rcName := range this.sortedConfigNames() {
		ctx.DistForGoal("droid", this.releaseMap[rcName].intermediateBinaryProtoPath)
		for _, goal := range []string{"docs", "droid", "sdk"} {
			ctx.DistForGoalWithFilename(goal, this.releaseMap[rcName].intermediateBinaryProtoPath, assembleFileName(rcName, "flags.pb"))
			ctx.DistForGoalWithFilename(goal, this.releaseMap[rcName].intermediateTextProtoPath, assembleFileName(rcName, "flags.textproto"))
		}
	}
	ctx.DistForGoalWithFilename("sdk", this.finalizedFlags, "finalized-flags.txt")
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
}# include <iostream>


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

                    