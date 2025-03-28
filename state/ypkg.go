package state

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"

	"github.com/GZGavinZhao/autobuild/config"
	"github.com/GZGavinZhao/autobuild/utils"
	"github.com/GZGavinZhao/autobuild/ypkg"
	"github.com/getsolus/libeopkg/pspec"
	"gopkg.in/yaml.v3"
)

func ypkgPackageExists(dir string) bool {
	return utils.PathExists(filepath.Join(dir, ypkg.YmlFile)) || utils.PathExists(filepath.Join(dir, ypkg.PspecFile))
}

func loadYpkgPackage(dir string) (pkgs []Package, err error) {
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
		// Synced:    false
	})
	pkg := &pkgs[0]

	// Combine the rundeps of all subpackages into a single list
	// Note to self: this website can inspect yaml ast nodes:
	// https://astexplorer.net/, might be useful when debugging
	//
	// TODO: separate rundeps and builddeps
	rundeps := ypkgYml.RunDeps
	if rundeps.Kind == yaml.SequenceNode {
		for _, children := range rundeps.Content {
			if children.Kind == yaml.ScalarNode {
				pkg.BuildDeps = append(pkg.BuildDeps, children.Value)
			} else if children.Kind == yaml.MappingNode {
				for _, subpkg := range children.Content {
					for _, rundep := range subpkg.Content {
						if rundep.Kind != yaml.ScalarNode {
							continue
						}

						pkg.BuildDeps = append(pkg.BuildDeps, rundep.Value)
					}
				}
			}
		}
	} else {
		err = fmt.Errorf("%s has unknown \"rundeps\" field kind: %s", dir, rundeps.Value)
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
	slices.Sort(pkg.Provides)

	if !utils.PathExists(cfgFile) {
		return
	}

	abConfig, err := config.Load(cfgFile)
	if err != nil {
		err = errors.New(fmt.Sprintf("Failed to load autobuild config file for %s: %s", dir, err))
	}

	for _, ignore := range abConfig.Solver.Ignore {
		pkg.Ignores = append(pkg.Ignores, ignore)
	}
	slices.Sort(pkg.Ignores)

	return
}

func loadEopkgIndex(xmlPath string) (st BinaryState, err error) {
	return
}

func loadEopkgRepo(name string) (st BinaryState, err error) {
	return
}
