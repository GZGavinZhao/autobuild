// SPDX-FileCopyrightText: Copyright © 2020-2023 Serpent OS Developers
//
// SPDX-License-Identifier: MPL-2.0

package common

// import (
// 	"errors"
// 	"fmt"
// 
// 	"github.com/DataDrake/waterlog"
// 	"github.com/dominikbraun/graph"
// )
// 
// func PrepareSrcAndDepGraph(sourcesPath string, indexPath string) (srcPkgs []Package, nameToSrcIdx map[string]int, depGraph graph.Graph[int, int], err error) {
// 	nameToSrcIdx = make(map[string]int)
// 
// 	srcPkgs, err = ReadSrcPkgs(sourcesPath)
// 	if err != nil {
// 		err = errors.New(fmt.Sprintf("Failed to walk through sources: %s\n", err))
// 		return
// 	}
// 
// 	for idx, srcPkg := range srcPkgs {
// 		nameToSrcIdx[srcPkg.Name] = idx
// 		for _, name := range srcPkg.Provides {
// 			nameToSrcIdx[name] = idx
// 		}
// 	}
// 
// 	// Iterate through every source package, and check if all of their
// 	// dependencies are present in the source repository.
// 	//
// 	// If not, then it's possible that missing dependency is a new package that
// 	// needs to be built to generate the `pspec_x86_64.xml` file that shows all
// 	// the binary packages that a source package provides. This usually happens
// 	// when `a` is a new package that has yet to be built locally and some
// 	// package `b` depends on `a-devel`.
// 	for idx := range srcPkgs {
// 		srcPkgs[idx].Resolve(nameToSrcIdx, srcPkgs)
// 	}
// 
// 	waterlog.Goodln("Dependency resolving complete. Now scanning binary index...")
// 
// 	err = CheckSrcPkgsSynced(indexPath, srcPkgs[:], nameToSrcIdx)
// 	if err != nil {
// 		err = errors.New(fmt.Sprintf("Failed to compare source packages with binary index %s: %s\n", indexPath, err))
// 		return
// 	}
// 	waterlog.Goodln("Scanning binary index complete. Constructing dependency graph...")
// 
// 	depGraph, err = BuildDepGraph(srcPkgs[:], nameToSrcIdx)
// 	return
// }
