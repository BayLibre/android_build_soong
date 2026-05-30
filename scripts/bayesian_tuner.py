import sys
import os
import subprocess
import json
import time
import numpy as np

# Design parameters
OUT_DIR = os.environ.get("OUT_DIR", "out")

# Define the 16 deviation features
FEATURES = [
    "-Xmx2G",                             # 0 (deviation from 4G)
    "-Xmx8G",                             # 1 (deviation from 4G)
    "-Xmx12G",                            # 2 (deviation from 4G)
    "-XX:+UseZGC",                        # 3 (deviation from G1GC)
    "-XX:+UseParallelGC",                 # 4 (deviation from G1GC)
    "-XX:TieredStopAtLevel=1",            # 5 (deviation from Level 4)
    "-XX:-TieredCompilation",             # 6 (deviation from Tiered)
    "-XX:CICompilerCount=2",              # 7 (deviation from 6)
    "-XX:CICompilerCount=4",              # 8 (deviation from 6)
    "-XX:CICompilerCount=8",              # 9 (deviation from 6)
    "-XX:+UseStringDeduplication",        # 10 (deviation from Disabled)
    "-XX:SoftRefLRUPolicyMSPerMB=10000",  # 11 (deviation from Default)
    "-XX:ParallelGCThreads=2",            # 12 (NEW: Limit GC threads)
    "-XX:CompileThreshold=2000",          # 13 (NEW: Aggressive JIT threshold)
    "-XX:+AlwaysPreTouch",                # 14 (NEW: Pre-touch heap memory)
    "-Xss512k"                            # 15 (NEW: Reduce thread stack size)
]

# Design Matrix X (17 rows: 1 baseline + 16 configs)
# Columns: [Intercept, Feature0, ..., Feature15]
X_DESIGN = np.array([
    [1, 0, 0, 0,  0, 0,  0, 0,  0, 0, 0,  0, 0,  0, 0, 0, 0], # Baseline (Config 0)
    [1, 1, 0, 0,  0, 1,  1, 0,  1, 0, 0,  0, 0,  1, 0, 0, 0], # Config 1
    [1, 0, 1, 0,  0, 0,  0, 1,  0, 0, 1,  1, 0,  0, 0, 1, 0], # Config 2
    [1, 0, 0, 1,  1, 0,  0, 1,  0, 1, 0,  0, 1,  0, 0, 0, 1], # Config 3
    [1, 0, 0, 0,  0, 1,  0, 1,  0, 1, 0,  1, 0,  0, 1, 0, 0], # Config 4
    [1, 0, 1, 0,  1, 0,  1, 0,  1, 0, 0,  0, 0,  1, 0, 0, 1], # Config 5
    [1, 1, 0, 0,  0, 0,  0, 1,  0, 0, 1,  0, 0,  0, 1, 1, 0], # Config 6
    [1, 0, 0, 1,  0, 1,  1, 0,  0, 0, 0,  1, 1,  1, 0, 0, 0], # Config 7
    [1, 0, 0, 0,  0, 0,  1, 0,  0, 1, 0,  1, 0,  0, 0, 1, 1], # Config 8
    [1, 0, 1, 0,  0, 1,  0, 1,  0, 0, 1,  0, 1,  0, 1, 0, 0], # Config 9
    [1, 0, 0, 1,  0, 0,  0, 0,  0, 1, 0,  1, 0,  1, 0, 1, 0], # Config 10
    [1, 1, 0, 0,  1, 0,  0, 1,  0, 0, 0,  1, 1,  0, 0, 0, 1], # Config 11
    [1, 0, 0, 0,  1, 0,  0, 0,  0, 0, 1,  0, 1,  1, 1, 0, 0], # Config 12
    [1, 0, 1, 0,  0, 1,  0, 0,  0, 1, 0,  0, 0,  0, 1, 1, 1], # Config 13
    [1, 0, 0, 1,  1, 0,  1, 0,  1, 0, 0,  1, 0,  1, 0, 0, 0], # Config 14
    [1, 1, 0, 0,  0, 1,  0, 1,  0, 0, 0,  1, 0,  0, 0, 1, 1], # Config 15
    [1, 0, 1, 0,  0, 0,  1, 0,  0, 0, 1,  0, 1,  1, 1, 0, 0]  # Config 16
])

# Mapping configuration indices back to actual Soong JVM flags lists
def get_soong_flags(config_idx):
    if config_idx == 0:
        return [] # Baseline
    
    vector = X_DESIGN[config_idx][1:] # Strip intercept
    flags = []
    
    # Heap Size (exclusive)
    if vector[0] == 1: flags.append("-Xmx2G")
    elif vector[1] == 1: flags.append("-Xmx8G")
    elif vector[2] == 1: flags.append("-Xmx12G")
    else: flags.append("-Xmx4G") # Default
    
    # GC (exclusive)
    if vector[3] == 1: flags.append("-XX:+UseZGC")
    elif vector[4] == 1: flags.append("-XX:+UseParallelGC")
    else: flags.append("-XX:+UseG1GC") # Default
    
    # JIT Tuning
    if vector[5] == 1: flags.append("-XX:TieredStopAtLevel=1")
    if vector[6] == 1: flags.append("-XX:-TieredCompilation")
    
    # JIT Threads
    if vector[7] == 1: flags.append("-XX:CICompilerCount=2")
    elif vector[8] == 1: flags.append("-XX:CICompilerCount=4")
    elif vector[9] == 1: flags.append("-XX:CICompilerCount=8")
    
    # Deduplication
    if vector[10] == 1: flags.append("-XX:+UseStringDeduplication")
    
    # SoftRef
    if vector[11] == 1: flags.append("-XX:SoftRefLRUPolicyMSPerMB=10000")
    
    # ParallelGCThreads (NEW)
    if vector[12] == 1: flags.append("-XX:ParallelGCThreads=2")
    
    # CompileThreshold (NEW)
    if vector[13] == 1: flags.append("-XX:CompileThreshold=2000")
    
    # AlwaysPreTouch (NEW)
    if vector[14] == 1: flags.append("-XX:+AlwaysPreTouch")
    
    # Xss (NEW)
    if vector[15] == 1: flags.append("-Xss512k")
    
    return flags

def clear_module_cache(module_name):
    cmd = f"find {OUT_DIR}/soong/.intermediates -name '{module_name}' -type d -exec rm -rf {{}} +"
    subprocess.run(cmd, shell=True, stderr=subprocess.DEVNULL)

def run_build_and_profile(module_name, flags):
    tuning_map = {module_name: flags}
    with open("jvm_tuning_map.json", "w") as f:
        json.dump(tuning_map, f)
        
    clear_module_cache(module_name)
    
    build_cmd = f"/usr/bin/time -v build/soong/bin/m {module_name}"
    env = os.environ.copy()
    
    start_t = time.time()
    result = subprocess.run(build_cmd, shell=True, capture_output=True, text=True, env=env)
    wall_time = time.time() - start_t
    
    if result.returncode != 0:
        return None, None, None
        
    peak_mem_kb = 0
    for line in result.stderr.splitlines():
        if "Maximum resident set size" in line:
            peak_mem_kb = int(line.split(":")[1].strip())
            break
            
    hash_cmd = f"find {OUT_DIR}/soong/.intermediates -path '*/{module_name}/*' \\( -name '*.jar' -o -name '*.srcjar' \\) -type f -exec sha256sum {{}} + | sort | sha256sum"
    hash_result = subprocess.run(hash_cmd, shell=True, capture_output=True, text=True)
    artifact_hash = hash_result.stdout.strip().split()[0] if hash_result.returncode == 0 else "NO_OUTPUT"
            
    return wall_time, peak_mem_kb, artifact_hash

def evaluate_config(module_name, config_idx, baseline_hash=None, iterations=2):
    flags = get_soong_flags(config_idx)
    times = []
    mems = []
    
    for i in range(iterations):
        t, m, h = run_build_and_profile(module_name, flags)
        if not t:
            return None, None, None
            
        if baseline_hash and h != baseline_hash:
            print(f"    [!] SILENT MISCOMPILATION DETECTED in Config {config_idx} (Hash: {h[:8]} != Base: {baseline_hash[:8]}). Rejecting.")
            return None, None, None
            
        times.append(t)
        mems.append(m)
        
    return np.median(times), np.median(mems), h

def solve_bayesian_weights(y_times, y_mems):
    X = X_DESIGN
    l2_reg = 0.1
    
    XT_X = np.dot(X.T, X)
    I = np.eye(XT_X.shape[0])
    I[0, 0] = 0
    
    beta_time = np.linalg.solve(XT_X + l2_reg * I, np.dot(X.T, y_times))
    beta_mem = np.linalg.solve(XT_X + l2_reg * I, np.dot(X.T, y_mems))
    
    return beta_time, beta_mem

def main():
    if len(sys.argv) < 2:
        print("Usage: python3 bayesian_tuner.py <module_name>")
        sys.exit(1)
        
    module_name = sys.argv[1]
    print(f"==============================================================")
    print(f"Starting 16-Flag Bayesian Overlap Tuning for {module_name}...")
    print(f"==============================================================")
    
    # 1. Establish Baseline (Config 0)
    print("\n[Phase 1] Evaluating baseline (default flags)...")
    base_t, base_m, baseline_hash = evaluate_config(module_name, 0, baseline_hash=None, iterations=3)
    if not base_t:
        print("Baseline build failed! Aborting.")
        sys.exit(1)
    print(f" -> Baseline Median Time: {base_t:.2f}s, Median RAM: {base_m/1024:.2f}MB, Hash: {baseline_hash[:8]}")
    
    y_times = [base_t]
    y_mems = [base_m]
    
    # 2. Run 16 Overlapping Configurations
    print("\n[Phase 2] Running 16 Overlapping Screening Configurations...")
    for idx in range(1, 17):
        flags = get_soong_flags(idx)
        print(f" -> Config {idx}/16: {flags} ...")
        med_t, med_m, h = evaluate_config(module_name, idx, baseline_hash=baseline_hash, iterations=2)
        if med_t is None:
            print(f"      Config {idx} failed correctness or execution. Setting high default penalty.")
            y_times.append(base_t * 1.5)
            y_mems.append(base_m * 1.5)
        else:
            print(f"      Time: {med_t:.2f}s (Diff: {med_t - base_t:+.2f}s), RAM: {med_m/1024:.2f}MB")
            y_times.append(med_t)
            y_mems.append(med_m)
            
    # 3. Solve Normal Equations to derive Bayesian Weights
    print("\n[Phase 3] Solving Normal Equations (Ridge Regression)...")
    beta_time, beta_mem = solve_bayesian_weights(np.array(y_times), np.array(y_mems))
    
    intercept_time = beta_time[0]
    weights_time = beta_time[1:]
    weights_mem = beta_mem[1:]
    
    print("\n==============================================================")
    print("Individual Flag Impact Analysis (Calculated Weights):")
    print("==============================================================")
    print(f"{'Flag / Feature':<45} | {'Time Weight (s)':<15} | {'RAM Weight (MB)':<15}")
    print("-" * 83)
    
    impactful_flags = []
    
    for k in range(len(FEATURES)):
        w_t = weights_time[k]
        w_m = weights_mem[k] / 1024 / 1024
        print(f"{FEATURES[k]:<45} | {w_t:+15.3f}s | {w_m:+15.1f}MB")
        
        if w_t < -0.3:
            impactful_flags.append((FEATURES[k], w_t))
            
    print("==============================================================")
    
    # 4. Propose Winning Combination (Phase 4)
    print("\n[Phase 4] Proposing Crown Optimization Candidate...")
    if not impactful_flags:
        print("No individual flags showed meaningful performance improvements over the baseline.")
        sys.exit(0)
        
    impactful_flags.sort(key=lambda x: x[1])
    
    # Filter exclusive groups (mutually exclusive JVM configurations)
    winning_flags = []
    
    # 1. Heap Size Group (exclusive)
    heap_flags = [x for x in impactful_flags if x[0] in ["-Xmx2G", "-Xmx8G", "-Xmx12G"]]
    if heap_flags:
        best_heap = min(heap_flags, key=lambda x: x[1])
        winning_flags.append(best_heap[0])
        
    # 2. GC Group (exclusive)
    gc_flags = [x for x in impactful_flags if x[0] in ["-XX:+UseZGC", "-XX:+UseParallelGC"]]
    if gc_flags:
        best_gc = min(gc_flags, key=lambda x: x[1])
        winning_flags.append(best_gc[0])
        
    # 3. Compiler Count Group (exclusive)
    cc_flags = [x for x in impactful_flags if x[0] in ["-XX:CICompilerCount=2", "-XX:CICompilerCount=4", "-XX:CICompilerCount=8"]]
    if cc_flags:
        best_cc = min(cc_flags, key=lambda x: x[1])
        winning_flags.append(best_cc[0])
        
    # 4. Non-exclusive flags
    for flag, weight in impactful_flags:
        if flag not in [
            "-Xmx2G", "-Xmx8G", "-Xmx12G", 
            "-XX:+UseZGC", "-XX:+UseParallelGC", 
            "-XX:CICompilerCount=2", "-XX:CICompilerCount=4", "-XX:CICompilerCount=8"
        ]:
            winning_flags.append(flag)
            
    print(f"The Bayesian model identified the following performance-promoting flags:")
    for flag, weight in impactful_flags:
        print(f"  * {flag} (Marginal Impact: {weight:.3f}s)")
        
    print(f"\nProposed Optimal Multi-Flag Combination:")
    print(f"  {winning_flags}")
    
    # 5. Final Validation Phase
    print("\n[Phase 5] Executing Final Validation Run...")
    t, m, h = run_build_and_profile(module_name, winning_flags)
    if t is None or h != baseline_hash:
        print("Validation failed! The proposed combined flags introduced errors or crashed.")
    else:
        speedup = base_t - t
        speedup_pct = (speedup / base_t) * 100
        print(f"\n==============================================================")
        print(f"CROWN CHAMPION VALIDATED SUCCESSFULLY!")
        print(f"==============================================================")
        print(f"Optimal Config: {winning_flags}")
        print(f"Baseline Time:  {base_t:.2f}s")
        print(f"Optimized Time: {t:.2f}s (Saved: {speedup:.2f}s, {speedup_pct:.1f}% speedup)")
        print(f"Optimized RAM:  {m/1024:.2f}MB (Base: {base_m/1024:.2f}MB)")
        print(f"==============================================================")
        
        tuning_map = {module_name: winning_flags}
        with open("jvm_tuning_map.json", "w") as f:
            json.dump(tuning_map, f)

if __name__ == "__main__":
    main()
