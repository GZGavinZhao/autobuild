package state

import (
	"github.com/GZGavinZhao/autobuild/common"
	"github.com/dominikbraun/graph"
)

type SourceState struct {
	packages     []common.Package
	nameToSrcIdx map[string]int
	depGraph     *graph.Graph[int, int]
	isGit        bool
}

func (s *SourceState) Packages() []common.Package {
	return s.packages
}

func (s *SourceState) NameToSrcIdx() map[string]int {
	return s.nameToSrcIdx
}

func (s *SourceState) DepGraph() *graph.Graph[int, int] {
	return s.depGraph
}

func (s *SourceState) IsGit() bool {
	return s.isGit
}

func (s *SourceState) BuildGraph() {

}

func LoadSource(path string) (state SourceState, err error) {
	return
}

func (cur *SourceState) Changed(old *State) (res []Diff) {
	for idx, pkg := range cur.packages {
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
				OldVer:    pkg.Version,
			})
		}
	}

	return
}
