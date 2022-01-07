package lib

import (
	"archive/zip"
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
)

type BuildConfig struct {
	OutputDir      string
	Name           string
	GoVersion      string
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

	return enhanceDistributions(res, config)

}

func enhanceDistributions(d DistributionSet, config BuildConfig) (res DistributionSet, err error) {
	res = d
	// nameTmpl, err := template.New("name").Parse(config.NameTemplate)
	// if err != nil {
	// 	return
	// }
	bundleTmpl, err := template.New("bundle").Parse(config.BundleTemplate)
	if err != nil {
		return
	}

	for i := range res {
		ext, cext := "", ".zip"
		if res[i].GOOS == "windows" {
			ext = ".exe"
		}
		data := map[string]string{"NAME": config.Name, "GOOS": res[i].GOOS, "GOARCH": res[i].GOARCH, "EXT": ext, "ZIP": cext}
		// binName, err := RenderString(nameTmpl, data)
		// if err != nil {
		// 	return
		// }
		bundleName, err := RenderString(bundleTmpl, data)
		if err != nil {
			return res, err
		}

		finalPath := filepath.Join(config.OutputDir, bundleName)
		res[i].BuildPath = finalPath
	}
	return
}

type GoModule struct {
	Name      string
	GoVersion string
}

// Parse package name and go version from a go.mod file
func ParseMod(loc string) (mod GoModule, err error) {
	f, err := os.Open(loc)
	if err != nil {
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if err = scanner.Err(); err != nil {
			return
		}
		line := scanner.Text()
		if mod.Name == "" && strings.HasPrefix(line, "module") {
			_, name, found := StringCut(line, " ")
			if !found {
				err = fmt.Errorf("invalid module name in %s", loc)
				return
			}
			mod.Name = name
		}
		if mod.GoVersion == "" && strings.HasPrefix(line, "go") {
			_, version, found := StringCut(line, " ")
			if !found {
				err = fmt.Errorf("invalid go version in %s", loc)
				return
			}
			mod.GoVersion = version
		}
		if mod.GoVersion != "" && mod.Name != "" {
			return
		}
	}
	return
}

func ApplyModule(config *BuildConfig) (mod GoModule, err error) {
	config.GoVersion = runtime.Version()
	mod, err = ParseMod("go.mod")
	if err != nil {
		// It's not strictly necessary that the "go.mod" file should exist
		if errors.Is(err, os.ErrNotExist) {
			err = nil
		}
		return
	}
	if config.Name == "" {
		config.Name = mod.Name
	}
	config.GoVersion = mod.GoVersion
	return
}
