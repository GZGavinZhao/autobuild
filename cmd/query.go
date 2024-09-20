// SPDX-FileCopyrightText: Copyright © 2020-2023 Serpent OS Developers
//
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"errors"
	"fmt"

	"github.com/DataDrake/waterlog"
	"github.com/GZGavinZhao/autobuild/common"
	st "github.com/GZGavinZhao/autobuild/state"
	"github.com/GZGavinZhao/autobuild/utils"
	"github.com/spf13/cobra"
	"github.com/yourbasic/graph"
)

var (
	dotPath  string
	tiers    bool
	forward  int
	reverse  int
	detailed bool
	showSub  bool

	cmdQuery = &cobra.Command{
		Use:   "query [src|bin|repo:path] [names/providers]",
		Short: "Query the build order of the given source recipes and providers",
		Long: `Query the build order of the given source recipes or the source recipes that provide the given providers.

For example: autobuild query src:../packages rocm-clr pytorch
	
Be aware that the names specified are the "name" field of the recipe. If you
want to query specific packages, use the provider syntax, such as "name(foo)".

When no arguments are passed, it tries to compute a build order of all the
packages it can find.`,
		Run: runQuery,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return errors.New("expects one arg for path to binary index or source repo")
			}
			return nil
		},
	}
)

func init() {
	cmdQuery.Flags().StringVar(&dotPath, "dot", "", "store the final build graph at the specified location in the DOT format")
	cmdQuery.Flags().BoolVarP(&tiers, "tiers", "t", false, "output tier-ed build order")
	cmdQuery.Flags().IntVarP(&forward, "forward", "F", 0, "extra level(s) of packages that depends on the list provided")
	cmdQuery.Flags().IntVarP(&reverse, "reverse", "R", 0, "extra level(s) of packages that the list provided depends on")
	// TODO(GZGavinZhao): do we always want detailed output?
	// Maybe we should output packages that are not parts of the query but are
	// on the dependency chain with a different color?
	cmdQuery.Flags().BoolVar(&detailed, "detailed", true, "report more detailed dependency chains during cycles output")
	cmdQuery.Flags().BoolVar(&showSub, "show-sub", false, "show the subpackages that a node represents instead of just the recipe name")
}

func execQuery(state st.State, queries []string) (res [][]common.Package, err error) {
	depGraph := state.DepGraph()
	if depGraph == nil {
		err = errors.New("Adjacency map for dependency graph is nil")
		return
	}

	revGraph := graph.Transpose(depGraph)
	qset := map[int]bool{}

	for _, query := range queries {
		var ids []int

		if ids = st.GetSourceIds(state, query); len(ids) == 0 {
			if _, idx := st.GetPackage(state, query); idx != -1 {
				ids = append(ids, idx)
			}
		}

		if len(ids) == 0 {
			err = fmt.Errorf("Unable to find package or provider %s", query)
			return
		}

		for _, idx := range ids {
			if idx < 0 {
				err = fmt.Errorf("Unable to find package %s", query)
				return
			}

			// TODO: currently there's a panic/fatal log when building the graph if
			// a dep is nonexistent, so we can ignore this for now.
			// if unresolved := pkg.Resolve(state.NameToSrcIdx(), state.Packages()); len(unresolved) > 0 {
			// 	waterlog.Warnf("Package %s has unresolved build dependencies, build graph may be incomplete:", pkg.Name)
			// 	for _, pkg := range unresolved {
			// 		waterlog.Printf(" %s", pkg)
			// 	}
			// 	waterlog.Println()
			// }

			qset[idx] = true
			utils.BFSWithDepth(depGraph, idx, func(node int, depth int) bool {
				if depth > forward {
					return true
				}
				qset[node] = true
				return false
			})
			if reverse > 0 {
				utils.BFSWithDepth(revGraph, idx, func(node int, depth int) bool {
					if depth > reverse {
						return true
					}
					qset[node] = true
					return false
				})
			}
		}
	}
	// waterlog.Debugf("query: %q\n", queries)
	// waterlog.Debugf("qset: %v\n", qset)
	waterlog.Goodln("Found all requested packages in state!")

	return st.QueryOrder(state, func(i int) bool { return qset[i] })
}

func runQuery(cmd *cobra.Command, args []string) {
	tpath := args[0]

	state, err := st.LoadState(tpath)
	if err != nil {
		waterlog.Fatalf("Failed to parse state: %s\n", err)
	}
	waterlog.Goodln("Successfully parsed state!")

	var queries []string
	if len(args) < 2 {
		waterlog.Infoln("No packages are provided, will try to query all packages")
		queries = make([]string, len(state.Packages()))
		for i, pkg := range state.Packages() {
			queries[i] = pkg.Source
		}
	} else {
		queries = args[1:]
	}
	queries = utils.Uniq2(queries)

	order, err := execQuery(state, queries)
	if err != nil {
		if qerr, ok := err.(st.QueryHasCyclesErr); ok {
			waterlog.Errorln("Graph contains cycles:")
			for cycleIdx, cycle := range qerr.Cycles {
				waterlog.Errorf("Cycle %d: ", cycleIdx+1)
				for _, pkg := range cycle.Members {
					fmt.Printf("%s ", pkg.Show(showSub, true))
				}
				fmt.Println()

				waterlog.Warnf("One of the dependency chains that led to this cycle: ")
				for _, pkg := range cycle.Chain {
					fmt.Printf("%s -> ", pkg.Show(showSub, true))
				}
				fmt.Println(cycle.Chain[0].Show(showSub, true))
			}
		}
		waterlog.Fatalf("Failed to query order: %s\n", err)
	}

	if tiers {
		for tierIdx, tier := range order {
			waterlog.Goodf("Tier %d: ", tierIdx+1)
			for _, pkg := range tier {
				fmt.Printf("%s ", pkg.Show(showSub, true))
			}
			fmt.Println()
		}
	} else {
		waterlog.Good("Build order: ")
		for _, pkg := range utils.Flatten(order) {
			fmt.Printf("%s ", pkg.Show(showSub, true))
		}
		fmt.Println()
	}
}
