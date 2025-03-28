// SPDX-FileCopyrightText: Copyright © 2020-2023 Serpent OS Developers
//
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/DataDrake/waterlog"
	"github.com/GZGavinZhao/autobuild/utils"
	"github.com/yourbasic/graph"
)

var (
	InvalidTPathError error = errors.New("Invalid tpath! Must be in the form \"[src|bin|repo]:path\"!")
)

type State interface {
	Packages() []Package
	SrcToPkgIds() map[string][]int
	PvdToPkgIdx() map[string]int
	DepGraph() *graph.Immutable
}

func GetSourceIds(s State, name string) []int {
	return s.SrcToPkgIds()[name]
}

func GetPackage(s State, pvd string) (Package, int) {
	idx, ok := s.PvdToPkgIdx()[pvd]
	if !ok {
		return Package{}, -1
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
	} else if splitted[0] == "repo" {
		state, err = LoadRepo(splitted[1])
	} else {
		// state, err = LoadEopkgRepo(splitted[1])
		err = errors.ErrUnsupported
	}

	return
}

func ChooseLatestSource(st *State, source string) (res int, err error) {
	ids, found := (*st).SrcToPkgIds()[source]
	if !found {
		err = fmt.Errorf("ChooseLatestSource: %s doesn't exist in state", source)
		return
	}

	res = ids[0]
	resPkg := (*st).Packages()[res]

	for _, id := range ids {
		curPkg := (*st).Packages()[id]

		if curPkg.Release == resPkg.Release && curPkg.Version == resPkg.Version {

		} else if resPkg.Release != curPkg.Release {
			var oldPkg, newPkg Package
			if resPkg.Release < curPkg.Release {
				oldPkg = resPkg
				newPkg = curPkg

				res = id
				resPkg = curPkg
			} else {
				oldPkg = curPkg
				newPkg = resPkg
			}

			slog.Debug("Same source produces different release, preferring newer", "old", oldPkg.Show(true, false), "new", newPkg.Show(true, false), "oldRel", oldPkg.Release, "newRel", newPkg.Release, "oldVer", oldPkg.Version, "newVer", newPkg.Version)
		} else if resPkg.Version != curPkg.Version {
			slog.Error("Same source produces same release with different versions", "left", resPkg.Show(true, false), "right", curPkg.Show(true, false), "er", resPkg.Version, "leftRel", resPkg.Release, "rightRel", curPkg.Release)
			err = fmt.Errorf("ChooseLatestSource: %s produces packages with different relnos", source)
			return
		}
	}

	return
}

func Changed(old *State, cur *State) (res []Diff, err error) {
	for src, _ := range (*cur).SrcToPkgIds() {
		var idx int
		idx, err = ChooseLatestSource(cur, src)
		if err != nil {
			return
		}
		pkg := (*cur).Packages()[idx]

		// WARNING:
		// we assume that packages that correspond to the same source recipe
		// always have the same release number and version.
		//
		// In general, this should always hold, but we should probably check it
		// somewhere.

		if _, found := (*old).SrcToPkgIds()[src]; !found {
			res = append(res, Diff{
				Idx:    idx,
				RelNum: pkg.Release,
				Ver:    pkg.Version,
			})
			continue
		}

		var oldIdx int
		oldIdx, err = ChooseLatestSource(old, src)
		if err != nil {
			return res, err
		}

		oldPkg := (*old).Packages()[oldIdx]
		if oldPkg.Release != pkg.Release || oldPkg.Version != pkg.Version {
			res = append(res, Diff{
				Idx:       idx,
				OldIdx:    oldIdx,
				RelNum:    pkg.Release,
				OldRelNum: oldPkg.Release,
				Ver:       pkg.Version,
				OldVer:    oldPkg.Version,
			})
		}
	}

	return
}

func QueryOrder(state State, choose func(int) bool) (res [][]Package, err error) {
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

			thisCycle.Members = make([]Package, len(cycle))
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

			thisCycle.Chain = make([]Package, len(depPath))
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
		res = append(res, make([]Package, len(tier)))
		for idx, pkgIdx := range tier {
			res[tIdx][idx] = state.Packages()[pkgIdx]
		}
	}
	return
}
