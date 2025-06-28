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

package build_flags

import (
	"path/filepath"

	"android/soong/android"

	"github.com/google/blueprint"
)

type ReleaseConfigContributionsProviderData struct {
	ContributionDir android.SourcePath
}

var ReleaseConfigContributionsProviderKey = blueprint.NewProvider[ReleaseConfigContributionsProviderData]()

// Soong uses `release_config_contributions` modules to produce the
// `build_flags/all_release_config_contributions.*` artifacts, listing *all* of
// the directories in the source tree that contribute to each release config,
// whether or not they are actually used for the lunch product.
//
// This artifact helps flagging automation determine in which directory a flag
// should be placed by default.
type ReleaseConfigContributionsModule struct {
	android.ModuleBase
	android.DefaultableModuleBase

	// Properties for "release_config_contributions"
	properties struct {
		// The `release_configs/*.textproto` files provided by this
		// directory, relative to this Android.bp file
		Srcs []string `android:"path"`
	}
}

func ReleaseConfigContributionsFactory() android.Module {
	module := &ReleaseConfigContributionsModule{}

	android.InitAndroidModule(module)
	android.InitDefaultableModule(module)
	module.AddProperties(&module.properties)

	return module
}

func (module *ReleaseConfigContributionsModule) GenerateAndroidBuildActions(ctx android.ModuleContext) {
	srcs := android.PathsForModuleSrc(ctx, module.properties.Srcs)
	if len(srcs) == 0 {
		return
	}
	contributionDir := filepath.Dir(filepath.Dir(srcs[0].String()))
	for _, file := range srcs {
		if filepath.Dir(filepath.Dir(file.String())) != contributionDir {
			ctx.ModuleErrorf("Cannot include %s with %s contributions", file, contributionDir)
		}
		if filepath.Base(filepath.Dir(file.String())) != "release_configs" || file.Ext() != ".textproto" {
			ctx.ModuleErrorf("Invalid contribution file %s", file)
		}
	}
	android.SetProvider(ctx, ReleaseConfigContributionsProviderKey, ReleaseConfigContributionsProviderData{
		ContributionDir: android.PathForSource(ctx, contributionDir),
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

                    