// SPDX-FileCopyrightText: Copyright © 2020-2023 Serpent OS Developers
//
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"github.com/GZGavinZhao/autobuild/cmd"
	_ "github.com/GZGavinZhao/autobuild/cmd"
	"github.com/GZGavinZhao/autobuild/common"
	_ "github.com/getsolus/libeopkg/index"
)

var (
	badPackages  = [...]string{"haskell-http-client-tls"}
	srcPkgs      = []common.Package{}
	nameToSrcIdx = make(map[string]int)
)

func main() {
	cmd.Execute()
}
