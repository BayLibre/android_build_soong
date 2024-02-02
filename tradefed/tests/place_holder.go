package tests

// In order for "m blueprint_tests" to find test_module_config_test, it has to be
// in a bootstrap_go_package rule (not a blueprint_go_binary.
// blueprint_tests will then run "go test tradefed/tests" directly.
// Our test depends on tradefed (where the rule is defined) and java (where java rules are defined).
// There is no way to declare our test in build/soong/tradefed/Android.bp#song-tradefed without
// creating a circular dependency.
// So this "tests" subdirectory exits to create a package that depends on both tradefed and java.
// We need this place_holder.go file because if we have a bootstrap_go_package with testSrcs but no
// actual srcs, the build fails.  We just need to declare our package and then both "m" works and "go test" works.
// Alternatively, we could find a way to augment the logic that "m blueprint_tests" uses to find tests
// or explicitly add `tradefed/tests` to the dependency list.
// Another alternative would be to move out test into java to break the dependency.
// Finally, we could add a bootstrap_go_package version for tests or not complain if there are no srcs.
