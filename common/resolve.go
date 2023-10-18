package common

// MapProvidesToIdx iterates through every source package (`a`) and maps the
// name of all of
// its binary packages (e.g. `a`, `a-devel`) and providers
// (e.g. `pkgconfig(a)`) to
// the source.
func MapProvidesToIdx(srcPkgs []Package, nameToSrcIdx map[string]int) {
	for idx, srcPkg := range srcPkgs {
		nameToSrcIdx[srcPkg.Name] = idx
		for _, name := range srcPkg.Provides {
			nameToSrcIdx[name] = idx
		}
	}
}
