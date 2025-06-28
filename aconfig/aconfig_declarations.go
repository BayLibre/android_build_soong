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
	"android/soong/android"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/google/blueprint"
)

type AconfigReleaseConfigValue struct {
	ReleaseConfig string
	Values        []string `blueprint:"mutated"`
}

type DeclarationsModule struct {
	android.ModuleBase
	android.DefaultableModuleBase
	blueprint.IncrementalModule

	// Properties for "aconfig_declarations"
	properties struct {
		// aconfig files, relative to this Android.bp file
		Srcs []string `android:"path"`

		// Release config flag package
		Package string

		// Values for release configs / RELEASE_ACONFIG_VALUE_SETS
		// The current release config is `ReleaseConfig: ""`, others
		// are from RELEASE_ACONFIG_EXTRA_RELEASE_CONFIGS.
		ReleaseConfigValues []AconfigReleaseConfigValue

		// Container(system/system_ext/vendor/apex) that this module belongs to
		Container string

		// The flags will only be repackaged if this prop is true.
		Exportable bool
	}
}

func DeclarationsFactory() android.Module {
	module := &DeclarationsModule{}

	android.InitAndroidModule(module)
	android.InitDefaultableModule(module)
	module.AddProperties(&module.properties)

	return module
}

type implicitValuesTagType struct {
	blueprint.BaseDependencyTag

	// The release config name for these values.
	// Empty string for the actual current release config.
	ReleaseConfig string
}

var implicitValuesTag = implicitValuesTagType{}

func (module *DeclarationsModule) DepsMutator(ctx android.BottomUpMutatorContext) {
	// Validate Properties
	if len(module.properties.Srcs) == 0 {
		ctx.PropertyErrorf("srcs", "missing source files")
		return
	}
	if len(module.properties.Package) == 0 {
		ctx.PropertyErrorf("package", "missing package property")
	}
	if len(module.properties.Container) == 0 {
		ctx.PropertyErrorf("container", "missing container property")
	}

	// Add a dependency on the aconfig_value_sets defined in
	// RELEASE_ACONFIG_VALUE_SETS, and add any aconfig_values that
	// match our package.
	valuesFromConfig := ctx.Config().ReleaseAconfigValueSets()
	if len(valuesFromConfig) > 0 {
		ctx.AddDependency(ctx.Module(), implicitValuesTag, valuesFromConfig...)
	}
	for rcName, valueSets := range ctx.Config().ReleaseAconfigExtraReleaseConfigsValueSets() {
		if len(valueSets) > 0 {
			ctx.AddDependency(ctx.Module(), implicitValuesTagType{ReleaseConfig: rcName}, valueSets...)
		}
	}
}

func joinAndPrefix(prefix string, values []string) string {
	var sb strings.Builder
	for _, v := range values {
		sb.WriteString(prefix)
		sb.WriteString(v)
	}
	return sb.String()
}

func optionalVariable(prefix string, value string) string {
	var sb strings.Builder
	if value != "" {
		sb.WriteString(prefix)
		sb.WriteString(value)
	}
	return sb.String()
}

// Assemble the actual filename.
// If `rcName` is not empty, then insert "-{rcName}" into the path before the
// file extension.
func assembleFileName(rcName, path string) string {
	if rcName == "" {
		return path
	}
	dir, file := filepath.Split(path)
	rcName = "-" + rcName
	ext := filepath.Ext(file)
	base := file[:len(file)-len(ext)]
	return dir + base + rcName + ext
}

func (module *DeclarationsModule) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	// Determine which release configs we are processing.
	//
	// We always process the current release config (empty string).
	// We may have been told to also create artifacts for some others.
	configs := append([]string{""}, ctx.Config().ReleaseAconfigExtraReleaseConfigs()...)
	slices.Sort(configs)

	values := make(map[string][]string)
	valuesFiles := make(map[string][]android.Path, 0)
	providerData := android.AconfigReleaseDeclarationsProviderData{}
	ctx.VisitDirectDeps(func(dep android.Module) {
		if depData, ok := android.OtherModuleProvider(ctx, dep, valueSetProviderKey); ok {
			depTag := ctx.OtherModuleDependencyTag(dep)
			for _, config := range configs {
				tag := implicitValuesTagType{ReleaseConfig: config}
				if depTag == tag {
					paths, ok := depData.AvailablePackages[module.properties.Package]
					if ok {
						valuesFiles[config] = append(valuesFiles[config], paths...)
						for _, path := range paths {
							values[config] = append(values[config], path.String())
						}
					}
				}
			}
		}
	})
	for _, config := range configs {
		module.properties.ReleaseConfigValues = append(module.properties.ReleaseConfigValues, AconfigReleaseConfigValue{
			ReleaseConfig: config,
			Values:        values[config],
		})

		// Intermediate format
		declarationFiles := android.PathsForModuleSrc(ctx, module.properties.Srcs)
		intermediateCacheFilePath := android.PathForModuleOut(ctx, assembleFileName(config, "intermediate.pb"))
		var defaultPermission string
		defaultPermission = ctx.Config().ReleaseAconfigFlagDefaultPermission()
		if config != "" {
			if confPerm, ok := ctx.Config().GetBuildFlag("RELEASE_ACONFIG_FLAG_DEFAULT_PERMISSION_" + config); ok {
				defaultPermission = confPerm
			}
		}
		var allowReadWrite bool
		if requireAllReadOnly, ok := ctx.Config().GetBuildFlag("RELEASE_ACONFIG_REQUIRE_ALL_READ_ONLY"); ok {
			// The build flag (RELEASE_ACONFIG_REQUIRE_ALL_READ_ONLY) is the negation of the aconfig flag
			// (allow-read-write) for historical reasons.
			// Bool build flags are always "" for false, and generally "true" for true.
			allowReadWrite = requireAllReadOnly == ""
		}
		inputFiles := make([]android.Path, len(declarationFiles))
		copy(inputFiles, declarationFiles)
		inputFiles = append(inputFiles, valuesFiles[config]...)
		args := map[string]string{
			"release_version":    ctx.Config().ReleaseVersion(),
			"package":            module.properties.Package,
			"declarations":       android.JoinPathsWithPrefix(declarationFiles, "--declarations "),
			"values":             joinAndPrefix(" --values ", values[config]),
			"default-permission": optionalVariable(" --default-permission ", defaultPermission),
			"allow-read-write":   optionalVariable(" --allow-read-write ", strconv.FormatBool(allowReadWrite)),
		}
		if len(module.properties.Container) > 0 {
			args["container"] = "--container " + module.properties.Container
		}
		ctx.Build(pctx, android.BuildParams{
			Rule:        aconfigRule,
			Output:      intermediateCacheFilePath,
			Inputs:      inputFiles,
			Description: "aconfig_declarations",
			Args:        args,
		})

		intermediateDumpFilePath := android.PathForModuleOut(ctx, assembleFileName(config, "intermediate.txt"))
		ctx.Build(pctx, android.BuildParams{
			Rule:        aconfigTextRule,
			Output:      intermediateDumpFilePath,
			Inputs:      android.Paths{intermediateCacheFilePath},
			Description: "aconfig_text",
		})

		providerData[config] = android.AconfigDeclarationsProviderData{
			Package:                     module.properties.Package,
			Container:                   module.properties.Container,
			Exportable:                  module.properties.Exportable,
			IntermediateCacheOutputPath: intermediateCacheFilePath,
			IntermediateDumpOutputPath:  intermediateDumpFilePath,
		}
	}
	android.SetProvider(ctx, android.AconfigDeclarationsProviderKey, providerData[""])
	android.SetProvider(ctx, android.AconfigReleaseDeclarationsProviderKey, providerData)
}

var _ blueprint.Incremental = &DeclarationsModule{}
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

                    