# Soong JVM Hyperparameter Tuning & Optimization Utility

This directory contains the automated tooling for the AOSP Soong JVM hyperparameter tuning framework. It enables developers and CI systems to automatically discover, optimize, and scale JVM compilation parameters (such as heap size, garbage collection algorithms, and JIT compiler threading) across the thousands of compilation modules in the Android build graph.

---

## Background & Package Drift

As the Android codebase evolves, new modules are added, existing modules grow in size, and Java/Kotlin source files drift. To maintain optimal compilation performance while guaranteeing absolute memory safety (preventing out-of-memory errors on constrained build nodes), the JVM tuning map should be updated periodically.

Updating the profile is completely automated. The **End-to-End Automated Pipeline** extracts the latest build graph from your local Soong compilation state, clusters the targets into workload archetypes, automatically selects representative modules, runs Bayesian overlap screening on those representatives, propagates the discovered optimal profiles across the entire graph, and commits the safety-capped profile directly to Soong.

---

## Workflow & Pipeline Steps

### Step 1: Build the Target Product
Before running the tuning utility, ensure you have executed a build of your target product (e.g., Cuttlefish) so that the Soong-generated Ninja files are populated.
```bash
# From the AOSP root directory
source build/envsetup.sh
lunch aosp_cf_x86_64_only_phone-userdebug
m nothing
```

### Step 2: Run the End-to-End Auto-Tuning Pipeline
To automatically discover and apply the optimized JVM configurations across all active build targets:
```bash
# From the AOSP root directory
python3 build/soong/scripts/auto_tune_pipeline.py [--k <num_clusters>]
```
*By default, `--k` is set to `8` to construct 8 distinct workload archetypes. You can specify any custom `K` to adjust the sweep granularity.*

This single command fully automates:
1. **Graph Extraction & Filtering**: Scans Soong-generated Ninja files (`out_hyper/soong/*.ninja`) to extract active JVM compilation modules and source counts.
2. **K-Means Archetype Clustering**: Clusters modules into `K` archetypes. Each archetype is dynamically labeled based on its feature space (e.g., Kotlin/Compose, Highly Sharded, Heavy Javac, etc.) to describe the workload profile.
3. **Representative Selection**: Identifies the top representative archetype module for each of the `K` clusters.
4. **Automated Bayesian Screening**: Dynamically sweeps each representative module through a 17-configuration fractional factorial matrix using Regularized Ridge Regression to discover the crown champion JVM flags.
5. **Tiered Memory Propagation & Capping**: Propagates winning configurations across all targets in each archetype while applying tiered safety caps, and writes the finalized, production-ready map to `build/soong/java/jvm_tuning_map.json`.

---

## Memory Conservation & Safety Controls

To guarantee build servers do not experience OOM (Out of Memory) crashes, the utility automatically applies a **Tiered Memory Filter** to all configurations before committing them:

| Target Profile | Criteria | Heap Flag Treatment (`-Xmx`) | Rationale |
| :--- | :--- | :--- | :--- |
| **Ultra-Heavy** | $>800$ source files OR contains `metalava`/`stubs` in name | Retains tuned heap sizes (up to **8G** or **12G**) | Truly massive compiles that genuinely require large memory pools to avoid garbage collection thrashing or OOMs. |
| **Moderately-Heavy** | Between $300$ and $800$ source files | Capped strictly at **`-Xmx4G`** | Provides a strong performance buffer while preventing excessive memory consumption on parallel build executors. |
| **Standard** | $\le 300$ source files | **Stripped of custom `-Xmx`** (falls back to baseline) | Conserves builder memory footprint across 90%+ of targets where a large heap is unnecessary. |

---

## File Structure

*   `auto_tune_pipeline.py`: The primary master runner that orchestrates the end-to-end build graph extraction, clustering, representative tuning, propagation, and capping.
*   `scaled_tuner.py`: Core graph extraction, K-Means clustering, and representative sampling components.
*   `bayesian_tuner.py`: Core profiling and Ridge Regression solver components.
*   `../java/jvm_tuning_map.json`: The production profile loaded hermetically by the Soong `hyperoptimizer` mutator at build-time.
