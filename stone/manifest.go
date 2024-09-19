package stone

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"

	"github.com/GZGavinZhao/autobuild/common"
	"github.com/GZGavinZhao/autobuild/config"
	"github.com/GZGavinZhao/autobuild/utils"
	"github.com/serpent-os/libstone-go"
	"github.com/serpent-os/libstone-go/stone1"
)

var (
	badProviders = [...]string{"soname(libz.so.1(x86))", "soname(libclang.so.15(x86_64))", "soname(libc.so.6(386))"}
)

func ParseManifest(path string, abconfig config.AutobuildConfig) (cpkgs []common.Package, err error) {
	// Prepare the `cpkg`-s that result from splitting.
	// `cpkgs[0]` is the default cpkg to read info into.
	// Starting from index 1 are the `cpkg` that are
	// `split`-ted
	cpkgs = append(cpkgs, common.Package{
		Ignores: abconfig.Solver.Ignore,
	})
	nameToIdx := make(map[string]int)

	for _, split := range abconfig.Solver.Split {
		nameToIdx[split] = len(cpkgs)
		cpkgs = append(cpkgs, common.Package{
			Ignores: abconfig.Solver.Ignore,
		})
	}

	// Open the manifest and read from it.
	file, err := os.Open(path)
	if err != nil {
		err = fmt.Errorf("Failed to open manifest %s, reason: %w", path, err)
		return
	}
	defer file.Close()

	genericPrelude, err := libstone.ReadPrelude(file)
	if err != nil {
		return
	}

	prelude, err := stone1.NewPrelude(genericPrelude)
	if err != nil {
		return
	}

	cache, err := os.CreateTemp("", "")
	if err != nil {
		err = fmt.Errorf("Failed to create temp for parsing stone file: %w", err)
		return
	}
	defer os.Remove(cache.Name())

	rdr := stone1.NewReader(prelude, file, cache)

	// Each iteraton is a sub-package.
	for rdr.NextPayload() {
		if rdr.Header.Kind != stone1.Meta {
			continue
		}

		// WARNING:
		// We rely on the convention that the first field in meta is Name, so we
		// know which splitted package this should belong to.
		var cpkg *common.Package
		for rdr.NextRecord() {
			switch record := rdr.Record.(type) {
			case *stone1.MetaRecord:
				switch record.Tag {
				case stone1.SourceID:
					cpkg.Source = record.Field.String()
				case stone1.Version:
					cpkg.Version = record.Field.String()
				case stone1.Release:
					cpkg.Release = int(record.Field.Value.(uint64))
				case stone1.Depends:
					dep := record.Field.String()
					if !slices.Contains(badProviders[:], dep) {
						if tos, ok := abconfig.Solver.Move[dep]; ok {
							for _, to := range tos {
								cpkgs[nameToIdx[to]].BuildDeps = append(cpkgs[nameToIdx[to]].BuildDeps, dep)
							}
						} else {
							cpkg.BuildDeps = append(cpkg.BuildDeps, dep)
						}
					}
				case stone1.Provides:
					cpkg.Provides = append(cpkg.Provides, record.Field.String())
				case stone1.Name:
					pkgName := record.Field.String()
					cpkg = &cpkgs[nameToIdx[pkgName]]

					cpkg.Names = append(cpkg.Names, pkgName)
					cpkg.Provides = append(cpkg.Provides, fmt.Sprintf("name(%s)", pkgName))
					// Implcitily assume that `X-dbginfo` is provieded by
					// package `X`.
					if !strings.HasPrefix(pkgName, "-dbginfo") {
						cpkg.Provides = append(cpkg.Provides, fmt.Sprintf("name(%s-dbginfo)", pkgName))
					}
				}
			default:
			}
		}

		if rdr.Err != nil {
			err = fmt.Errorf("stone reader failure: %w", rdr.Err)
			return
		}
	}

	for idx, cpkg := range cpkgs {
		// Check if splitted packages are actually set when iterating through
		// the subpackages.
		if len(cpkg.Source) == 0 || len(cpkg.Version) == 0 || len(cpkg.Names) == 0 {
			if idx == 0 {
				err = errors.New("default package not set properly")
				return
			} else {
				slog.Warn("Split seems to be unnecessary", "split", abconfig.Solver.Split[idx-1], "path", path)
			}
		}

		// Sort dependencies for reproducibility
		slices.Sort(cpkgs[idx].BuildDeps)
		// fmt.Println(cpkgs[idx].BuildDeps)
		cpkgs[idx].BuildDeps = utils.Uniq2(cpkgs[idx].BuildDeps)
		// fmt.Println(cpkgs[idx].BuildDeps)

		slices.Sort(cpkgs[idx].Provides)
		cpkgs[idx].Provides = utils.Uniq2(cpkgs[idx].Provides)

		slices.Sort(cpkgs[idx].Ignores)
		cpkgs[idx].Ignores = utils.Uniq2(cpkgs[idx].Ignores)
	}

	return
}
