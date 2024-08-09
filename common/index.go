// SPDX-FileCopyrightText: Copyright © 2020-2023 Serpent OS Developers
//
// SPDX-License-Identifier: MPL-2.0

package common

import (
	"errors"
	"fmt"

	"github.com/DataDrake/waterlog"
	"github.com/dominikbraun/graph"
	"github.com/getsolus/libeopkg/index"
)

func CheckSrcPkgsSynced(indexPath string, srcPkgs []Package, nameToSrcIdx map[string]int) error {
	eopkgIndex, err := index.Load(indexPath)
	if err != nil {
		return err
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

	return nil
}

func BuildDepGraph(srcPkgs []Package, nameToSrcIdx map[string]int) (depGraph graph.Graph[int, int], err error) {
	depGraph = graph.New(graph.IntHash, graph.Directed(), graph.Acyclic())

	for pkgIdx, pkg := range srcPkgs {
		attrsFunc := func(p *graph.VertexProperties) {
			p.Attributes["label"] = pkg.Show(true, false)
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
				err = depGraph.AddEdge(depIdx, pkgIdx, graph.EdgeWeight(1))
				if err != nil && !errors.Is(err, graph.ErrEdgeAlreadyExists) {
					err = errors.New(fmt.Sprintf("Failed to create edge from %s to %s: %s\n", dep, pkg.Show(true, false), err))
					return
				}
			}
		}
	}

	return
}
