// Copyright (C) 2022 The Android Open Source Project
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

package apex

// This file contains branch specific constants. They are stored in a separate
// file to minimise the potential of merge conflicts between branches when
// the code from the package is changed.

// The default manifest version for all the modules on this branch. A module
// can override this version by setting the version field in its manifest.json.
// The value follows the schema from go/mainline-version-codes, and is chosen
// based on the branch such that the builds from testing and development
// branches will have a version higher than the prebuilts.
const defaultManifestVersion = "339990000"
