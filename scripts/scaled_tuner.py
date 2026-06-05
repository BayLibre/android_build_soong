import re
import os
import sys
import time
import json
import argparse
import numpy as np
import glob

sys.path.append(os.path.dirname(__file__))
import bayesian_tuner

NINJA_PATH = "out_hyper/soong/build.aosp_cf_x86_64_only_phone.ninja"


def parse_intermediates_path(path):
    if "/.intermediates/" not in path:
        return None, None, None

    parts = path.split("/.intermediates/")
    rest = parts[1]
    segments = rest.split("/")

    # Find the variant segment
    variant_idx = -1
    for idx, seg in enumerate(segments):
        if seg.startswith("android_") or seg.startswith(
                "linux_") or seg.startswith(
                    "host_") or seg == "common" or seg.startswith(
                        "windows_") or seg.startswith("darwin_"):
            variant_idx = idx
            break

    if variant_idx == -1:
        for idx, seg in enumerate(segments):
            if seg in [
                    "common", "linux_x86_64", "linux_glibc_common",
                    "linux_glibc_x86_64", "windows_x86_64"
            ]:
                variant_idx = idx
                break

    if variant_idx <= 0:
        return None, None, None

    module_name = segments[variant_idx - 1]
    module_path = "/".join(segments[:variant_idx - 1])
    variant = segments[variant_idx]

    return module_path, module_name, variant


def read_ninja_lines(file_path):
    with open(file_path, "r", errors="ignore") as f:
        pending_line = ""
        for line in f:
            line_stripped = line.rstrip("\r\n")
            if line_stripped.endswith("$"):
                pending_line += line_stripped[:-1]
            else:
                yield pending_line + line_stripped
                pending_line = ""


def scan_single_file(path, modules):
    count = 0
    for line in read_ninja_lines(path):
        if not line.startswith("build "):
            continue

        count += 1
        parts = line.split(":", 1)
        if len(parts) < 2:
            continue

        outputs_part = parts[0][6:].strip()
        rule_inputs_part = parts[1].strip()

        out_files = outputs_part.split()
        if not out_files:
            continue

        module_path, module_name, variant = parse_intermediates_path(
            out_files[0])
        if not module_name:
            continue

        key = (module_name, variant)
        if key not in modules:
            modules[key] = {
                "name": module_name,
                "path": module_path,
                "variant": variant,
                "java_srcs": set(),
                "kt_srcs": set(),
                "shards": set(),
                "rules": set(),
                "apt": False,
                "res": False
            }

        mod = modules[key]

        rule_parts = rule_inputs_part.split(None, 1)
        rule = rule_parts[0]
        mod["rules"].add(rule)

        if "apt" in rule or "kapt" in rule:
            mod["apt"] = True
        if "aapt" in rule or "r_jar" in rule or "busybox" in outputs_part:
            mod["res"] = True

        if len(rule_parts) > 1:
            inputs = rule_parts[1]
            for item in inputs.split():
                item = item.rstrip("$")
                if item.endswith(".java"):
                    mod["java_srcs"].add(item)
                elif item.endswith(".kt"):
                    mod["kt_srcs"].add(item)

        if "javac-" in outputs_part:
            shard_match = re.search(r'javac-(\d+)', outputs_part)
            if shard_match:
                mod["shards"].add(shard_match.group(1))

    return count


def extract_all_modules():
    ninja_files = []
    main_path = NINJA_PATH
    if os.path.exists(main_path):
        ninja_files.append(main_path)

    inc_path = "out_hyper/soong/build.aosp_cf_x86_64_only_phone.incremental.ninja"
    if os.path.exists(inc_path) and os.path.getsize(inc_path) > 0:
        ninja_files.append(inc_path)

    for i in range(15):
        path = f"out_hyper/soong/build.aosp_cf_x86_64_only_phone.{i}.ninja"
        if os.path.exists(path) and os.path.getsize(path) > 0:
            ninja_files.append(path)

    print(f"[Phase 1] Extracting build graph from Soong Ninja shards...")
    start_t = time.time()
    modules = {}
    total_statements = 0

    for path in ninja_files:
        file_statements = scan_single_file(path, modules)
        total_statements += file_statements

    print(
        f"  * Total: Scanned {total_statements} build statements in {time.time() - start_t:.2f}s."
    )
    return modules


def filter_jvm_modules(modules):
    print(f"\n[Phase 2] Filtering for active JVM compilation modules...")
    jvm_modules = []

    for key, m in modules.items():
        # A module is JVM-bound if it compiles Java/Kotlin
        has_sources = len(m["java_srcs"]) > 0 or len(m["kt_srcs"]) > 0

        if has_sources:
            jvm_modules.append(m)

    print(
        f"  * Extracted {len(jvm_modules)} active JVM compilation targets (excluding prebuilts)."
    )
    return jvm_modules


def cluster_modules_kmeans(jvm_modules, k=4):
    print(
        f"\n[Phase 3] Clustering JVM modules using K-Means (Normalized Feature Space)..."
    )

    # 1. Construct feature matrix
    # Features:
    # 0: log(Java sources + 1)
    # 1: log(Kotlin sources + 1)
    # 2: Kotlin ratio (kt / (kt + java + 1))
    # 3: Shards count
    # 4: 1 if APT else 0
    # 5: 1 if Res else 0
    X = []
    for m in jvm_modules:
        java_c = len(m["java_srcs"])
        kt_c = len(m["kt_srcs"])
        kt_ratio = kt_c / (kt_c + java_c + 1.0)
        shards = len(m["shards"])
        apt = 1.0 if m["apt"] else 0.0
        res = 1.0 if m["res"] else 0.0

        X.append([
            np.log1p(java_c),
            np.log1p(kt_c), kt_ratio,
            float(shards), apt, res
        ])

    X = np.array(X)

    # Standardize features (zero mean, unit variance)
    X_mean = np.mean(X, axis=0)
    X_std = np.std(X, axis=0)
    X_std[X_std == 0] = 1.0  # Prevent divide by zero
    X_norm = (X - X_mean) / X_std

    # Run custom K-Means (to keep dependencies to pure numpy)
    np.random.seed(42)
    centroids = X_norm[np.random.choice(X_norm.shape[0], k, replace=False)]

    for iteration in range(20):  # 20 iterations is enough for convergence
        # Calculate distances from points to centroids
        distances = np.linalg.norm(X_norm[:, np.newaxis] - centroids, axis=2)
        labels = np.argmin(distances, axis=1)

        # Update centroids
        new_centroids = np.array([
            X_norm[labels == i].mean(axis=0) if np.sum(
                labels == i) > 0 else centroids[i] for i in range(k)
        ])
        if np.allclose(centroids, new_centroids):
            break
        centroids = new_centroids

    print(f"  * K-Means converged successfully in {iteration+1} iterations.")

    # Map clusters back to modules
    clusters = {i: [] for i in range(k)}
    for idx, m in enumerate(jvm_modules):
        m["cluster_id"] = labels[idx]
        m["feature_vector"] = X_norm[idx]
        clusters[labels[idx]].append(m)

    return clusters, centroids


def get_dynamic_archetype_name(centroid, cluster_id):
    java_f, kt_f, ratio_f, shards_f, apt_f, res_f = centroid
    
    labels = []
    if kt_f > 0.5:
        labels.append("Kotlin")
    if ratio_f > 0.5:
        labels.append("Compose/UI")
    if shards_f > 0.8:
        labels.append("Highly Sharded")
    if apt_f > 0.8:
        labels.append("APT-Active")
    if res_f > 0.8:
        labels.append("Resource-Heavy")
    if java_f > 0.8 and shards_f <= 0.8:
        labels.append("Heavy Javac")
        
    if not labels:
        if java_f < -0.5 and kt_f < -0.5:
            return f"Small Standard (Archetype {cluster_id})"
        return f"Standard Library (Archetype {cluster_id})"
        
    return f"{' / '.join(labels)} (Archetype {cluster_id})"


def load_buildable_modules():
    out_dir = os.environ.get("OUT_DIR", "out")
    paths = glob.glob(f"{out_dir}/target/product/*/module-info.json")
    if not paths:
        print(f"  * Warning: Could not find module-info.json in {out_dir}/target/product/*/. Cannot filter for buildable representatives.")
        return None
    
    path = paths[0]
    print(f"  * Loading buildable modules list from {path}...")
    try:
        with open(path, "r") as f:
            data = json.load(f)
            return set(data.keys())
    except Exception as e:
        print(f"  * Error loading {path}: {e}")
        return None


def sample_representatives(clusters, centroids, reps_per_cluster=2, buildable_modules=None):
    print(f"\n[Phase 4] Sampling Cluster Representative Centroids...")
    representatives = {}

    for cluster_id, modules in clusters.items():
        if not modules:
            continue

        centroid = centroids[cluster_id]

        # Calculate Euclidean distance of all modules in cluster to the cluster centroid
        distances = []
        for m in modules:
            dist = np.linalg.norm(m["feature_vector"] - centroid)
            distances.append((m, dist))

        # Sort by distance to find the absolute closest (most representative)
        distances.sort(key=lambda x: x[1])

        # Pick top distinct representatives
        reps = []
        for m, dist in distances:
            if buildable_modules and m["name"] not in buildable_modules:
                continue
            # Ensure we don't pick duplicates or very minor modules if possible
            if len(m["java_srcs"]) > 5 or len(m["kt_srcs"]) > 5:
                reps.append(m)
            if len(reps) >= reps_per_cluster:
                break

        # Fallback if no heavy modules in cluster
        if len(reps) < reps_per_cluster:
            filtered_distances = [x for x in distances if not buildable_modules or x[0]["name"] in buildable_modules]
            reps = [x[0] for x in filtered_distances[:reps_per_cluster]]

        representatives[cluster_id] = reps

    # Print proposed sweep plan
    print("\n==============================================================")
    # Print cluster summary statistics
    print(
        f"{'Cluster ID':<12} | {'Modules':<8} | {'Archetype Description / Typical Representative':<45}"
    )
    print("-" * 72)

    for c_id, mods in clusters.items():
        reps_names = [m["name"] for m in representatives.get(c_id, [])]
        desc = get_dynamic_archetype_name(centroids[c_id], c_id)
        print(f"Cluster {c_id:<3} | {len(mods):<8} | {desc:<45}")
        print(f"             |          | Representatives: {reps_names}")
        print("-" * 72)
    print("==============================================================")

    return representatives


def get_host_resources():
    try:
        pagesvalue = os.sysconf('SC_PAGE_SIZE') * os.sysconf('SC_PHYS_PAGES')
        total_ram_gb = pagesvalue / (1024.**3)
    except Exception:
        total_ram_gb = 32.0  # safe default
        
    try:
        total_cores = os.cpu_count() or 4
    except Exception:
        total_cores = 4
        
    return total_ram_gb, total_cores


def generalize_flags(clusters, representatives, tuned_reps):
    print(
        f"\n[Phase 4.5] Generalizing representative flags to clusters using 1-NN..."
    )
    global_tuning_map = {}

    for cluster_id, modules in clusters.items():
        reps = representatives.get(cluster_id, [])
        # Filter reps that were successfully tuned
        valid_reps = [
            r for r in reps
            if r["name"] in tuned_reps and tuned_reps[r["name"]] is not None
        ]

        if not valid_reps:
            print(
                f"  * Warning: No successfully tuned representatives for Cluster {cluster_id}. Skipping generalization."
            )
            continue

        print(
            f"  * Generalizing Cluster {cluster_id} using representatives: {[r['name'] for r in valid_reps]}"
        )

        for m in modules:
            m_name = m["name"]

            # If it's a representative itself, keep its own flags
            if m_name in tuned_reps:
                global_tuning_map[m_name] = tuned_reps[m_name]
                continue

            # Nearest Neighbor logic
            if len(valid_reps) == 1:
                best_rep = valid_reps[0]
            else:
                # Calculate distance to both and pick nearest
                d1 = np.linalg.norm(
                    m["feature_vector"] - valid_reps[0]["feature_vector"]
                )
                d2 = np.linalg.norm(
                    m["feature_vector"] - valid_reps[1]["feature_vector"]
                )
                best_rep = valid_reps[0] if d1 < d2 else valid_reps[1]

            global_tuning_map[m_name] = tuned_reps[best_rep["name"]]

    return global_tuning_map


def apply_oom_safety_filter(modules, tuning_map_path):
    ram_gb, cores = get_host_resources()
    print(
        f"\n[Phase 5] Applying Hardware-Aware OOM-Safety Filtering ({ram_gb:.1f}GB RAM, {cores} Cores)..."
    )

    if not os.path.exists(tuning_map_path):
        print(
            f"  * Warning: {tuning_map_path} does not exist. Skipping OOM safety filter."
        )
        return

    # If RAM > 128GB, we can be very generous (e.g. high-end workstation)
    is_high_end = ram_gb > 128.0
    medium_cap = "6G" if is_high_end else "4G"
    medium_cap_val = 6 if is_high_end else 4

    print(
        f"  * Host memory tier: {'HIGH-END' if is_high_end else 'STANDARD'}. Enforcing moderately heavy cap: {medium_cap}"
    )

    with open(tuning_map_path, "r") as f:
        tuning_map = json.load(f)

    modified_count = 0
    total_modules = 0
    heavy_count = 0

    oom_safe_map = {}

    # Pre-compute source counts for O(1) lookup during filtering
    src_counts = {}
    for (m_name, variant), m in modules.items():
        count = len(m["java_srcs"]) + len(m["kt_srcs"])
        src_counts[m_name] = max(src_counts.get(m_name, 0), count)

    for name, flags in tuning_map.items():
        total_modules += 1
        src_count = src_counts.get(name, 0)
        
        # Strip -Xss from all configs to prevent StackOverflowError
        flags = [f for f in flags if not f.startswith("-Xss")]

        # Thresholds for memory conservation:
        # - Ultra heavy modules (>800 sources) can retain large heaps (8G, 12G)
        # - Moderately heavy modules (300-800 sources) are capped at medium_cap
        # - Small modules (<=300 sources) are stripped of -Xmx to fallback to baseline Soong defaults
        is_ultra_heavy = src_count > 800
        is_moderately_heavy = 300 < src_count <= 800

        if is_ultra_heavy:
            heavy_count += 1
            # Heavy floor for ultra-heavy modules to prevent OOM under generalization
            heavy_floor = "8G" if is_high_end else "6G"
            heavy_floor_val = 8 if is_high_end else 6
            
            has_xmx = False
            upgraded_flags = []
            for f in flags:
                if f.startswith("-Xmx"):
                    has_xmx = True
                    match = re.match(r"-Xmx(\d+)([gGmM])", f)
                    if match:
                        val = int(match.group(1))
                        unit = match.group(2).lower()
                        # If it has Xmx but it is smaller than heavy_floor, upgrade it
                        if (unit == 'g' and val < heavy_floor_val) or (
                            unit == 'm' and val < (heavy_floor_val * 1024)
                        ):
                            upgraded_flags.append(f"-Xmx{heavy_floor}")
                            modified_count += 1
                        else:
                            upgraded_flags.append(f)
                    else:
                        upgraded_flags.append(f"-Xmx{heavy_floor}")
                        modified_count += 1
                else:
                    upgraded_flags.append(f)
            
            if not has_xmx:
                upgraded_flags.append(f"-Xmx{heavy_floor}")
                modified_count += 1
                
            oom_safe_map[name] = upgraded_flags
        elif is_moderately_heavy:
            capped_flags = []
            modified = False
            for f in flags:
                if f.startswith("-Xmx"):
                    match = re.match(r"-Xmx(\d+)([gGmM])", f)
                    if match:
                        val = int(match.group(1))
                        unit = match.group(2).lower()
                        if (unit == 'g' and val > medium_cap_val) or (
                            unit == 'm' and val > (medium_cap_val * 1024)
                        ):
                            capped_flags.append(f"-Xmx{medium_cap}")
                            modified = True
                        else:
                            capped_flags.append(f)
                    else:
                        capped_flags.append(f"-Xmx{medium_cap}")
                        modified = True
                else:
                    capped_flags.append(f)
            if modified:
                modified_count += 1
            oom_safe_map[name] = capped_flags
        else:
            cleaned_flags = [f for f in flags if not f.startswith("-Xmx")]
            if len(cleaned_flags) != len(flags):
                modified_count += 1
            oom_safe_map[name] = cleaned_flags

    with open(tuning_map_path, "w") as f:
        json.dump(oom_safe_map, f, indent=2)

    print(
        f"  * Completed OOM-Safety filtering on {tuning_map_path} successfully!"
    )
    print(f"    - Total modules processed: {total_modules}")
    print(f"    - Heavy modules kept with Xmx: {heavy_count}")
    print(
        f"    - Small/medium modules capped or stripped of Xmx: {modified_count}"
    )


def ensure_valid_tuning_map(path):
    if os.path.exists(path):
        try:
            with open(path, "r") as f:
                json.load(f)
        except Exception:
            print(f"Warning: {path} is invalid JSON. Initializing with empty object.")
            try:
                with open(path, "w") as f:
                    json.dump({}, f)
            except Exception as e:
                print(f"Error initializing {path}: {e}")


def main():
    parser = argparse.ArgumentParser(description="Soong JVM Optimization & Clustering Tuner")
    parser.add_argument("--k", type=int, default=8, help="Number of cluster archetypes to construct (default: 8)")
    parser.add_argument("--module", type=str, help="Single package tuning mode: only tune this module")
    parser.add_argument("--extreme-effort", action="store_true", help="Extreme effort mode: tune every package individually, bypassing clustering")
    parser.add_argument("--only-oom-filter", action="store_true", help="Only apply the OOM safety filter to the existing map, bypassing tuning")
    parser.add_argument("--bottlenecks", action="store_true", help="Tune only the top critical-path bottleneck modules")
    parser.add_argument("--reps-per-cluster", type=int, default=2, help="Number of representative modules to tune per cluster (default: 2)")
    parser.add_argument("--iterations", type=int, default=1, help="Number of iterations to run each configuration to calculate median time/RAM (default: 1)")
    args = parser.parse_args()

    start_t = time.time()
    tuning_map_path = "build/soong/java/jvm_tuning_map.json"
    ensure_valid_tuning_map(tuning_map_path)

    buildable_modules = None
    if not args.only_oom_filter:
        buildable_modules = load_buildable_modules()

    if args.module:
        print(f"\n[Mode] Single Module Tuning: {args.module}")
        # We still extract modules to have features for OOM safety filter
        modules = extract_all_modules()
        
        # Run tuner on this module
        winning_flags, miscomps = bayesian_tuner.tune_module(args.module, iterations=args.iterations)
        
        if winning_flags is not None:
            # Load existing map or create new
            tuning_map = {}
            if os.path.exists(tuning_map_path):
                try:
                    with open(tuning_map_path, "r") as f:
                        tuning_map = json.load(f)
                except Exception:
                    pass
            
            tuning_map[args.module] = winning_flags
            
            with open(tuning_map_path, "w") as f:
                json.dump(tuning_map, f, indent=2)
                
            # Apply OOM safety filter to the updated map
            apply_oom_safety_filter(modules, tuning_map_path)
        else:
            print(f"Tuning failed for module {args.module}")
            sys.exit(1)

    elif args.bottlenecks:
        print(f"\n[Mode] Targeted Critical-Path Bottlenecks Tuning")
        bottlenecks = [
            "platform-bootclasspath",
            "framework-minus-apex",
            "framework-minus-apex-intdefs",
            "Settings-core",
            "SystemUI-tests",
            "framework-doc-system-stubs",
            "android-non-updatable-doc-stubs",
            "framework-doc-stubs",
            "system-api-stubs-docs-non-updatable",
            "services.core.unboosted",
            "SystemUI-core"
        ]
        
        # 1. Feature Extraction
        modules = extract_all_modules()
        
        # Start fresh to avoid stale noise
        tuning_map = {}
        
        total_tuned = 0
        for idx, m_name in enumerate(bottlenecks):
            print(f"\n[{idx+1}/{len(bottlenecks)}] Tuning Bottleneck: {m_name}...")
            winning_flags, miscomps = bayesian_tuner.tune_module(m_name, iterations=args.iterations)
            
            if winning_flags is not None:
                tuning_map[m_name] = winning_flags
                total_tuned += 1
                # Save progress incrementally
                with open(tuning_map_path, "w") as f:
                    json.dump(tuning_map, f, indent=2)
            else:
                print(f"Tuning failed or baseline failed for {m_name}. Skipping.")
                
        # Initialize all other active JVM modules to [] in the map
        jvm_modules = filter_jvm_modules(modules)
        for m in jvm_modules:
            name = m["name"]
            if name not in tuning_map:
                tuning_map[name] = []
                
        with open(tuning_map_path, "w") as f:
            json.dump(tuning_map, f, indent=2)
            
        # Apply OOM safety filter
        apply_oom_safety_filter(modules, tuning_map_path)

    elif args.only_oom_filter:
        print(f"\n[Mode] Only OOM-Safety Filtering")
        modules = extract_all_modules()
        apply_oom_safety_filter(modules, tuning_map_path)

    elif args.extreme_effort:
        print(f"\n[Mode] Extreme Effort Tuning: Tuning all packages individually")
        # 1. Feature Extraction
        modules = extract_all_modules()

        # 2. Filter JVM modules
        jvm_modules = filter_jvm_modules(modules)
        if buildable_modules:
            jvm_modules = [m for m in jvm_modules if m["name"] in buildable_modules]
        
        print(f"Found {len(jvm_modules)} active (and buildable) JVM modules to tune.")
        
        # Load existing map to accumulate
        tuning_map = {}
        if os.path.exists(tuning_map_path):
            try:
                with open(tuning_map_path, "r") as f:
                    tuning_map = json.load(f)
            except Exception:
                pass

        total_tuned = 0
        for idx, m in enumerate(jvm_modules):
            m_name = m["name"]
            print(f"\n[{idx+1}/{len(jvm_modules)}] Tuning {m_name}...")
            
            winning_flags, miscomps = bayesian_tuner.tune_module(m_name, iterations=args.iterations)
            
            if winning_flags is not None:
                tuning_map[m_name] = winning_flags
                total_tuned += 1
                # Save progress incrementally
                with open(tuning_map_path, "w") as f:
                    json.dump(tuning_map, f, indent=2)
            else:
                print(f"Tuning failed or baseline failed for {m_name}. Skipping.")

        print(f"\nTuning phase completed. Successfully tuned {total_tuned}/{len(jvm_modules)} modules.")
        
        # Apply OOM safety filter
        apply_oom_safety_filter(modules, tuning_map_path)

    else:
        # Standard Cluster-and-Sample Mode (FULLY AUTOMATED!)
        print(f"\n[Mode] Automated Cluster-and-Sample Tuning Pipeline")

        # 1. Feature Extraction
        modules = extract_all_modules()

        # 2. Filter JVM modules
        jvm_modules = filter_jvm_modules(modules)

        # 3. Cluster modules
        clusters, centroids = cluster_modules_kmeans(jvm_modules, k=args.k)

        # 4. Sample representatives
        representatives = sample_representatives(clusters, centroids, reps_per_cluster=args.reps_per_cluster, buildable_modules=buildable_modules)

        # 5. Tune representatives sequentially
        print(f"\n[Phase 4] Tuning Representative Modules sequentially...")
        tuned_reps = {}

        # Flatten reps to compile a clean list to tune
        all_reps_to_tune = []
        for cluster_id, reps in representatives.items():
            for r in reps:
                all_reps_to_tune.append((cluster_id, r["name"]))

        print(
            f"Found {len(all_reps_to_tune)} representative modules to tune: {[r[1] for r in all_reps_to_tune]}"
        )

        for idx, (c_id, rep_name) in enumerate(all_reps_to_tune):
            print(
                f"\n[{idx+1}/{len(all_reps_to_tune)}] Tuning Cluster {c_id} Representative: {rep_name}..."
            )

            winning_flags, miscomps = bayesian_tuner.tune_module(rep_name, iterations=args.iterations)

            # Even if winning_flags is empty [], we store it (means default flags are best)
            # Only if it failed completely (None), we skip it.
            if winning_flags is not None:
                tuned_reps[rep_name] = winning_flags
            else:
                print(
                    f"  * Warning: Tuning failed completely for representative {rep_name}."
                )

        # 6. Generalize Representative flags to all cluster modules using 1-NN
        global_tuning_map = generalize_flags(
            clusters, representatives, tuned_reps
        )

        # Write the raw generalized map
        with open(tuning_map_path, "w") as f:
            json.dump(global_tuning_map, f, indent=2)

        # 7. Apply Hardware-Aware OOM-safety filter to the generalized map
        apply_oom_safety_filter(modules, tuning_map_path)

    print(f"\nCOMPLETED SUCCESSFULLY in {time.time() - start_t:.2f}s.")


if __name__ == "__main__":
    main()
