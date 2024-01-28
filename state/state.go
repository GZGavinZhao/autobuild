// SPDX-FileCopyrightText: Copyright © 2020-2023 Serpent OS Developers
//
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"errors"
	"slices"
	"strings"

	"github.com/GZGavinZhao/autobuild/common"
	"github.com/dominikbraun/graph"
)

var (
	InvalidTPathError error = errors.New("Invalid tpath! Must be in the form \"[src|bin|repo]:path\"!")
)

type State interface {
	Packages() []common.Package
	NameToSrcIdx() map[string]int
	DepGraph() *graph.Graph[int, int]
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
		oldIdx, found := (*old).NameToSrcIdx()[pkg.Name]

		if !found {
			res = append(res, Diff{
				Idx:    idx,
				RelNum: pkg.Release,
				Ver:    pkg.Version,
			})
			continue
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
