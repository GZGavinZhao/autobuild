// SPDX-FileCopyrightText: Copyright © 2020-2023 Serpent OS Developers
//
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"cmp"
	"fmt"
	"io/fs"
	"log/slog"
	"path/filepath"
	"slices"
	"sync"

	"github.com/GZGavinZhao/autobuild/config"
	"github.com/GZGavinZhao/autobuild/utils"
	"github.com/charlievieth/fastwalk"
	"github.com/yourbasic/graph"
)

var (
	badPackages = [...]string{"haskell-http-client-tls"}
)

type SourceState struct {
	packages    []Package
	depGraph    *graph.Immutable
	pvdToPkgIdx map[string]int
	srcToPkgIds map[string][]int
	isGit       bool
	root        string
}

func (s *SourceState) Packages() []Package {
	return s.packages
}

func (s *SourceState) DepGraph() *graph.Immutable {
	return s.depGraph
}

func (s *SourceState) SrcToPkgIds() map[string][]int {
	return s.srcToPkgIds
}

func (s *SourceState) PvdToPkgIdx() map[string]int {
	return s.pvdToPkgIdx
}

func (s *SourceState) IsGit() bool {
	return s.isGit
}

func (s *SourceState) buildGraph() {
	g := graph.New(len(s.packages))

	for pkgIdx, pkg := range s.packages {
		for _, dep := range pkg.BuildDeps {
			depIdx, depFound := s.pvdToPkgIdx[dep]

			// Check if this package or any of its providers are requested to be
			// ignored
			for _, ignore := range pkg.Ignores {
				depPkg := s.packages[depIdx]
				// TODO: clearly logging on why dropped
				if depPkg.Source == ignore || slices.Contains(depPkg.Names, ignore) || slices.Contains(depPkg.Provides, ignore) {
					// waterlog.Debugf("Dropping dependency %s (requested by %s) due to ignore %s\n", pkg.Show(true, false), dep, ignore)
					slog.Debug("Dropping depdendency", "dep", dep, "pkg", pkg.Show(true, false), "ignore", ignore)
					continue
				}
			}

			if !depFound {
				// waterlog.Warnf("Dependency %s of package %s is not found!\n", dep, pkg.Show(true, false))
				slog.Warn("Dependency not found", "dep", dep, "pkg", pkg.Show(true, false))
			} else if pkgIdx != depIdx {
				// prevent self loops
				g.Add(depIdx, pkgIdx)
			}
		}
	}

	s.depGraph = graph.Sort(g)
}

func LoadSource(dir string) (state *SourceState, err error) {
	state = &SourceState{}
	state.pvdToPkgIdx = make(map[string]int)
	state.srcToPkgIds = make(map[string][]int)

	if utils.PathExists(filepath.Join(dir, ".git")) {
		state.isGit = true
	}

	walkConf := fastwalk.Config{
		Follow: false,
	}
	_ = walkConf // this is a no-op purely for ease of debugging
	var mutex sync.Mutex

	// err = filepath.WalkDir(path, func(pkgpath string, d fs.DirEntry, err error) error {
	err = fastwalk.Walk(&walkConf, dir, func(pkgdir string, entry fs.DirEntry, err error) error {
		if !entry.IsDir() {
			return nil
		}

		// Some hard-coded problematic packages
		if slices.Contains(badPackages[:], filepath.Base(pkgdir)) {
			return nil
		}

		var abConfig config.AutobuildConfig
		for _, cfgFile := range []string{"autobuild.yaml", "autobuild.yml"} {
			cfgFile = filepath.Join(pkgdir, cfgFile)
			if utils.PathExists(cfgFile) {
				slog.Debug("LoadSource: loading config file", "pkg", filepath.Base(pkgdir), "configPath", cfgFile)
				abConfig, err = config.Load(cfgFile)
				if err != nil {
					return fmt.Errorf("LoadSource: failed to load autobuild config file at %s: %w", cfgFile, err)
				}

				if abConfig.Ignore {
					return filepath.SkipDir
				}

				break
			}
		}

		var pkgs []Package

		// ypkgFile := filepath.Join(pkgpath, "package.yml")
		// stoneFile := filepath.Join(pkgpath, "stone.yaml")

		if ypkgPackageExists(pkgdir) {
			if pkgs, err = loadYpkgPackage(pkgdir, &abConfig); err != nil {
				return fmt.Errorf("Failed to parse ypkg package %s: %w", pkgdir, err)
			}
			// } else if utils.PathExists(stoneFile) {
			// 	if pkgs, err = stone.ParsePackage(pkgpath, abConfig); err != nil {
			// 		return fmt.Errorf("Failed to parse %s: %w", stoneFile, err)
			// 	}
		} else {
			return nil
		}

		for i := range pkgs {
			pkgs[i].Root = dir
		}

		mutex.Lock()
		state.packages = append(state.packages, pkgs...)
		mutex.Unlock()

		return filepath.SkipDir
	})

	if err != nil {
		return
	}

	slices.SortFunc(state.packages, func(a, b Package) int {
		if a.Source == b.Source {
			// If we want to be really precise, we should compare the entire
			// `Names` slice, but just comparing the first element should be
			// enough.
			return cmp.Compare(a.Names[0], b.Names[0])
		} else {
			return cmp.Compare(a.Source, b.Source)
		}
	})

	for idx, pkg := range state.packages {
		state.srcToPkgIds[pkg.Source] = append(state.srcToPkgIds[pkg.Source], idx)

		for _, pvd := range pkg.Provides {
			if pidx, ok := state.pvdToPkgIdx[pvd]; ok && pidx != idx {
				// waterlog.Errorf("Duplicate provider for %s from %s, currently %s\n", pvd, pkg.Show(true, false), state.packages[pidx].Show(true, false))
				slog.Warn("Duplicate provider", "provider", pvd, "from", pkg.Show(true, false), "orig", state.packages[pidx].Show(true, false))
			}
			state.pvdToPkgIdx[pvd] = idx
		}
	}

	state.buildGraph()

	return
}
