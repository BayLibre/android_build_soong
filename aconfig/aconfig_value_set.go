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
	"fmt"
	"strings"

	"github.com/google/blueprint"
)

// Properties for "aconfig_value_set"
type ValueSetModule struct {
	android.ModuleBase
	android.DefaultableModuleBase

	properties struct {
		// aconfig_values modules
		Values []string

		// Paths to the Android.bp files where the aconfig_values modules are defined.
		Srcs []string
	}
}

func ValueSetFactory() android.Module {
	module := &ValueSetModule{}

	android.InitAndroidModule(module)
	android.InitDefaultableModule(module)
	module.AddProperties(&module.properties)

	return module
}

// Dependency tag for values property
type valueSetType struct {
	blueprint.BaseDependencyTag
}

var valueSetTag = valueSetType{}

// Provider published by aconfig_value_set
type valueSetProviderData struct {
	// The package of each of the
	// (map of package --> aconfig_module)
	AvailablePackages map[string]android.Paths
}

var valueSetProviderKey = blueprint.NewProvider[valueSetProviderData]()

func (module *ValueSetModule) FindAconfigValuesFromSrc(ctx android.BottomUpMutatorContext) map[string]android.Path {
	moduleDir := ctx.ModuleDir()
	srcs := android.PathsForModuleSrcExcludes(ctx, module.properties.Srcs, []string{ctx.BlueprintsFile()})

	aconfigValuesPrefix := strings.Replace(module.Name(), "aconfig_value_set", "aconfig-values", 1)
	moduleNamesSrcMap := make(map[string]android.Path)
	for _, src := range srcs {
		subDir := strings.TrimPrefix(src.String(), moduleDir+"/")
		packageName, _, found := strings.Cut(subDir, "/")
		if found {
			moduleName := fmt.Sprintf("%s-%s-all", aconfigValuesPrefix, packageName)
			moduleNamesSrcMap[moduleName] = src
		}
	}
	return moduleNamesSrcMap
}

func (module *ValueSetModule) DepsMutator(ctx android.BottomUpMutatorContext) {

	// TODO: b/366285733 - Replace the file path based solution with more robust solution.
	aconfigValuesMap := module.FindAconfigValuesFromSrc(ctx)
	for _, moduleName := range android.SortedKeys(aconfigValuesMap) {
		if ctx.OtherModuleExists(moduleName) {
			ctx.AddDependency(ctx.Module(), valueSetTag, moduleName)
		} else {
			ctx.ModuleErrorf("module %q not found. Rename the aconfig_values module defined in %q to %q", moduleName, aconfigValuesMap[moduleName], moduleName)
		}
	}

	deps := ctx.AddDependency(ctx.Module(), valueSetTag, module.properties.Values...)
	for _, dep := range deps {
		_, ok := dep.(*ValuesModule)
		if !ok {
			ctx.PropertyErrorf("values", "values must be a aconfig_values module")
			return
		}
	}
}

func (module *ValueSetModule) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	// Accumulate the packages of the values modules listed, and set that as an
	// valueSetProviderKey provider that aconfig modules can read and use
	// to append values to their aconfig actions.
	packages := make(map[string]android.Paths)
	ctx.VisitDirectDeps(func(dep android.Module) {
		if depData, ok := android.OtherModuleProvider(ctx, dep, valuesProviderKey); ok {
			srcs := make([]android.Path, len(depData.Values))
			copy(srcs, depData.Values)
			packages[depData.Package] = srcs
		}

	})
	android.SetProvider(ctx, valueSetProviderKey, valueSetProviderData{
		AvailablePackages: packages,
	})
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

                    