// Package buildinfo resolves metadata embedded by the Go toolchain.
package buildinfo

import "runtime/debug"

const developmentVersion = "dev"

// Version prefers a linker-injected version, then the module version recorded
// by go install, and finally the development placeholder.
func Version(injected string) string {
	info, ok := debug.ReadBuildInfo()
	return resolveVersion(injected, info, ok)
}

func resolveVersion(injected string, info *debug.BuildInfo, ok bool) string {
	if injected != "" && injected != developmentVersion {
		return injected
	}
	if ok && info != nil && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return developmentVersion
}
