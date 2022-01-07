package lib

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type BuildConfig struct {
	OutputDir      string
	Name           string
	NameTemplate   string
	BundleTemplate string
	BuildArgs      []string
	Clean          bool
	Dry            bool
	ShowTargets    bool

	Aliases         StringSlice
	DistributionSet DistributionSet
}

func BundleFile(loc string, dist Distribution, config BuildConfig) (err error) {
	nameTmpl, err := template.New("name").Parse(config.NameTemplate)
	if err != nil {
		return err
	}
	bundleTmpl, err := template.New("bundle").Parse(config.BundleTemplate)
	if err != nil {
		return err
	}
	ext, cext := "", ".zip"
	if dist.GOOS == "windows" {
		ext = ".exe"
	}
	data := map[string]string{"NAME": config.Name, "GOOS": dist.GOOS, "GOARCH": dist.GOARCH, "EXT": ext, "ZIP": cext}
	name, err := RenderString(nameTmpl, data)
	if err != nil {
		return err
	}
	bundleName, err := RenderString(bundleTmpl, data)
	if err != nil {
		return err
	}

	finalPath := filepath.Join(config.OutputDir, bundleName)
	outf, err := os.Create(finalPath)
	if err != nil {
		return err
	}
	defer outf.Close()
	inF, err := os.Open(loc)
	if err != nil {
		return err
	}
	defer inF.Close()
	writer := zip.NewWriter(outf)
	defer writer.Close()
	outz, err := writer.Create(name)
	if err != nil {
		return err
	}
	_, err = io.Copy(outz, inF)
	return
}

// Compute aliases based on the available distributions
func GetAliases(availableDistributions DistributionSet) (aliases map[string]DistributionSet) {
	aliases = map[string]DistributionSet{
		"all":     availableDistributions,
		"mobile":  availableDistributions.Only("android", "ios"),
		"web":     availableDistributions.Only("js"),
		"apple":   availableDistributions.Only("darwin", "ios"),
		"desktop": availableDistributions.Only("windows", "darwin", "linux"),
		"unix":    availableDistributions.Only("linux", "aix", "dragonfly", "freebsd", "illumos", "netbsd", "openbsd", "plan9", "solaris"),
	}

	for _, d := range availableDistributions {
		// Aliases for all operating systems
		aliases[d.GOOS] = availableDistributions.Only(d.GOOS)
		// Aliases for all architectures
		aliases[d.GOARCH] = availableDistributions.OnlyArch(d.GOARCH)
		// Aliases for all first-class support
		if d.FirstClass {
			aliases["first-class"] = append(aliases["first-class"], d)
		} else {
			aliases["second-class"] = append(aliases["second-class"], d)
		}
		// Aliases for cgo supported
		if d.CgoSupported {
			aliases["cgo"] = append(aliases["cgo"], d)
		}
	}

	return
}

func GetBuildTargets(config BuildConfig) (res DistributionSet, err error) {
	availableDistributions, err := GetAllDistributions()
	if err != nil {
		return
	}

	aliases := GetAliases(availableDistributions)
	if len(config.Aliases) == 0 {
		config.Aliases = append(config.Aliases, "first-class")
	}

	for _, alias := range config.Aliases {
		isDiff := strings.HasPrefix(alias, "-")
		if isDiff {
			alias = alias[1:]
		}
		if vals, ok := aliases[alias]; ok {
			if isDiff {
				res = res.Difference(vals)
			} else {
				res = res.Union(vals)
			}
		} else {
			err = fmt.Errorf("unknown alias: %s", alias)
			return
		}
	}
	return
}

func GetModName(config BuildConfig) (name string, err error) {
	if config.Name != "" {
		return config.Name, nil
	}
	mod, err := parseMod("go.mod")
	if errors.Is(err, os.ErrNotExist) {
		return "", errors.New("must define the executable name or have a go.mod file present")
	} else if err != nil {
		return "", err
	}
	return mod.Name, err
}
