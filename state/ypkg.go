package state

import (
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"
	"slices"

	"github.com/GZGavinZhao/autobuild/config"
	"github.com/GZGavinZhao/autobuild/utils"
	"github.com/GZGavinZhao/autobuild/ypkg"
	"github.com/getsolus/libeopkg/index"
	"github.com/getsolus/libeopkg/pspec"
	"github.com/ulikunitz/xz"
)

func ypkgPackageExists(dir string) bool {
	return utils.PathExists(filepath.Join(dir, ypkg.YmlFile)) || utils.PathExists(filepath.Join(dir, ypkg.PspecFile))
}

func loadYpkgPackage(dir string, abConfig *config.AutobuildConfig) (pkgs []Package, err error) {
	// Check if the given directory contains a package definition
	pkgFile := filepath.Join(dir, "package.yml")
	pspecFile := filepath.Join(dir, "pspec_x86_64.xml")
	cfgFile := filepath.Join(dir, "autobuild.yml")

	ypkgYml, err := ypkg.Load(pkgFile)
	if err != nil {
		err = errors.New(fmt.Sprintf("Failed to load package.yml file for %s: %s", dir, err))
		return
	}

	pkgs = append(pkgs, Package{
		Path:      dir,
		Source:    ypkgYml.Name,
		Names:     []string{ypkgYml.Name},
		Version:   ypkgYml.Version,
		Release:   ypkgYml.Release,
		BuildDeps: ypkgYml.BuildDeps,
	})
	pkg := &pkgs[0]

	// Combine the rundeps of all subpackages into a single list
	// Note to self: this website can inspect yaml ast nodes:
	// https://astexplorer.net/, might be useful when debugging
	//
	// TODO: separate rundeps and builddeps
	pkg.RunDeps, err = ypkgYml.ParseRunDeps()
	if err != nil {
		err = fmt.Errorf("Failed to parse rundeps of %s: %w", pkg.Source, err)
		return
	}

	if ypkgYml.Clang {
		pkg.BuildDeps = append(pkg.BuildDeps, "llvm-clang-devel")
	}

	if !utils.PathExists(pspecFile) {
		return
	}

	pspecXml, err := pspec.Load(pspecFile)
	if err != nil {
		err = fmt.Errorf("Failed to load pspec_x86_64.xml for %s: %w", dir, err)
		return
	}
	for _, subPkg := range pspecXml.Packages {
		pkg.Provides = append(pkg.Provides, subPkg.Name)

		for _, pcProvide := range ypkg.GetPcProvidesFromPspecPkg(&subPkg) {
			pkg.Provides = append(pkg.Provides, pcProvide)
		}
	}

	slices.Sort(pkg.BuildDeps)
	slices.Sort(pkg.RunDeps)
	slices.Sort(pkg.Provides)

	if !utils.PathExists(cfgFile) {
		return
	}

	if abConfig != nil {
		for _, ignore := range abConfig.Solver.Ignore {
			pkg.Ignores = append(pkg.Ignores, ignore)
		}
		slices.Sort(pkg.Ignores)
	}

	return
}

func parseEopkgIndex(idx *index.Index) (state *BinaryState, err error) {
	state = &BinaryState{}
	state.packages = make([]Package, len(idx.Packages))
	state.pvdToPkgIdx = make(map[string]int)
	state.srcToPkgIds = make(map[string][]int)

	for idx, ipkg := range idx.Packages {
		pvd := fmt.Sprintf("name(%s)", ipkg.Name)
		if ext, ok := state.pvdToPkgIdx[pvd]; ok {
			err = fmt.Errorf("Duplicate provider %s, %s provides but already provided by %s", pvd, ipkg.Name, state.packages[ext].Show(true, false))
			return
		}

		var pkg Package
		pkg.Source = ipkg.Source.Name
		pkg.Names = append(pkg.Names, ipkg.Name)
		// TODO: we should be able to add pkgconfig provides as well
		pkg.Provides = append(pkg.Provides, fmt.Sprintf("name(%s)", ipkg.Name))

		latest := ipkg.History[0]
		pkg.Release = latest.Release
		pkg.Version = latest.Version

		for _, dep := range ipkg.RuntimeDependencies {
			pkg.RunDeps = append(pkg.RunDeps, dep.Name)
		}

		state.pvdToPkgIdx[pvd] = idx
		state.srcToPkgIds[ipkg.Source.Name] = append(state.srcToPkgIds[ipkg.Source.Name], idx)
		state.packages[idx] = pkg
	}

	return
}

func loadEopkgIndex(path string) (*BinaryState, error) {
	idx, err := index.Load(path)
	if err != nil {
		err = fmt.Errorf("Failed to load index at %s: %w", path, err)
		return nil, err
	}

	return parseEopkgIndex(idx)
}

func loadEopkgRepo(name string) (st *BinaryState, err error) {
	if name != "stable" && name != "unstable" {
		err = fmt.Errorf("loadEopkgRepo: only supports \"stable\" or \"unstable\" repo, got %s", name)
		return
	}

	indexUrl := fmt.Sprintf("https://packages.getsol.us/%s/eopkg-index.xml.xz", name)
	resp, err := http.Get(indexUrl)
	slog.Debug("Fetching index", "url", indexUrl)
	if err != nil {
		err = fmt.Errorf("Failed to fetch binary index from url %s: %w", indexUrl, err)
		return
	}

	r, err := xz.NewReader(resp.Body)
	if err != nil {
		err = fmt.Errorf("Failed to create XZ reader with binary index from url %s: %w", indexUrl, err)
		return
	}

	dec := xml.NewDecoder(r)
	var i index.Index
	err = dec.Decode(&i)
	if err != nil {
		err = fmt.Errorf("Failed to decode binary index from url %s: %w", indexUrl, err)
		return
	}

	st, err = parseEopkgIndex(&i)
	return
}
