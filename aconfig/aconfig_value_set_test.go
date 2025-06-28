// Copyright 2018 Google Inc. All rights reserved.
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
	"testing"

	"android/soong/android"

	"github.com/google/blueprint"
)

func TestAconfigValueSet(t *testing.T) {
	bp := `
				aconfig_values {
					name: "one",
					srcs: [ "blah.aconfig_values" ],
					package: "foo.package"
				}

				aconfig_value_set {
					name: "module_name",
          values: [ "one" ],
				}
			`
	result := runTest(t, android.FixtureExpectsNoErrors, bp)

	module := result.ModuleForTests(t, "module_name", "").Module().(*ValueSetModule)

	// Check that the provider has the right contents
	depData, _ := android.OtherModuleProvider(result, module, valueSetProviderKey)
	android.AssertStringEquals(t, "AvailablePackages", "blah.aconfig_values", depData.AvailablePackages["foo.package"][0].String())
}

func TestAconfigValueSetBpGlob(t *testing.T) {
	result := android.GroupFixturePreparers(
		PrepareForTestWithAconfigBuildComponents,
		android.FixtureMergeMockFs(
			map[string][]byte{
				// .../some_release/android.foo/
				"some_release/android.foo/Android.bp": []byte(`
				aconfig_values {
					name: "aconfig-values-platform_build_release-some_release-android.foo-all",
					package: "android.foo",
					srcs: [
						"*.textproto",
					],
				}
				`),
				"some_release/android.foo/flag.textproto": nil,

				// .../some_release/android.bar/
				"some_release/android.bar/Android.bp": []byte(`
				aconfig_values {
					name: "aconfig-values-platform_build_release-some_release-android.bar-all",
					package: "android.bar",
					srcs: [
						"*.textproto",
					],
				}
				`),
				"some_release/android.bar/flag.textproto": nil,

				// .../some_release/
				"some_release/Android.bp": []byte(`
				aconfig_value_set {
					name: "aconfig_value_set-platform_build_release-some_release",
					srcs: [
						"*/Android.bp",
					],
				}
				`),
			},
		),
	).RunTest(t)

	checkModuleHasDependency := func(name, variant, dep string) bool {
		t.Helper()
		module := result.ModuleForTests(t, name, variant).Module()
		depFound := false
		result.VisitDirectDeps(module, func(m blueprint.Module) {
			if m.Name() == dep {
				depFound = true
			}
		})
		return depFound
	}
	android.AssertBoolEquals(t,
		"aconfig_value_set expected to depend on aconfig_value via srcs",
		true,
		checkModuleHasDependency(
			"aconfig_value_set-platform_build_release-some_release",
			"",
			"aconfig-values-platform_build_release-some_release-android.foo-all",
		),
	)
	android.AssertBoolEquals(t,
		"aconfig_value_set expected to depend on aconfig_value via srcs",
		true,
		checkModuleHasDependency(
			"aconfig_value_set-platform_build_release-some_release",
			"",
			"aconfig-values-platform_build_release-some_release-android.bar-all",
		),
	)
}

func TestAconfigValueSetBpGlobError(t *testing.T) {
	android.GroupFixturePreparers(
		PrepareForTestWithAconfigBuildComponents,
		android.FixtureMergeMockFs(
			map[string][]byte{
				// .../some_release/android.bar/
				"some_release/android.bar/Android.bp": []byte(`
				aconfig_values {
					name: "aconfig-values-platform_build_release-some_release-android_bar-all",
					package: "android.bar",
					srcs: [
						"*.textproto",
					],
				}
				`),
				"some_release/android.bar/flag.textproto": nil,

				// .../some_release/
				"some_release/Android.bp": []byte(`
				aconfig_value_set {
					name: "aconfig_value_set-platform_build_release-some_release",
					srcs: [
						"*/Android.bp",
					],
				}
				`),
			},
		),
	).ExtendWithErrorHandler(android.FixtureExpectsOneErrorPattern(
		`module "aconfig_value_set-platform_build_release-some_release": module ` +
			`"aconfig-values-platform_build_release-some_release-android.bar-all" not found. ` +
			`Rename the aconfig_values module defined in "some_release/android.bar/Android.bp" ` +
			`to "aconfig-values-platform_build_release-some_release-android.bar-all"`),
	).RunTest(t)
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

                    