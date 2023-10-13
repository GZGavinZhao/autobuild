// SPDX-FileCopyrightText: Copyright © 2020-2023 Serpent OS Developers
//
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/DataDrake/waterlog"
	"github.com/DataDrake/waterlog/format"
	"github.com/charlievieth/fastwalk"
	"github.com/dominikbraun/graph"
	"github.com/dominikbraun/graph/draw"
	"github.com/getsolus/infrastructure-tooling/autobuild/ypkg"
	"github.com/getsolus/libeopkg/index"
	"github.com/getsolus/libeopkg/pspec"
	"golang.org/x/exp/slices"
)

var (
	badPackages  = [...]string{"haskell-http-client-tls"}
	srcPkgs      = []Package{}
	nameToSrcIdx = make(map[string]int)
)

type Package struct {
	Path      string
	Name      string
	Version   string
	Release   int
	Provides  []string
	BuildDeps []string
	Resolved  bool
	Synced    bool
}

func readSrcPkgs(path string) (pkgs []Package, err error) {
	walkConf := fastwalk.Config{
		Follow: false,
	}
	// ch := make(chan int)
	var mutex sync.Mutex
	var cntA atomic.Uint32

	err = fastwalk.Walk(&walkConf, path, func(path string, d fs.DirEntry, err error) error {
		// err = filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() {
			return nil
		}

		if fileExists(filepath.Join(path, ignoreFile)) {
			return nil
		}

		// Some hard-coded problematic packages
		if slices.Contains(badPackages[:], filepath.Base(path)) {
			return nil
		}

		// Check if the given directory contains a package definition
		pkgFile := filepath.Join(path, "package.yml")
		pspecFile := filepath.Join(path, "pspec_x86_64.xml")
		// TODO: handle legacy XML packages too
		if !fileExists(pkgFile) {
			return nil
		}

		ypkgYml, err := ypkg.Load(pkgFile)
		if err != nil {
			return errors.New(fmt.Sprintf("Failed to load package.yml file for %s: %s", path, err))
		}

		pkg := Package{
			Path:      path,
			Name:      ypkgYml.Name,
			Version:   ypkgYml.Version,
			Release:   ypkgYml.Release,
			BuildDeps: ypkgYml.BuildDeps,
			Synced:    false,
			Resolved:  true,
		}
		cntA.Add(1)

		if fileExists(pspecFile) {
			pspecXml, err := pspec.Load(pspecFile)
			if err != nil {
				return errors.New(fmt.Sprintf("Failed to load pspec_x86_64.xml for %s: %s", path, err))
			}
			for _, subPkg := range pspecXml.Packages {
				pkg.Provides = append(pkg.Provides, subPkg.Name)

				for _, pcProvide := range getPcProvides(&subPkg) {
					pkg.Provides = append(pkg.Provides, pcProvide)
				}
			}
		} else {
			pkg.Resolved = false
		}

		// ch <- pkg
		mutex.Lock()
		pkgs = append(pkgs, pkg)
		mutex.Unlock()

		return filepath.SkipDir
	})
	if err != nil {
		return
	}

	// cnt := int(cntA.Load())
	// pkgs = make([]Package, cnt)

	// for i := 0; i < cnt; i++ {
	// 	pkgs[i] = <-ch
	// }

	if int(cntA.Load()) != len(pkgs) {
		err = errors.New("cnt doesn't match the length of pkgs!")
		return
	}

	return
}

func main() {
	waterlog.SetLevel(7)
	waterlog.SetFormat(format.Min)

	sourcesPath := os.Args[1]
	indexPath := os.Args[2]

	var err error
	srcPkgs, err = readSrcPkgs(sourcesPath)
	if err != nil {
		waterlog.Fatalf("Failed to walk through sources: %s\n", err)
	}
	waterlog.Goodln("Scan for source packages complete. Now trying to resolve dependencies...")

	// Iterate through every source package (`a`) and maps the name of all of
	// its binary packages (e.g. `a-devel`) and providers (`pkgconfig(a)`) to
	// the source.
	for idx, srcPkg := range srcPkgs {
		nameToSrcIdx[srcPkg.Name] = idx
		for _, name := range srcPkg.Provides {
			nameToSrcIdx[name] = idx
		}
	}

	// Iterate through every source package, and check if all of their
	// dependencies are present in the source repository.
	//
	// If not, then it's possible that missing dependency is a new package that
	// needs to be built to generate the `pspec_x86_64.xml` file that shows all
	// the binary packages that a source package provides. This usually happens
	// when `a` is a new package that has yet to be built locally and some
	// package `b` depends on `a-devel`.
	for idx, srcPkg := range srcPkgs {
		if !srcPkg.Resolved {
			continue
		}

		for _, dep := range srcPkg.BuildDeps {
			_, depFound := nameToSrcIdx[dep]
			if !depFound {
				// waterlog.Infof("Package %s is unresolved due to dependency %s\n", srcPkg.Name, dep)
				srcPkgs[idx].Resolved = false
				break
			}
		}
	}
	waterlog.Goodln("Dependency resolving complete. Now scanning binary index...")

	eopkgIndex, err := index.Load(indexPath)
	if err != nil {
		waterlog.Fatalf("Failed to load index file at %s: %s\n", indexPath, err)
	}

	// Iterate through the eopkg index and check if there are version/release
	// discrepancies between the source repository and the binary index.
	for _, binPkg := range eopkgIndex.Packages {
		if binPkg.Name != binPkg.Source.Name {
			continue
		}

		pkgName := binPkg.Name
		binRelNum := binPkg.History[0].Release
		binVer := binPkg.History[0].Version

		srcIdx, srcFound := nameToSrcIdx[binPkg.Name]
		if !srcFound {
			waterlog.Warnf("No source was found for binary package %s\n", binPkg.Name)
			continue
		}
		srcPkg := &srcPkgs[srcIdx]

		srcRelNum := srcPkg.Release
		srcVer := srcPkg.Version

		srcPkg.Synced = true
		if binRelNum > srcRelNum {
			waterlog.Warnf("Package %s has an older source release %d and version %s, but index provides release %d and version %s\n", pkgName, srcRelNum, srcVer, binRelNum, binVer)
		} else if binRelNum == srcRelNum && binVer != srcVer {
			waterlog.Warnf("Package %s has the same release number but has version mismatch between source %s and binary %s\n", pkgName, srcVer, binVer)
		} else if binRelNum < srcRelNum {
			srcPkg.Synced = false
			waterlog.Infof("%s: %s (%d) -> %s (%d)\n", pkgName, binVer, binRelNum, srcVer, srcRelNum)
		}
	}
	waterlog.Goodln("Scanning binary index complete. Constructing dependency graph...")

	depGraph := graph.New(graph.IntHash, graph.Directed())
	for pkgIdx, pkg := range srcPkgs {
		attrsFunc := func(p *graph.VertexProperties) {
			p.Attributes["label"] = pkg.Name
			p.Attributes["color"] = "2"
			p.Attributes["fillcolor"] = "1"
			if pkg.Synced {
				// if !pkg.Resolved {
				// 	waterlog.Fatalf("Package %s is synced but not all of its dependencies are solved!", pkg.Name)
				// }
				p.Attributes["colorscheme"] = "greens3"
				p.Attributes["style"] = "filled"
			} else if !pkg.Resolved {
				p.Attributes["colorscheme"] = "reds3"
				p.Attributes["style"] = "filled"
			} else if !pkg.Synced {
				p.Attributes["colorscheme"] = "ylorbr3"
				p.Attributes["style"] = "filled"
			}
		}

		depGraph.AddVertex(pkgIdx, attrsFunc)
	}
	for pkgIdx, pkg := range srcPkgs {
		for _, dep := range pkg.BuildDeps {
			depIdx, depFound := nameToSrcIdx[dep]
			if !depFound {
				// waterlog.Fatalf("Dependency %s of package %s is not found!\n", dep, pkg.Name)
			} else if pkgIdx != depIdx {
				err := depGraph.AddEdge(depIdx, pkgIdx)
				if err != nil && !errors.Is(err, graph.ErrEdgeAlreadyExists) {
					waterlog.Fatalf("Failed to create edge from %s to %s: %s\n", dep, pkg.Name, err)
				}
			}
		}
	}
	gsize, _ := depGraph.Size()
	gord, _ := depGraph.Order()
	waterlog.Debugf("Full edge count: %d\n", gsize)
	waterlog.Debugf("Full package count: %d\n", gord)
	waterlog.Goodln("Constructing dependency graph complete. Determining which packages to build...")

	// var subg graph.Graph[int, int]
	// requestedIdx := []int{}

	// if len(os.Args) > 3 {
	// 	for _, pkg := range os.Args[3:] {
	// 		pkgIdx, pkgFound := nameToSrcIdx[pkg]
	// 		if !pkgFound {
	// 			waterlog.Fatalf("Unable to find package", pkg)
	// 		}
	// 		if srcPkgs[pkgIdx].Synced {
	// 			waterlog.Warnf("Warning: package %s is already synced with the binary index, ignoring...\n", pkg)
	// 		} else {
	// 			requestedIdx = append(requestedIdx, pkgIdx)
	// 		}
	// 	}

	// 	subg, err = subgraph(&depGraph, requestedIdx[:])
	// 	if err != nil {
	// 		waterlog.Fatalf("Failed to generate subgraph from depgraph: %s\n", err)
	// 	}
	// 	subgord, _ := subg.Order()
	// 	waterlog.Debugf("Subgraph package count: %d\n", subgord)
	// } else {
	// 	subg, err = depGraph.Clone()
	// 	if err != nil {
	// 		waterlog.Fatalf("Failed to clone dependency graph for isolation: %s\n", err)
	// 	}
	// }

	// // subgDot, _ := os.Create("./subg.gv")
	// // _ = draw.DOT(subg, subgDot)

	// toBuildPkgsIdx := []int{}

	// // subgAdjMap, err := subg.AdjacencyMap()
	// // if err != nil {
	// // 	waterlog.Fatalf("Failed to get adjacency map for subgraph: %s\n", err)
	// // }
	// // for pkgIdx := range subgAdjMap {
	// // 	pkg := srcPkgs[pkgIdx]
	// // 	// if !pkg.Resolved || !pkg.Synced {
	// // 	if pkg.Resolved && !pkg.Synced {
	// // 		toBuildPkgsIdx = append(toBuildPkgsIdx, pkgIdx)
	// // 	}
	// // }
	// for _, idx := range requestedIdx {
	// 	if srcPkgs[idx].Resolved && !srcPkgs[idx].Synced {
	// 		toBuildPkgsIdx = append(toBuildPkgsIdx, idx)
	// 	}
	// }

	// if len(toBuildPkgsIdx) == 0 {
	// 	waterlog.Goodln("No updates detected.")
	// 	os.Exit(0)
	// }

	// waterlog.Info("Calculating build order for the following packages:")
	// for _, pkgIdx := range toBuildPkgsIdx {
	// 	fmt.Printf(" %s", srcPkgs[pkgIdx].Name)
	// }
	// fmt.Println()

	// fing, err := isolate(&subg, toBuildPkgsIdx[:])
	// if err != nil {
	// 	waterlog.Fatalf("Failed to isolate final build graph: %s\n", err)
	// }
	// fingord, _ := fing.Order()
	// waterlog.Debugf("Final build graph pcackage count: %d\n", fingord)

	qset := map[int]bool{}
	if len(os.Args) > 3 {
		for _, pkg := range os.Args[3:] {
			idx, ok := nameToSrcIdx[pkg]
			if !ok {
				waterlog.Fatalf("Unable to find package", pkg)
			}

			if !srcPkgs[idx].Resolved {
				waterlog.Warnf("Package %s has unresolved build dependencies, build graph may be incomplete!\n", srcPkgs[idx].Name)
			}

			qset[idx] = true
		}
	} else {
		for idx, pkg := range srcPkgs {
			if !pkg.Synced /* || !pkg.Resolved */ {
				qset[idx] = true
			}
		}
	}

	fing, err := liftgraph(&depGraph, func(i int) bool {
		return qset[i]
	})
	if err != nil {
		waterlog.Fatalf("Failed to lift final graph from requested nodes: %s\n", err)
	}

	fingDot, _ := os.Create("./fing.gv")
	_ = draw.DOT(fing, fingDot)

	order, err := graph.TopologicalSort(fing)
	if err != nil {
		waterlog.Fatalf("Failed to get topological sort order: %s\n", err)
	}

	waterlog.Good("Build order:")
	for _, orderIdx := range order {
		fmt.Printf(" %s", srcPkgs[orderIdx].Name)
	}
	fmt.Println()
}
