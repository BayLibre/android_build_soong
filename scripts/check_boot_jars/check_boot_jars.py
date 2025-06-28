#!/usr/bin/env python
"""Check boot jars.

Usage: check_boot_jars.py <dexdump_path> <package_allow_list_file> <jar1> \
<jar2> ...
"""
import logging
import re
import subprocess
import sys
import xml.etree.ElementTree

# The compiled allow list RE.
allow_list_re = None


def LoadAllowList(filename):
    """ Load and compile allow list regular expressions from filename."""
    lines = []
    with open(filename, 'r') as f:
        for line in f:
            line = line.strip()
            if not line or line.startswith('#'):
                continue
            lines.append(line)
    combined_re = r'^(%s)$' % '|'.join(lines)
    global allow_list_re #pylint: disable=global-statement
    try:
        allow_list_re = re.compile(combined_re)
    except re.error:
        logging.exception(
            'Cannot compile package allow list regular expression: %r',
            combined_re)
        allow_list_re = None
        return False
    return True

def CheckDexJar(dexdump_path, allow_list_path, jar):
    """Check a dex jar file."""
    # Use dexdump to generate the XML representation of the dex jar file.
    p = subprocess.Popen(
        args='%s -l xml %s' % (dexdump_path, jar),
        stdout=subprocess.PIPE,
        shell=True)
    stdout, _ = p.communicate()
    if p.returncode != 0:
        return False

    packages = 0
    try:
        # TODO(b/172063475) - improve performance
        root = xml.etree.ElementTree.fromstring(stdout)
    except xml.etree.ElementTree.ParseError as e:
        print('Error processing jar %s - %s' % (jar, e), file=sys.stderr)
        print(stdout, file=sys.stderr)
        return False
    for package_elt in root.iterfind('package'):
        packages += 1
        package_name = package_elt.get('name')
        if not package_name or not allow_list_re.match(package_name):
            # Report the name of a class in the package as it is easier to
            # navigate to the source of a concrete class than to a package
            # which is often required to investigate this failure.
            class_name = package_elt[0].get('name')
            if package_name:
                class_name = package_name + '.' + class_name
            print((
                'Error: %s contains class file %s, whose package name "%s" is '
                'empty or not in the allow list %s of packages allowed on the '
                'bootclasspath.'
                % (jar, class_name, package_name, allow_list_path)),
                  file=sys.stderr)
            return False
    if packages == 0:
        print(('Error: %s does not contain any packages.' % jar),
              file=sys.stderr)
        return False
    return True

def main(argv):
    if len(argv) < 3:
        print(__doc__)
        return 1
    dexdump_path = argv[0]
    allow_list_path = argv[1]

    if not LoadAllowList(allow_list_path):
        return 1

    passed = True
    for jar in argv[2:]:
        if not CheckDexJar(dexdump_path, allow_list_path, jar):
            passed = False
    if not passed:
        return 1

    return 0


if __name__ == '__main__':
    sys.exit(main(sys.argv[1:]))
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