// SPDX-FileCopyrightText: Copyright © 2020-2023 Serpent OS Developers
//
// SPDX-License-Identifier: MPL-2.0

package stone

import (
	"errors"
	"fmt"
	"path/filepath"
	_ "regexp"
	"slices"

	_ "github.com/DataDrake/waterlog"
	"github.com/GZGavinZhao/autobuild/common"
	"github.com/GZGavinZhao/autobuild/config"
	"github.com/GZGavinZhao/autobuild/utils"
)

func ParsePackage(path string) (cpkg common.Package, err error) {
	manifestPath := filepath.Join(path, "manifest.x86_64.bin")

	stonePath := filepath.Join(path, "stone.yaml")
	if !utils.PathExists(stonePath) {
		return
	}

	spkg, err := Load(stonePath)
	if err != nil {
		return
	}

	if utils.PathExists(manifestPath) {
		if cpkg, err = ParseManifest(manifestPath); err != nil {
			return
		}

		if cpkg.Name != spkg.Name {
			err = fmt.Errorf("Manifest and stone.yml name mismatch: manifest has %s, stone.yml has %s", cpkg.Name, spkg.Name)
			return
		}
	} else {
		// TODO: the below is much more incomplete than the .bin parsing.
		// We may need to fallback to `.yml` parsing in the case of inspecting
		// build order before a package is build.
		cpkg = common.Package{
			Path:      stonePath,
			Name:      spkg.Name,
			Version:   spkg.Version,
			Release:   spkg.Release,
			BuildDeps: append(spkg.BuildDeps, spkg.CheckDeps...),
			Synced:    false,
		}

		cpkg.BuildDeps = append(cpkg.BuildDeps, spkg.CollectRunDeps()...)

		if spkg.Toolchain == "clang" {
			cpkg.BuildDeps = append(cpkg.BuildDeps, "llvm-clang-devel")
		} else if spkg.Toolchain == "gnu" {
			cpkg.BuildDeps = append(cpkg.BuildDeps, "gcc-devel")
		}
	}

	for _, cfgBase := range []string{"autobuild.yaml", "autobuild.yml"} {
		cfgPath := filepath.Join(path, cfgBase)

		if !utils.PathExists(cfgPath) {
			continue
		}

		abConfig, err := config.Load(cfgPath)
		if err != nil {
			err = errors.New(fmt.Sprintf("Failed to load autobuild config file for %s: %s", path, err))
			return cpkg, err
		}

		// ignoreRegexes := []regexp.Regexp{}
		for _, ignore := range abConfig.Solver.Ignore {
			cpkg.Ignores = append(cpkg.Ignores, ignore)
			// ignoreRegexes = append(ignoreRegexes, *regexp.MustCompile(ignore))
		}
		// cpkg.BuildDeps = utils.Filter(cpkg.BuildDeps, func(dep string) bool {
		// 	for _, regex := range ignoreRegexes {
		// 		if regex.FindString(dep) != "" {
		// 			waterlog.Debugf("Dropping builddep %s from %s due to ignore regex %s\n", dep, cpkg.Name, regex.String())
		// 			return false
		// 		}
		// 	}
		// 	return true
		// })

		break
	}

	slices.Sort(cpkg.BuildDeps)
	slices.Sort(cpkg.Provides)
	// waterlog.Debugf("%s: %q\n", cpkg.Name, cpkg.BuildDeps)
	return
}
