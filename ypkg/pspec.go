package ypkg

import (
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/getsolus/libeopkg/pspec"
)

var (
	// pcre = regexp.MustCompile(`/usr/(lib|lib64|lib32|share)/[^/]+\.pc`)
	pcre    = regexp.MustCompile(`/usr/(lib|lib64|lib32|share)/.+\.pc$`)
	oldpcre = regexp.MustCompile(`/usr/(lib|lib64|lib32|share)/.+\.pc`)
)

func GetPcProvidesFromPspecPkg(pkg *pspec.Package) []string {
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
