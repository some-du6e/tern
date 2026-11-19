package diff

import (
	"regexp"
	"strings"
)

// T3's file kinds by path (packages/shared/src/changedFileKind.ts), the
// part the Diff panel's "Hide test files" reads.

var (
	generatedFiles = set("bun.lock", "bun.lockb", "cargo.lock", "composer.lock", "deno.lock", "flake.lock",
		"gemfile.lock", "go.sum", "gradle.lockfile", "mix.lock", "npm-shrinkwrap.json", "package-lock.json",
		"packages.lock.json", "pipfile.lock", "pnpm-lock.yaml", "podfile.lock", "poetry.lock", "pubspec.lock",
		"uv.lock", "yarn.lock")
	generatedName = regexp.MustCompile(`(\.generated\.|\.gen\.|\.g\.dart$|\.pb\.go$|_pb2(_grpc)?\.py$|\.pb\.(cc|h)$|\.min\.(js|css)$|\.(js|css)\.map$)`)
	generatedDirs = set("__generated__", "_generated", "generated")

	mockName = regexp.MustCompile(`(\.snap$|\.(mocks?|fixtures?|stubs?|fakes?)\.[^.]+$|^(mocks?|fixtures?|stubs?|fakes?)([-_.A-Z]|$))`)
	mockDirs = set("__fixtures__", "__mocks__", "__snapshots__", "fakes", "fixture", "fixtures", "mock", "mocks",
		"stubs", "testdata")

	testName = regexp.MustCompile(`(?i)(\.(test|spec|e2e|eval)\.[^.]+$|_(test|spec)\.[^.]+$|^test_.+\.py$|^conftest\.py$|^(vitest|jest)\.setup\.[^.]+$)`)
	// Case-sensitive: FooTest.java is a test class, Latest.java is not.
	classTestName = regexp.MustCompile(`(Tests?|Spec)\.(java|kt|kts|scala|groovy|cs|php|swift|m)$`)
	testDirs      = set("__tests__", "__test__", "cypress", "e2e", "evals", "spec", "specs", "test", "tests")
)

func set(names ...string) map[string]bool {
	m := map[string]bool{}
	for _, n := range names {
		m[n] = true
	}
	return m
}

// IsTest reports whether path is a test or test support (a mock, fixture
// or snapshot). Generated files never are, whatever their folder.
func IsTest(path string) bool {
	segs := strings.Split(strings.ReplaceAll(path, `\`, "/"), "/")
	name := segs[len(segs)-1]
	in := func(dirs map[string]bool) bool {
		for _, d := range segs[:len(segs)-1] {
			if dirs[strings.ToLower(d)] {
				return true
			}
		}
		return false
	}
	if generatedFiles[strings.ToLower(name)] || generatedName.MatchString(name) || in(generatedDirs) {
		return false
	}
	return mockName.MatchString(name) || in(mockDirs) ||
		testName.MatchString(name) || classTestName.MatchString(name) || in(testDirs)
}
