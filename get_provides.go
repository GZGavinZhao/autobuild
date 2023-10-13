// SPDX-FileCopyrightText: Copyright © 2020-2023 Serpent OS Developers
//
// SPDX-License-Identifier: MPL-2.0

package main

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/getsolus/libeopkg/pspec"
	"golang.org/x/exp/slices"
)

var (
	re = regexp.MustCompile(`/usr/(lib|lib64|lib32|share)/pkgconfig/[^/]+\.pc`)
)

func getPcProvides(pkg *pspec.Package) []string {
	var provides []string

	for _, file := range pkg.Files {
		match := re.FindString(file.Value)
		pcFile := filepath.Base(match)
		if pcFile == "." {
			continue
		}

		splitted := strings.Split(match, "/")
		if slices.Contains(splitted, "lib32") {
			provides = append(provides, fmt.Sprintf("pkgconfig32(%s)", pcFile[:len(pcFile)-3]))
		} else if slices.Contains(splitted, "share") {
			provides = append(provides, fmt.Sprintf("pkgconfig(%s)", pcFile[:len(pcFile)-3]))
			provides = append(provides, fmt.Sprintf("pkgconfig32(%s)", pcFile[:len(pcFile)-3]))
		} else {
			provides = append(provides, fmt.Sprintf("pkgconfig(%s)", pcFile[:len(pcFile)-3]))
		}
	}

	return provides
}
