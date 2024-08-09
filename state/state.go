// SPDX-FileCopyrightText: Copyright © 2020-2023 Serpent OS Developers
//
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/DataDrake/waterlog"
	"github.com/GZGavinZhao/autobuild/common"
	"github.com/GZGavinZhao/autobuild/utils"
	"github.com/yourbasic/graph"
)

var (
	InvalidTPathError error = errors.New("Invalid tpath! Must be in the form \"[src|bin|repo]:path\"!")
)

type State interface {
	Packages() []common.Package
	SrcToPkgIds() map[string][]int
	PvdToPkgIdx() map[string]int
	DepGraph() *graph.Immutable
	// GetPackage(string) (common.Package, int)
	// GetPackageIdx(string) int
	// PackageExists(string) bool
}

func GetSourceIds(s State, name string) []int {
	return s.SrcToPkgIds()[name]
}

func GetPackage(s State, pvd string) (common.Package, int) {
	idx, ok := s.PvdToPkgIdx()[pvd]
	if !ok {
		return common.Package{}, -1
	} else {
		return s.Packages()[idx], idx
	}
}

func GetPackageIdx(s State, pvd string) int {
	return s.PvdToPkgIdx()[pvd]
}

func PackageExists(s State, pvd string) bool {
	_, ok := s.PvdToPkgIdx()[pvd]
	return ok
}

func ValidTPath(tpath string) bool {
	splitted := strings.Split(tpath, ":")

	if len(splitted) > 2 {
		return false
	}

	return slices.Contains([]string{"src", "bin", "repo"}, splitted[0])
}

func LoadState(tpath string) (state State, err error) {
	if !ValidTPath(tpath) {
		err = InvalidTPathError
		return
	}

	splitted := strings.Split(tpath, ":")
	if splitted[0] == "src" {
		state, err = LoadSource(splitted[1])
	} else if splitted[0] == "bin" {
		state, err = LoadBinary(splitted[1])
	} else {
		state, err = LoadEopkgRepo(splitted[1])
	}

	return
}

func Changed(old *State, cur *State) (res []Diff) {
	for src, ids := range (*cur).SrcToPkgIds() {
		idx := ids[0]
		pkg := (*cur).Packages()[idx]
		// WARNING:
		// we assume that packages that correspond to the same source recipe
		// always have the same release number and version.
		//
		// In general, this should always hold, but we should probably check it
		// somewhere.
		oldIds, found := (*old).SrcToPkgIds()[src]

		if !found {
			res = append(res, Diff{
				Idx:    idx,
				RelNum: pkg.Release,
				Ver:    pkg.Version,
			})
			continue
		}

		oldPkg := (*old).Packages()[oldIds[0]]
		if oldPkg.Release != pkg.Release || oldPkg.Version != pkg.Version {
			res = append(res, Diff{
				Idx:       idx,
				OldIdx:    oldIds[0],
				RelNum:    pkg.Release,
				OldRelNum: oldPkg.Release,
				Ver:       pkg.Version,
				OldVer:    oldPkg.Version,
			})
		}
	}

	return
}

func QueryOrder(state State, choose func(int) bool) (res [][]common.Package, err error) {
	depGraph := state.DepGraph()
	if depGraph == nil {
		waterlog.Fatalf("Failed to obtain adjacency map for dependency graph: %s\n", err)
	}

	lifted := graph.Sort(utils.LiftGraph(depGraph, choose))
	if err != nil {
		err = fmt.Errorf("Failed to lift final graph from requested nodes: %s", err)
		return
	}
	waterlog.Goodln("Successfully built dependency graph!")

	waterlog.Debugf("depgraph hash: %s\n", utils.GraphHash(depGraph))
	waterlog.Debugf("depgraph stats: %+v\n", graph.Check(depGraph))
	waterlog.Debugf("liftgraph hash: %s\n", utils.GraphHash(lifted))
	waterlog.Debugf("liftgraph stats: %+v\n", graph.Check(lifted))

	order, ok := utils.TieredTopSort(lifted)
	if !ok {
		// Try to dump cycles if topological sort failed.
		// if cycles, err := graph.StrongComponents(lifted); err == nil {
		cycles := graph.StrongComponents(lifted)
		cycles = utils.Filter(cycles, func(cycle []int) bool { return len(cycle) > 1 })
		if len(cycles) == 0 {
			err = errors.New("Cannot topological sort but no cycles detected?!?")
			return
		}

		cyclesErr := QueryHasCyclesErr{}
		for _, cycle := range cycles {
			if len(cycle) <= 1 {
				continue
			}

			thisCycle := Cycle{}

			thisCycle.Members = make([]common.Package, len(cycle))
			for idx, nodeIdx := range cycle {
				thisCycle.Members[idx] = state.Packages()[nodeIdx]
			}

			// the order in `cycle` may not be deterministic, so we have to
			// deterministically choose a starting node by ourselves
			startIdx := 0
			for idx, nodeIdx := range cycle {
				if nodeIdx < cycle[startIdx] {
					startIdx = idx
				}
			}
			nextIdx := (startIdx + 1) % len(cycle)

			// We always want the longer shortest path
			depPath := utils.LongerShortestPath(depGraph, cycle[startIdx], cycle[nextIdx])
			if len(depPath) < 2 {
				err = fmt.Errorf("Failed to calculate dependency path that led to this cycle, got: %q", depPath)
				return
			}

			thisCycle.Chain = make([]common.Package, len(depPath))
			for idx, nodeIdx := range depPath {
				thisCycle.Chain[idx] = state.Packages()[nodeIdx]
			}
			cyclesErr.Cycles = append(cyclesErr.Cycles, thisCycle)
		}

		err = cyclesErr
		return
	}

	// Note that we still need an extra filter on the tier output,
	// because due to the limitation of the graph API, the lifted graph
	// includes nodes [0, n), not just the nodes in `query`/`qset`, so they will
	// appear in the topological sort output.
	for tIdx, tier := range order {
		tier = utils.Filter(tier, choose)
		res = append(res, make([]common.Package, len(tier)))
		for idx, pkgIdx := range tier {
			res[tIdx][idx] = state.Packages()[pkgIdx]
		}
	}
	return
}
