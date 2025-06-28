// Copyright 2022 Google Inc. All rights reserved.
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
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

// The ConfiguredJarList struct provides methods for handling a list of (apex, jar) pairs.
// Such lists are used in the build system for things like bootclasspath jars or system server jars.
// The apex part is either an apex name, or a special names "platform" or "system_ext". Jar is a
// module name. The pairs come from Make product variables as a list of colon-separated strings.
//
// Examples:
//   - "com.android.art:core-oj"
//   - "platform:framework"
//   - "system_ext:foo"
type ConfiguredJarList struct {
	// A list of apex components, which can be an apex name,
	// or special names like "platform" or "system_ext".
	apexes []string

	// A list of jar module name components.
	jars []string
}

// Len returns the length of the list of jars.
func (l *ConfiguredJarList) Len() int {
	return len(l.jars)
}

// Jar returns the idx-th jar component of (apex, jar) pairs.
func (l *ConfiguredJarList) Jar(idx int) string {
	return l.jars[idx]
}

// Apex returns the idx-th apex component of (apex, jar) pairs.
func (l *ConfiguredJarList) Apex(idx int) string {
	return l.apexes[idx]
}

// ContainsJar returns true if the (apex, jar) pairs contains a pair with the
// given jar module name.
func (l *ConfiguredJarList) ContainsJar(jar string) bool {
	return InList(jar, l.jars)
}

// If the list contains the given (apex, jar) pair.
func (l *ConfiguredJarList) containsApexJarPair(apex, jar string) bool {
	for i := 0; i < l.Len(); i++ {
		if apex == l.apexes[i] && jar == l.jars[i] {
			return true
		}
	}
	return false
}

// ApexOfJar returns the apex component of the first pair with the given jar name on the list, or
// an empty string if not found.
func (l *ConfiguredJarList) ApexOfJar(jar string) string {
	if idx := IndexList(jar, l.jars); idx != -1 {
		return l.Apex(IndexList(jar, l.jars))
	}
	return ""
}

// IndexOfJar returns the first pair with the given jar name on the list, or -1
// if not found.
func (l *ConfiguredJarList) IndexOfJar(jar string) int {
	return IndexList(jar, l.jars)
}

func copyAndAppend(list []string, item string) []string {
	// Create the result list to be 1 longer than the input.
	result := make([]string, len(list)+1)

	// Copy the whole input list into the result.
	count := copy(result, list)

	// Insert the extra item at the end.
	result[count] = item

	return result
}

// Append an (apex, jar) pair to the list.
func (l *ConfiguredJarList) Append(apex string, jar string) ConfiguredJarList {
	// Create a copy of the backing arrays before appending to avoid sharing backing
	// arrays that are mutated across instances.
	apexes := copyAndAppend(l.apexes, apex)
	jars := copyAndAppend(l.jars, jar)

	return ConfiguredJarList{apexes, jars}
}

// Append a list of (apex, jar) pairs to the list.
func (l *ConfiguredJarList) AppendList(other *ConfiguredJarList) ConfiguredJarList {
	apexes := make([]string, 0, l.Len()+other.Len())
	jars := make([]string, 0, l.Len()+other.Len())

	apexes = append(apexes, l.apexes...)
	jars = append(jars, l.jars...)

	apexes = append(apexes, other.apexes...)
	jars = append(jars, other.jars...)

	return ConfiguredJarList{apexes, jars}
}

// RemoveList filters out a list of (apex, jar) pairs from the receiving list of pairs.
func (l *ConfiguredJarList) RemoveList(list ConfiguredJarList) ConfiguredJarList {
	apexes := make([]string, 0, l.Len())
	jars := make([]string, 0, l.Len())

	for i, jar := range l.jars {
		apex := l.apexes[i]
		if !list.containsApexJarPair(apex, jar) {
			apexes = append(apexes, apex)
			jars = append(jars, jar)
		}
	}

	return ConfiguredJarList{apexes, jars}
}

// Filter keeps the entries if a jar appears in the given list of jars to keep. Returns a new list
// and any remaining jars that are not on this list.
func (l *ConfiguredJarList) Filter(jarsToKeep []string) (ConfiguredJarList, []string) {
	var apexes []string
	var jars []string

	for i, jar := range l.jars {
		if InList(jar, jarsToKeep) {
			apexes = append(apexes, l.apexes[i])
			jars = append(jars, jar)
		}
	}

	return ConfiguredJarList{apexes, jars}, RemoveListFromList(jarsToKeep, jars)
}

// CopyOfJars returns a copy of the list of strings containing jar module name
// components.
func (l *ConfiguredJarList) CopyOfJars() []string {
	return CopyOf(l.jars)
}

// CopyOfApexJarPairs returns a copy of the list of strings with colon-separated
// (apex, jar) pairs.
func (l *ConfiguredJarList) CopyOfApexJarPairs() []string {
	pairs := make([]string, 0, l.Len())

	for i, jar := range l.jars {
		apex := l.apexes[i]
		pairs = append(pairs, apex+":"+jar)
	}

	return pairs
}

// BuildPaths returns a list of build paths based on the given directory prefix.
func (l *ConfiguredJarList) BuildPaths(ctx PathContext, dir OutputPath) WritablePaths {
	paths := make(WritablePaths, l.Len())
	for i, jar := range l.jars {
		paths[i] = dir.Join(ctx, ModuleStem(ctx.Config(), l.Apex(i), jar)+".jar")
	}
	return paths
}

// BuildPathsByModule returns a map from module name to build paths based on the given directory
// prefix.
func (l *ConfiguredJarList) BuildPathsByModule(ctx PathContext, dir OutputPath) map[string]WritablePath {
	paths := map[string]WritablePath{}
	for i, jar := range l.jars {
		paths[jar] = dir.Join(ctx, ModuleStem(ctx.Config(), l.Apex(i), jar)+".jar")
	}
	return paths
}

// UnmarshalJSON converts JSON configuration from raw bytes into a
// ConfiguredJarList structure.
func (l *ConfiguredJarList) UnmarshalJSON(b []byte) error {
	// Try and unmarshal into a []string each item of which contains a pair
	// <apex>:<jar>.
	var list []string
	err := json.Unmarshal(b, &list)
	if err != nil {
		// Did not work so return
		return err
	}

	apexes, jars, err := splitListOfPairsIntoPairOfLists(list)
	if err != nil {
		return err
	}
	l.apexes = apexes
	l.jars = jars
	return nil
}

func (l *ConfiguredJarList) MarshalJSON() ([]byte, error) {
	if len(l.apexes) != len(l.jars) {
		return nil, errors.New(fmt.Sprintf("Inconsistent ConfiguredJarList: apexes: %q, jars: %q", l.apexes, l.jars))
	}

	list := make([]string, 0, len(l.apexes))

	for i := 0; i < len(l.apexes); i++ {
		list = append(list, l.apexes[i]+":"+l.jars[i])
	}

	return json.Marshal(list)
}

func OverrideConfiguredJarLocationFor(cfg Config, apex string, jar string) (newApex string, newJar string) {
	for _, entry := range cfg.productVariables.ConfiguredJarLocationOverrides {
		tuple := strings.Split(entry, ":")
		if len(tuple) != 4 {
			panic("malformed configured jar location override '%s', expected format: <old_apex>:<old_jar>:<new_apex>:<new_jar>")
		}
		if apex == tuple[0] && jar == tuple[1] {
			return tuple[2], tuple[3]
		}
	}
	return apex, jar
}

// ModuleStem returns the overridden jar name.
func ModuleStem(cfg Config, apex string, jar string) string {
	_, newJar := OverrideConfiguredJarLocationFor(cfg, apex, jar)
	return newJar
}

// DevicePaths computes the on-device paths for the list of (apex, jar) pairs,
// based on the operating system.
func (l *ConfiguredJarList) DevicePaths(cfg Config, ostype OsType) []string {
	paths := make([]string, l.Len())
	for i := 0; i < l.Len(); i++ {
		apex, jar := OverrideConfiguredJarLocationFor(cfg, l.Apex(i), l.Jar(i))
		name := jar + ".jar"

		var subdir string
		if apex == "platform" {
			subdir = "system/framework"
		} else if apex == "system_ext" {
			subdir = "system_ext/framework"
		} else {
			subdir = filepath.Join("apex", apex, "javalib")
		}

		if ostype.Class == Host || cfg.IsEnvTrue("ART_USE_SIMULATOR") {
			paths[i] = filepath.Join(cfg.Getenv("OUT_DIR"), "host", cfg.PrebuiltOS(), subdir, name)
		} else {
			paths[i] = filepath.Join("/", subdir, name)
		}
	}
	return paths
}

func (l *ConfiguredJarList) String() string {
	var pairs []string
	for i := 0; i < l.Len(); i++ {
		pairs = append(pairs, l.apexes[i]+":"+l.jars[i])
	}
	return strings.Join(pairs, ",")
}

func splitListOfPairsIntoPairOfLists(list []string) ([]string, []string, error) {
	// Now we need to populate this list by splitting each item in the slice of
	// pairs and appending them to the appropriate list of apexes or jars.
	apexes := make([]string, len(list))
	jars := make([]string, len(list))

	for i, apexjar := range list {
		apex, jar, err := splitConfiguredJarPair(apexjar)
		if err != nil {
			return nil, nil, err
		}
		apexes[i] = apex
		jars[i] = jar
	}

	return apexes, jars, nil
}

// Expected format for apexJarValue = <apex name>:<jar name>
func splitConfiguredJarPair(str string) (string, string, error) {
	pair := strings.SplitN(str, ":", 2)
	if len(pair) == 2 {
		apex := pair[0]
		jar := pair[1]
		if apex == "" {
			return apex, jar, fmt.Errorf("invalid apex '%s' in <apex>:<jar> pair '%s', expected format: <apex>:<jar>", apex, str)
		}
		return apex, jar, nil
	} else {
		return "error-apex", "error-jar", fmt.Errorf("malformed (apex, jar) pair: '%s', expected format: <apex>:<jar>", str)
	}
}

// EmptyConfiguredJarList returns an empty jar list.
func EmptyConfiguredJarList() ConfiguredJarList {
	return ConfiguredJarList{}
}

// IsConfiguredJarForPlatform returns true if the given apex name is a special name for the platform.
func IsConfiguredJarForPlatform(apex string) bool {
	return apex == "platform" || apex == "system_ext"
}

var earlyBootJarsKey = NewOnceKey("earlyBootJars")
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

                    