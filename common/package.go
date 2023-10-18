package common

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/GZGavinZhao/autobuild/utils"
	"github.com/GZGavinZhao/autobuild/ypkg"
	"github.com/getsolus/libeopkg/pspec"
)

var (
	pcre = regexp.MustCompile(`/usr/(lib|lib64|lib32|share)/pkgconfig/[^/]+\.pc`)
)

type Package struct {
	Path      string
	Name      string
	Version   string
	Release   int
	Provides  []string
	BuildDeps []string
	Resolved  bool
	Built     bool
	Synced    bool
}

func (p *Package) Resolve(nameToSrcIdx map[string]int) bool {
	if !p.Resolved {
		p.Resolved = true

		for _, dep := range p.BuildDeps {
			_, ok := nameToSrcIdx[dep]

			if !ok {
				p.Resolved = false
				break
			}
		}
	}

	return p.Resolved
}

// ParsePackage parses a source package that is within the given `dir`
// directory. In other words, a `package.yml` file must be located at
// `dir/package.yml`.
func ParsePackage(dir string) (pkg Package, err error) {
	// Check if the given directory contains a package definition
	pkgFile := filepath.Join(dir, "package.yml")
	pspecFile := filepath.Join(dir, "pspec_x86_64.xml")

	ypkgYml, err := ypkg.Load(pkgFile)
	if err != nil {
		err = errors.New(fmt.Sprintf("Failed to load package.yml file for %s: %s", dir, err))
		return
	}

	pkg = Package{
		Path:      dir,
		Name:      ypkgYml.Name,
		Version:   ypkgYml.Version,
		Release:   ypkgYml.Release,
		BuildDeps: ypkgYml.BuildDeps,
		Synced:    false,
	}

	if !utils.FileExists(pspecFile) {
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

	return
}

func getPcProvides(pkg *pspec.Package) []string {
	var provides []string

	for _, file := range pkg.Files {
		match := pcre.FindString(file.Value)
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
