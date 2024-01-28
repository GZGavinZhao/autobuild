// SPDX-FileCopyrightText: Copyright © 2020-2023 Serpent OS Developers
//
// SPDX-License-Identifier: MPL-2.0

package stone

import (
	"path/filepath"

	"github.com/GZGavinZhao/autobuild/common"
	"github.com/GZGavinZhao/autobuild/utils"
)

func ParsePackage(path string) (cpkg common.Package, err error) {
	stonePath := filepath.Join(path, "stone.yml")
	if !utils.PathExists(stonePath) {
		return
	}

	spkg, err := Load(stonePath)
	if err != nil {
		return
	}

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

	return
}
