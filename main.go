package main

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
)

var systems, archs StringSlice

type Config struct {
	OutputDir      string
	Name           string
	NameTemplate   string
	BundleTemplate string
	BuildArgs      []string
	Clean          bool

	Aliases StringSlice
	Include StringSlice
	Exclude StringSlice
}

var allArchs = []string{
	"386",
	"amd64",
	// "amd64p32",
	"arm",
	// "armbe",
	"arm64",
	// "arm64be",
	"ppc64",
	"ppc64le",
	"mips",
	"mipsle",
	"mips64",
	"mips64le",
	// "mips64p32",
	// "mips64p32le",
	// "ppc",
	// "riscv",
	"riscv64",
	// "s390",
	"s390x",
	// "sparc",
}

var uncommonArchs = []string{}

var ioArchs = []string{}

var pairDefinitions = SMap{
	"windows":   {"386", "amd64", "arm", "arm64"},
	"linux":     allArchs,
	"darwin":    {"amd64", "arm64"},
	"android":   allArchs,
	"js":        {"wasm"},
	"aix":       allArchs,
	"dragonfly": allArchs,
	"freebsd":   allArchs,
	"hurd":      allArchs,
	"illumos":   allArchs,
	"netbsd":    allArchs,
	"openbsd":   allArchs,
	"plan9":     allArchs,
	"solaris":   allArchs,
	"zos":       allArchs,
}

var aliases = map[string]SMap{
	"all":    pairDefinitions.Copy(),
	"common": pairDefinitions.OnlyKeys("windows", "linux", "darwin"),
	"mobile": pairDefinitions.OnlyKeys("android"),
	"web":    pairDefinitions.OnlyKeys("js"),
	"unix":   pairDefinitions.OnlyKeys("linux", "darwin", "aix", "dragonfly", "freebsd", "hurd", "illumos", "netbsd", "openbsd", "plan9", "solaris", "zos"),
}

func renderString(tmpl *template.Template, data interface{}) (string, error) {
	buf := bytes.NewBufferString("")
	if err := tmpl.Execute(buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func move(loc string, sys string, arch string, config Config) (err error) {
	nameTmpl, err := template.New("name").Parse(config.NameTemplate)
	if err != nil {
		return err
	}
	bundleTmpl, err := template.New("bundle").Parse(config.BundleTemplate)
	if err != nil {
		return err
	}
	ext, cext := "", ".zip"
	if sys == "windows" {
		ext = ".exe"
	}
	data := map[string]string{"NAME": config.Name, "GOOS": sys, "GOARCH": arch, "EXT": ext, "ZIP": cext}
	name, err := renderString(nameTmpl, data)
	if err != nil {
		return err
	}
	bundleName, err := renderString(bundleTmpl, data)
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

func clean(dir string) error {
	names, err := filepath.Glob(dir + "/*.zip")
	if err != nil {
		return err
	}
	if len(names) != 0 {
		fmt.Printf("cleaning %d files from %s", len(names), dir)
	}
	for _, p := range names {
		fmt.Println("removing", p)
		if err = os.Remove(p); err != nil {
			return err
		}
	}
	return nil
}

func getBuildTargets(config Config) (res SMap, err error) {
	if len(config.Aliases) == 0 {
		config.Aliases = append(config.Aliases, "common")
	}
	res = make(SMap, len(config.Aliases))
	for _, alias := range config.Aliases {
		if vals, ok := aliases[alias]; ok {
			res = res.Union(vals)
		} else {
			err = fmt.Errorf("unknown alias: %s", alias)
			return
		}
	}
	return
}

func getName(config Config) (name string, err error) {
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

func build(config Config) (err error) {

	config.OutputDir = filepath.Clean(config.OutputDir)
	if config.Clean {
		if err = clean(config.OutputDir); err != nil {
			return
		}
	}

	if config.Name, err = getName(config); err != nil {
		return
	}

	buildTargets, err := getBuildTargets(config)
	if err != nil {
		return
	}

	args := []string{"build"}
	for sys, archs := range buildTargets {
		for _, arch := range archs {
			outPath := filepath.Join(config.OutputDir, config.Name)
			cmdArgs := append(args, "-o", outPath)
			cmdArgs = append(cmdArgs, config.BuildArgs...)
			cmd := exec.CommandContext(context.Background(), "go", cmdArgs...)
			cmd.Env = os.Environ()
			cmd.Env = append(cmd.Env, []string{fmt.Sprintf("GOOS=%s", sys), fmt.Sprintf("GOARCH=%s", arch)}...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			fmt.Printf("building %s\\%s\n", sys, arch)
			if err = cmd.Run(); err != nil {
				fmt.Println(err)
				// return err
			}
			if err == nil {
				if err = move(outPath, sys, arch, config); err != nil {
					return
				}
				if err = os.Remove(outPath); err != nil {
					return
				}
			}
		}
	}
	return nil
}

func main() {
	config := Config{}
	flag.StringVar(&config.OutputDir, "o", "", "output directory")
	flag.StringVar(&config.Name, "name", "", "executable name")
	flag.StringVar(&config.NameTemplate, "name-template", "{{.NAME}}{{.EXT}}", "template to use for each file")
	flag.StringVar(&config.BundleTemplate, "bundle-template", "{{.NAME}}_{{.GOOS}}_{{.GOARCH}}{{.ZIP}}", "template to use for each bundle")
	flag.BoolVar(&config.Clean, "clean", false, "clean the output directory before building")
	flag.Var(&config.Aliases, "a", "the primary set of combinations to include in the release (default: common)")
	flag.Var(&config.Exclude, "e", "values to exclude. can be either os or archs")
	flag.Var(&config.Include, "i", "values to include apart from those defined in aliases")

	// Allow passing additional args separated by "--"
	os.Args, config.BuildArgs, _ = StringSliceCut(os.Args, "--")
	flag.Parse()

	if err := build(config); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
