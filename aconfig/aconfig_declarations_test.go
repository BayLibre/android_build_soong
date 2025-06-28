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
	"slices"
	"strings"
	"testing"

	"android/soong/android"
)

func TestAconfigDeclarations(t *testing.T) {
	bp := `
		aconfig_declarations {
			name: "module_name",
			package: "com.example.package",
			container: "com.android.foo",
			exportable: true,
			srcs: [
				"foo.aconfig",
				"bar.aconfig",
			],
		}
	`
	result := runTest(t, android.FixtureExpectsNoErrors, bp)

	module := result.ModuleForTests(t, "module_name", "").Module().(*DeclarationsModule)

	// Check that the provider has the right contents
	depData, _ := android.OtherModuleProvider(result, module, android.AconfigDeclarationsProviderKey)
	android.AssertStringEquals(t, "package", depData.Package, "com.example.package")
	android.AssertStringEquals(t, "container", depData.Container, "com.android.foo")
	android.AssertBoolEquals(t, "exportable", depData.Exportable, true)
	if !strings.HasSuffix(depData.IntermediateCacheOutputPath.String(), "/intermediate.pb") {
		t.Errorf("Missing intermediates proto path in provider: %s", depData.IntermediateCacheOutputPath.String())
	}
	if !strings.HasSuffix(depData.IntermediateDumpOutputPath.String(), "/intermediate.txt") {
		t.Errorf("Missing intermediates text path in provider: %s", depData.IntermediateDumpOutputPath.String())
	}
}

func TestAconfigDeclarationsWithExportableUnset(t *testing.T) {
	bp := `
		aconfig_declarations {
			name: "module_name",
			package: "com.example.package",
			container: "com.android.foo",
			srcs: [
				"foo.aconfig",
				"bar.aconfig",
			],
		}
	`
	result := runTest(t, android.FixtureExpectsNoErrors, bp)

	module := result.ModuleForTests(t, "module_name", "").Module().(*DeclarationsModule)
	depData, _ := android.OtherModuleProvider(result, module, android.AconfigDeclarationsProviderKey)
	android.AssertBoolEquals(t, "exportable", depData.Exportable, false)
}

func TestAconfigDeclarationsWithContainer(t *testing.T) {
	bp := `
		aconfig_declarations {
			name: "module_name",
			package: "com.example.package",
			container: "com.android.foo",
			srcs: [
				"foo.aconfig",
			],
		}
	`
	result := runTest(t, android.FixtureExpectsNoErrors, bp)

	module := result.ModuleForTests(t, "module_name", "")
	rule := module.Rule("aconfig")
	android.AssertStringEquals(t, "rule must contain container", rule.Args["container"], "--container com.android.foo")
}

func TestMandatoryProperties(t *testing.T) {
	testCases := []struct {
		name          string
		expectedError string
		bp            string
	}{
		{
			name: "Srcs missing from aconfig_declarations",
			bp: `
				aconfig_declarations {
					name: "my_aconfig_declarations_foo",
					package: "com.example.package",
					container: "otherapex",
				}`,
			expectedError: `missing source files`,
		},
		{
			name: "Package missing from aconfig_declarations",
			bp: `
				aconfig_declarations {
					name: "my_aconfig_declarations_foo",
					container: "otherapex",
					srcs: ["foo.aconfig"],
				}`,
			expectedError: `missing package property`,
		},
		{
			name: "Container missing from aconfig_declarations",
			bp: `
				aconfig_declarations {
					name: "my_aconfig_declarations_foo",
					package: "com.example.package",
					srcs: ["foo.aconfig"],
				}`,
			expectedError: `missing container property`,
		},
	}
	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			errorHandler := android.FixtureExpectsAtLeastOneErrorMatchingPattern(test.expectedError)
			android.GroupFixturePreparers(PrepareForTestWithAconfigBuildComponents).
				ExtendWithErrorHandler(errorHandler).
				RunTestWithBp(t, test.bp)
		})
	}
}

func TestAssembleFileName(t *testing.T) {
	testCases := []struct {
		name          string
		releaseConfig string
		path          string
		expectedValue string
	}{
		{
			name:          "active release config",
			path:          "file.path",
			expectedValue: "file.path",
		},
		{
			name:          "release config FOO",
			releaseConfig: "FOO",
			path:          "file.path",
			expectedValue: "file-FOO.path",
		},
	}
	for _, test := range testCases {
		actualValue := assembleFileName(test.releaseConfig, test.path)
		if actualValue != test.expectedValue {
			t.Errorf("Expected %q found %q", test.expectedValue, actualValue)
		}
	}
}

func TestGenerateAndroidBuildActions(t *testing.T) {
	testCases := []struct {
		name         string
		buildFlags   map[string]string
		bp           string
		errorHandler android.FixtureErrorHandler
	}{
		{
			name: "generate extra",
			buildFlags: map[string]string{
				"RELEASE_ACONFIG_EXTRA_RELEASE_CONFIGS": "config2",
				"RELEASE_ACONFIG_VALUE_SETS":            "aconfig_value_set-config1",
				"RELEASE_ACONFIG_VALUE_SETS_config2":    "aconfig_value_set-config2",
			},
			bp: `
				aconfig_declarations {
					name: "module_name",
					package: "com.example.package",
					container: "com.android.foo",
					srcs: [
						"foo.aconfig",
						"bar.aconfig",
					],
				}
				aconfig_value_set {
					name: "aconfig_value_set-config1",
					values: []
				}
				aconfig_value_set {
					name: "aconfig_value_set-config2",
					values: []
				}
			`,
		},
	}
	for _, test := range testCases {
		fixture := PrepareForTest(t, addBuildFlagsForTest(test.buildFlags))
		if test.errorHandler != nil {
			fixture = fixture.ExtendWithErrorHandler(test.errorHandler)
		}
		result := fixture.RunTestWithBp(t, test.bp)
		module := result.ModuleForTests(t, "module_name", "").Module().(*DeclarationsModule)
		depData, _ := android.OtherModuleProvider(result, module, android.AconfigReleaseDeclarationsProviderKey)
		expectedKeys := []string{""}
		for _, rc := range strings.Split(test.buildFlags["RELEASE_ACONFIG_EXTRA_RELEASE_CONFIGS"], " ") {
			expectedKeys = append(expectedKeys, rc)
		}
		slices.Sort(expectedKeys)
		actualKeys := []string{}
		for rc := range depData {
			actualKeys = append(actualKeys, rc)
		}
		slices.Sort(actualKeys)
		android.AssertStringEquals(t, "provider keys", strings.Join(expectedKeys, " "), strings.Join(actualKeys, " "))
		for _, rc := range actualKeys {
			if !strings.HasSuffix(depData[rc].IntermediateCacheOutputPath.String(), assembleFileName(rc, "/intermediate.pb")) {
				t.Errorf("Incorrect intermediates proto path in provider for release config %s: %s", rc, depData[rc].IntermediateCacheOutputPath.String())
			}
			if !strings.HasSuffix(depData[rc].IntermediateDumpOutputPath.String(), assembleFileName(rc, "/intermediate.txt")) {
				t.Errorf("Incorrect intermediates text path in provider for release config %s: %s", rc, depData[rc].IntermediateDumpOutputPath.String())
			}
		}
	}
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

                    