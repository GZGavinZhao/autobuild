// spdx-filecopyrighttext: copyright © 2020-2023 serpent os developers
//
// spdx-license-identifier: mpl-2.0

package common

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	_ "github.com/DataDrake/waterlog"
	"github.com/GZGavinZhao/autobuild/config"
	"github.com/GZGavinZhao/autobuild/utils"
	"github.com/GZGavinZhao/autobuild/ypkg"
	"github.com/getsolus/libeopkg/index"
	"github.com/getsolus/libeopkg/pspec"
	"github.com/jwalton/gchalk"
	"gopkg.in/yaml.v3"
)

var (
	// pcre = regexp.MustCompile(`/usr/(lib|lib64|lib32|share)/[^/]+\.pc`)
	pcre    = regexp.MustCompile(`/usr/(lib|lib64|lib32|share)/.+\.pc$`)
	oldpcre = regexp.MustCompile(`/usr/(lib|lib64|lib32|share)/.+\.pc`)
)

type Package struct {
	Path      string
	Names     []string
	Source    string
	Version   string
	Root      string
	Release   int
	Provides  []string
	BuildDeps []string
	Ignores   []string
	Resolved  bool
	Built     bool
	Synced    bool
}

// // Merge the info from `other` to itself. Prefer `other` if different.
// func (p *Package) Merge(o Package) {
// 	if len(o.Name) > 0 {
// 		p.Name = o.Name
// 	}
// 	if len(o.Version) > 0 {
// 		p.Version = o.Version
// 	}
// 	if len()
// }

// Show is the toString method for a package.
//
// When `sub` is true, show the subpackages that this package (node) represents.
//
// When `color` is true, show the subpackages in gray color for easier viewing.
// Obviously this has no effects when `sub` is false.
func (p *Package) Show(sub bool, color bool) string {
	if !sub {
		return p.Source
	} else if color {
		return p.Source + gchalk.Gray(fmt.Sprintf("{%s}", strings.Join(p.Names, ", ")))
	} else {
		return fmt.Sprintf("%s{%s}", p.Source, strings.Join(p.Names, ", "))
	}
}

func (p *Package) Resolve(nameToSrcIdx map[string]int, pkgs []Package) (res []string) {
	// if !p.Resolved {
	// 	p.Resolved = true
	//
	// 	for idx, dep := range p.BuildDeps {
	// 		srcIdx, ok := nameToSrcIdx[dep]
	//
	// 		if !ok {
	// 			p.Resolved = false
	// 			res = append(res, dep)
	// 		} else {
	// 			p.BuildDeps[idx] = pkgs[srcIdx].Name
	// 		}
	// 	}
	//
	// 	slices.Sort(p.BuildDeps)
	// 	p.BuildDeps = utils.Uniq(p.BuildDeps)
	//
	// 	ignoreRegexes := []regexp.Regexp{}
	// 	for _, ignore := range p.Ignores {
	// 		ignoreRegexes = append(ignoreRegexes, *regexp.MustCompile(ignore))
	// 	}
	// 	p.BuildDeps = utils.Filter(p.BuildDeps, func(dep string) bool {
	// 		for _, regex := range ignoreRegexes {
	// 			if regex.FindString(dep) == dep {
	// 				waterlog.Debugf("Package.Resolve: Dropping builddep %s from %s due to ignore regex %s\n", dep, p.Name, regex.String())
	// 				return false
	// 			}
	// 		}
	// 		return true
	// 	})
	// }

	return
}

// ParsePackage parses a source package that is within the given `dir`
// directory. In other words, a `package.yml` file must be located at
// `dir/package.yml`.
func ParsePackage(dir string) (pkgs []Package, err error) {
	// Check if the given directory contains a package definition
	pkgFile := filepath.Join(dir, "package.yml")
	pspecFile := filepath.Join(dir, "pspec_x86_64.xml")
	cfgFile := filepath.Join(dir, "autobuild.yml")

	ypkgYml, err := ypkg.Load(pkgFile)
	if err != nil {
		err = errors.New(fmt.Sprintf("Failed to load package.yml file for %s: %s", dir, err))
		return
	}

	pkgs = append(pkgs, Package{
		Path:      dir,
		Source:    ypkgYml.Name,
		Names:     []string{ypkgYml.Name},
		Version:   ypkgYml.Version,
		Release:   ypkgYml.Release,
		BuildDeps: ypkgYml.BuildDeps,
		Synced:    false},
	)
	pkg := &pkgs[0]

	// Combine the rundeps of all subpackages into a single list
	// Note to self: this website can inspect yaml ast nodes:
	// https://astexplorer.net/, might be useful when debugging
	rundeps := ypkgYml.RunDeps
	if rundeps.Kind == yaml.SequenceNode {
		for _, children := range rundeps.Content {
			if children.Kind == yaml.ScalarNode {
				pkg.BuildDeps = append(pkg.BuildDeps, children.Value)
			} else if children.Kind == yaml.MappingNode {
				for _, subpkg := range children.Content {
					for _, rundep := range subpkg.Content {
						if rundep.Kind != yaml.ScalarNode {
							continue
						}

						pkg.BuildDeps = append(pkg.BuildDeps, rundep.Value)
					}
				}
			}
		}
	} else {
		err = errors.New(fmt.Sprintf("%s has unknown \"rundeps\" field kind: %s", dir, rundeps.Value))
	}

	if ypkgYml.Clang {
		pkg.BuildDeps = append(pkg.BuildDeps, "llvm-clang-devel")
	}

	if !utils.PathExists(pspecFile) {
		return
	}

	pspecXml, err := pspec.Load(pspecFile)
	if err != nil {
		err = errors.New(fmt.Sprintf("Failed to load pspec_x86_64.xml for %s: %s", dir, err))
		return
	}
	for _, subPkg := range pspecXml.Packages {
		pkg.Provides = append(pkg.Provides, subPkg.Name)

		for _, pcProvide := range getPcProvides(&subPkg) {
			pkg.Provides = append(pkg.Provides, pcProvide)
		}
	}

	if !utils.PathExists(cfgFile) {
		return
	}

	abConfig, err := config.Load(cfgFile)
	if err != nil {
		err = errors.New(fmt.Sprintf("Failed to load autobuild config file for %s: %s", dir, err))
	}

	// ignoreRegexes := []regexp.Regexp{}
	for _, ignore := range abConfig.Solver.Ignore {
		// ignoreRegexes = append(ignoreRegexes, *regexp.MustCompile(ignore))
		pkg.Ignores = append(pkg.Ignores, ignore)
	}
	// pkg.BuildDeps = utils.Filter(pkg.BuildDeps, func(dep string) bool {
	// 	for _, regex := range ignoreRegexes {
	// 		if regex.FindString(dep) == dep {
	// 			waterlog.Debugf("Dropping builddep %s from %s due to ignore regex %s\n", dep, pkg.Name, regex.String())
	// 			return false
	// 		}
	// 	}
	// 	return true
	// })

	slices.Sort(pkg.BuildDeps)
	slices.Sort(pkg.Provides)
	slices.Sort(pkg.Ignores)

	return
}

func getPcProvides(pkg *pspec.Package) []string {
	var provides []string

	for _, file := range pkg.Files {
		match := pcre.FindString(file.Value)
		pcFile := filepath.Base(match)
		if pcFile == "." || pcFile == "*.pc" {
			continue
		}

		splitted := strings.Split(match, "/")
		if len(splitted) > 5 {
			continue
		}

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

func ParseIndexPackage(ipkg index.Package) (pkg Package, err error) {
	pkg.Source = ipkg.Source.Name
	pkg.Names = append(pkg.Names, ipkg.Name)
	pkg.Provides = append(pkg.Provides, fmt.Sprintf("name(%s)", ipkg.Name))

	latest := ipkg.History[0]
	pkg.Release = latest.Release
	pkg.Version = latest.Version

	// TODO: we intentionally don't add anything besides version information,
	// because we don't need to. Right now, we only use binary index for diffing
	// against another source/binary index to see what packages have changed.

	return
}
