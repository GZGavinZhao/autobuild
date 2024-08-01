// SPDX-FileCopyrightText: Copyright © 2020-2023 Serpent OS Developers
//
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"errors"
	"slices"
	"strings"

	"github.com/GZGavinZhao/autobuild/common"
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

func GetSourceIds(s State, name string) ([]int) {
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
	for idx, pkg := range (*cur).Packages() {
		// WARNING:
		// we assume that packages that correspond to the same source recipe
		// always have the same release number and version.
		//
		// In general, this should always hold, but we should probably check it
		// somewhere.
		oldIds, found := (*old).SrcToPkgIds()[pkg.Source]

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
