package buildinfo

import (
	"runtime/debug"
	"testing"
)

func TestVersionPrefersInjectedVersion(t *testing.T) {
	t.Parallel()

	if got, want := Version("v1.2.3-custom"), "v1.2.3-custom"; got != want {
		t.Errorf("Version(linker value) = %q, want %q", got, want)
	}
}

func TestVersionFallsBackForDevelopmentBuild(t *testing.T) {
	t.Parallel()

	if got, want := Version("dev"), "dev"; got != want {
		t.Errorf("Version(development build) = %q, want %q", got, want)
	}
}

func TestResolveVersionUsesInstalledModuleVersion(t *testing.T) {
	t.Parallel()

	info := &debug.BuildInfo{Main: debug.Module{Version: "v1.2.3"}}
	if got, want := resolveVersion("dev", info, true), "v1.2.3"; got != want {
		t.Errorf("resolveVersion(module build) = %q, want %q", got, want)
	}
}
