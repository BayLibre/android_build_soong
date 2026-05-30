# Soong JVM Hyperparameter Tuning & Optimization Utility

This directory contains the automated tooling for the AOSP Soong JVM hyperparameter tuning framework. It enables developers and CI systems to automatically discover, optimize, and scale JVM compilation parameters (such as heap size, garbage collection algorithms, and JIT compiler threading) across the thousands of compilation modules in the Android build graph.

---

## Background & Package Drift

As the Android codebase evolves, new modules are added, existing modules grow in size, and Java/Kotlin source files drift. To maintain optimal compilation performance while guaranteeing absolute memory safety (preventing out-of-memory errors on constrained build nodes), the JVM tuning map should be updated periodically.

Updating the profile is as simple as **rerunning the scaling pipeline**. The pipeline automatically extracts the latest build graph from your local Soong compilation state, runs clustering to classify targets, and updates the production tuning profile with tiered OOM-safety controls.

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

### Step 2: Optional - Optimize a Representative Target
If you want to run the 16-flag Bayesian overlap tuning to discover optimal flags for a specific class of targets, run:
```bash
# From the AOSP root directory
python3 build/soong/scripts/bayesian_tuner.py <module_name>
```
*This performs a 17-configuration screening using Ridge Regression to identify the exact JIT, GC, and heap settings that maximize build speed and minimize memory.*

### Step 3: Rerun the Graph Extraction & Scaling Pipeline
To automatically propagate these settings across the entire build graph, map them to the cluster archetypes, and apply production safety guards:
```bash
# From the AOSP root directory
python3 build/soong/scripts/scaled_tuner.py
```
This command executes:
1. **Graph Extraction**: Parses all Soong-generated Ninja files (`out_hyper/soong/*.ninja`) to extract active Java, Kotlin, and Metalava compilation modules along with their source counts.
2. **K-Means Clustering**: Clusters the thousands of modules into 4 key archetypes:
   - **Cluster 0**: Standard Javac Libraries
   - **Cluster 1**: Massively Concurrent Sharded Javac
   - **Cluster 2**: Heavy Single JVMs (Metalava/Droidstubs)
   - **Cluster 3**: Kotlin / Compose UI Libraries
3. **OOM-Safety & Tiered Memory Capping**: Applies memory conservation filters and writes the final optimized, production-ready profile to `build/soong/java/jvm_tuning_map.json`.

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

*   `scaled_tuner.py`: The main entry point for build graph scanning, clustering, mapping, and safety filtering.
*   `bayesian_tuner.py`: Screening utility that profiles configurations locally and uses linear models to solve individual flag weights.
*   `../java/jvm_tuning_map.json`: The production profile loaded hermetically by the Soong `hyperoptimizer` mutator at build-time.
