// SPDX-FileCopyrightText: Copyright © 2021-2023 Serpent OS Developers
//
// SPDX-License-Identifier: MPL-2.0

package cmd

import (
	"github.com/DataDrake/waterlog"
	"github.com/GZGavinZhao/autobuild/common"
	"github.com/GZGavinZhao/autobuild/state"
	"github.com/spf13/cobra"
)

var (
	cmdPush = &cobra.Command{
		Use:   "push <[src|bin|repo]:path-to-old> <[src|bin|repo]:path-to-new> <packages-to-push>",
		Short: "Push package changes to the build server",
		Long: `Essentially the same as query, but also push the packages to the build server.

When no arguments are passed, it tries to diff the new and old state to find 
the packages that are updated and push them. Otherwise, it will only try to 
push the packages that are passed.

If you get a cycles output, query the build order of those packages with 
autobuild query to get a more detailed output on the cycle.
`,
		Run: runPush,
	}
)

func init() {
	cmdPush.Flags().BoolP("force", "f", false, "whether to ignore safety checks")
	cmdPush.Flags().BoolP("dry-run", "n", true, "don't publish anything")
	cmdPush.Flags().BoolP("push", "p", true, "git push packages before publishing")
}

func runPush(cmd *cobra.Command, args []string) {
	oldTPath := args[0]
	newTPath := args[1]

	var oldState, newState state.State

	oldState, err := state.LoadState(oldTPath)
	if err != nil {
		waterlog.Fatalf("Failed to load old state %s: %s\n", oldTPath, err)
	}
	waterlog.Goodln("Successfully parsed old state!")

	newState, err = state.LoadState(newTPath)
	if err != nil {
		waterlog.Fatalf("Failed to load new state %s: %s\n", newTPath, err)
	}
	waterlog.Goodln("Successfully parsed new state!")

	bumped := []common.Package{}
	bset := make(map[int]bool)
	outdated := []common.Package{}
	bad := []common.Package{}

	waterlog.Infoln("Diffing...")
	var changes []state.Diff
	if len(args) > 2 {
		for _, name := range args[2:] {
			oldPkg, oldIdx := state.GetPackage(oldState, name)
			if oldIdx == -1 {
				waterlog.Fatalf("Cannot find %s in old state!\n", name)
			}

			newPkg, newIdx := state.GetPackage(newState, name)
			if newIdx == -1 {
				waterlog.Fatalf("Cannot find %s in old state!\n", name)
			}

			changes = append(changes, state.Diff{
				Idx:       newIdx,
				OldIdx:    oldIdx,
				RelNum:    newPkg.Release,
				OldRelNum: oldPkg.Release,
				Ver:       newPkg.Version,
				OldVer:    oldPkg.Version,
			})
		}
	} else {
		changes = state.Changed(&oldState, &newState)
	}

	for _, diff := range changes {
		pkg := newState.Packages()[diff.Idx]
		if diff.IsSame() {
			waterlog.Warnf("Package %s hasn't changed, skipping...\n", pkg.Source)
		} else if diff.IsNewRel() {
			bumped = append(bumped, pkg)
			bset[diff.Idx] = true
		} else if diff.IsSameRel() && !diff.IsSame() {
			bad = append(bad, pkg)
		} else if diff.IsDowngrade() {
			outdated = append(outdated, pkg)
		}
	}

	// force, _ := cmd.Flags().GetBool("force")
	// dryRun, _ := cmd.Flags().GetBool("dry-run")
	// prePush, _ := cmd.Flags().GetBool("push")

	//if len(bad) != 0 {
	//	waterlog.Warnf("The following packages have the same release number but different version:")
	//	for _, pkg := range bad {
	//		waterlog.Printf(" %s", pkg.Name)
	//	}
	//	waterlog.Println()
	//	if !force {
	//		os.Exit(1)
	//	}
	//}

	//if len(outdated) != 0 {
	//	waterlog.Warnf("The following packages have older release numbers:")
	//	for _, pkg := range outdated {
	//		waterlog.Printf(" %s", pkg.Name)
	//	}
	//	waterlog.Println()
	//}

	//if len(bumped) == 0 {
	//	waterlog.Infoln("No packages to update. Exiting...")
	//	return
	//}

	//// Check that the dependencies of every package already exist
	//var unresolved []common.Package
	//for _, pkg := range bumped {
	//	// TODO: we should probably just be able to call pkg.Resolved?
	//	if len(pkg.Resolve(newState.NameToSrcIdx(), newState.Packages())) > 0 {
	//		unresolved = append(unresolved, pkg)
	//	}
	//}
	//if len(unresolved) != 0 {
	//	// waterlog.Errorf("The following packages have nonexistent build dependencies:")
	//	waterlog.Errorln("The following packages have nonexistent build dependencies:")
	//	for _, pkg := range unresolved {
	//		// waterlog.Printf(" %s", pkg.Name)
	//		waterlog.Errorf("%s:", pkg.Name)
	//		for _, dep := range pkg.BuildDeps {
	//			if _, ok := newState.NameToSrcIdx()[dep]; !ok {
	//				waterlog.Printf(" %s", dep)
	//			}
	//		}
	//		waterlog.Println()
	//	}

	//	// waterlog.Println()
	//	if !force {
	//		os.Exit(1)
	//	}
	//}

	//waterlog.Goodf("The following packages will be updated:")
	//for _, pkg := range bumped {
	//	waterlog.Printf(" %s", pkg.Name)
	//}
	//waterlog.Println()

	//depGraph := newState.DepGraph()
	//waterlog.Goodln("Successfully generated dependency graph!")

	//lifted := utils.LiftGraph(depGraph, func(i int) bool { return bset[i] })
	//if err != nil {
	//	waterlog.Fatalf("Failed to lift updated packages from dependency graph: %s\n", err)
	//}
	//waterlog.Goodln("Successfully isolated packages to update!")

	//res, ok := utils.TieredTopSort(lifted)
	//order := utils.Filter(utils.Flatten(res), func(i int) bool { return bset[i] })
	//if !ok {
	//	cycles := utils.Filter(graph.StrongComponents(lifted), func(i []int) bool { return len(i) > 1 })
	//	cycleIdx := 0

	//	for _, cycle := range cycles {
	//		if len(cycle) <= 1 {
	//			continue
	//		}

	//		waterlog.Warnf("Cycle %d:", cycleIdx+1)
	//		cycleIdx++

	//		for _, nodeIdx := range cycle {
	//			waterlog.Printf(" %s", newState.Packages()[nodeIdx].Name)
	//		}
	//		waterlog.Println()
	//	}
	//	waterlog.Fatalln("Failed to compute build order: lifted graph has cycles! Run `autobuild query` on the cycle to get more info.")
	//}

	//waterlog.Goodln("Here's the build order:")
	//for _, idx := range order {
	//	waterlog.Println(newState.Packages()[idx].Name)
	//}

	//if dryRun {
	//	return
	//}

	//red := color.New(color.FgRed).SprintFunc()
	//green := color.New(color.FgGreen).SprintFunc()
	//for _, idx := range order {
	//	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	//	defer s.Stop()
	//	pkg := newState.Packages()[idx]

	//	s.Prefix = " "
	//	s.Suffix = fmt.Sprintf("  Publishing %s", pkg.Name)
	//	s.Color("white")
	//	s.Restart()

	//	job, err := push.Publish(pkg, prePush)
	//	jobid := job.ID
	//	if err != nil {
	//		s.FinalMSG = fmt.Sprintf("%s failed to publish %s: %s", red("[x]"), pkg.Name, err)
	//		s.Stop()
	//		os.Exit(1)
	//	}

	//	s.Color("yellow")
	//	s.Suffix = fmt.Sprintf("  Package %s (%d) is waiting to be claimed", pkg.Name, jobid)
	//	s.Restart()
	//	for job.Status == "UNCLAIMED" {
	//		job, err = push.Query(jobid)
	//		time.Sleep(1 * time.Second)
	//	}

	//	s.Suffix = fmt.Sprintf("  Package %s (%d) is claimed, waiting to be built", pkg.Name, jobid)
	//	for job.Status == "CLAIMED" {
	//		job, err = push.Query(jobid)
	//		time.Sleep(1 * time.Second)
	//	}

	//	if job.Status == "BUILDING" {
	//		s.Color("green")
	//		s.Suffix = fmt.Sprintf("  Package %s (%d) is building", pkg.Name, jobid)
	//		s.Restart()
	//	}
	//	for job.Status == "BUILDING" {
	//		job, err = push.Query(jobid)
	//		time.Sleep(15 * time.Second)
	//	}

	//	if job.Status == "OK" {
	//		s.FinalMSG = fmt.Sprintf("%s %s (%d) built successfully!\n", green("[✓]"), pkg.Name, jobid)
	//		s.Stop()
	//	} else {
	//		if job.Status == "FAILED" {
	//			s.FinalMSG = fmt.Sprintf("%s %s (%d) failed to build\n", red("[x]"), pkg.Name, jobid)
	//		} else {
	//			s.FinalMSG = fmt.Sprintf("%s %s (%d) has unknown status %s\n", red("[x]"), pkg.Name, jobid, job.Status)
	//		}
	//		s.Stop()
	//		os.Exit(1)
	//	}
	//}
}
