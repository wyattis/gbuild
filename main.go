package main

import (
	"archive/zip"
	"context"
	"embed"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed docs
var docs embed.FS

type Config struct {
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

func bundleFile(loc string, dist Distribution, config Config) (err error) {
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

// Compute aliases based on the available distributions
func getAliases(availableDistributions DistributionSet) (aliases map[string]DistributionSet) {
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

func getBuildTargets(config Config) (res DistributionSet, err error) {
	availableDistributions, err := getAllDistributions()
	if err != nil {
		return
	}

	aliases := getAliases(availableDistributions)
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

func runBuild(set *flag.FlagSet, config Config) (err error) {
	config.Aliases = set.Args()
	config.DistributionSet, err = getBuildTargets(config)
	if err != nil {
		return
	}
	if config.Dry {
		fmt.Println("** dry run **")
	}
	fmt.Printf("preparing to build %d packages\n", len(config.DistributionSet))

	config.OutputDir = filepath.Clean(config.OutputDir)
	if !config.Dry && config.Clean {
		if err = cleanDirGlob(config.OutputDir, "*.zip"); err != nil {
			return
		}
	}

	if config.Name, err = getName(config); err != nil {
		return
	}

	args := []string{"build"}
	for _, dist := range config.DistributionSet {
		outPath := filepath.Join(config.OutputDir, config.Name)
		cmdArgs := append(args, "-o", outPath)
		cmdArgs = append(cmdArgs, config.BuildArgs...)
		cmd := exec.CommandContext(context.Background(), "go", cmdArgs...)
		cmd.Env = os.Environ()
		cmd.Env = append(cmd.Env, []string{fmt.Sprintf("GOOS=%s", dist.GOOS), fmt.Sprintf("GOARCH=%s", dist.GOARCH)}...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		fmt.Printf("building %s\\%s\n", dist.GOOS, dist.GOARCH)
		if config.Dry {
			continue
		}
		if err = cmd.Run(); err != nil {
			fmt.Println(err)
			// return err
		}
		if err == nil {
			if err = bundleFile(outPath, dist, config); err != nil {
				return
			}
			if err = os.Remove(outPath); err != nil {
				return
			}
		}
	}
	return nil
}

func runAliases(set *flag.FlagSet, config Config) (err error) {
	distributions, err := getAllDistributions()
	if err != nil {
		return
	}
	aliases := getAliases(distributions)
	keys := flag.Args()
	if len(keys) == 0 {
		for key := range aliases {
			keys = append(keys, key)
		}
	}

	if config.ShowTargets {
		for _, key := range keys {
			fmt.Printf("%s:\n", key)
			for _, val := range aliases[key] {
				fmt.Printf("\t%s\n", val.String())
			}
		}
	} else {
		for _, key := range keys {
			fmt.Println(key)
		}
	}

	return
}

func setFlagUsage(flagSet *flag.FlagSet) {
	flagSet.Usage = func() {
		out := flagSet.Output()
		d, err := docs.ReadFile("docs/" + flagSet.Name() + ".md")
		if err != nil {
			fmt.Fprintln(out, err)
		}
		out.Write(d)
		flagSet.PrintDefaults()
	}
}

type Cmd struct {
	FlagSet *flag.FlagSet
	Exec    func(*flag.FlagSet, Config) error
}

func run() (err error) {
	config := Config{}

	flagBuild := flag.NewFlagSet("build", flag.ExitOnError)
	flagBuild.StringVar(&config.OutputDir, "o", "release", "output directory")
	flagBuild.StringVar(&config.Name, "name", "", "executable name")
	flagBuild.StringVar(&config.NameTemplate, "name-template", "{{.NAME}}{{.EXT}}", "template to use for each file")
	flagBuild.StringVar(&config.BundleTemplate, "bundle-template", "{{.NAME}}_{{.GOOS}}_{{.GOARCH}}{{.ZIP}}", "template to use for each bundle")
	flagBuild.BoolVar(&config.Clean, "clean", false, "clean the output directory before building")
	flagBuild.BoolVar(&config.Dry, "dry", false, "run without actually building anything")

	flagAliases := flag.NewFlagSet("aliases", flag.ExitOnError)
	flagAliases.BoolVar(&config.ShowTargets, "targets", false, "include a list of targets for each alias")

	cmds := []Cmd{
		{flagBuild, runBuild},
		{flagAliases, runAliases},
	}

	// Allow passing additional args separated by "--"
	os.Args, config.BuildArgs, _ = StringSliceCut(os.Args, "--")
	flag.Parse()
	flag.CommandLine.SetOutput(os.Stdin)

	for _, cmd := range cmds {
		if flag.Arg(0) == cmd.FlagSet.Name() {
			if err = cmd.FlagSet.Parse(flag.Args()[1:]); err != nil {
				return
			}
			return cmd.Exec(cmd.FlagSet, config)
		}
	}

	fmt.Println("must supply a valid command")
	for _, cmd := range cmds {
		fmt.Fprintln(cmd.FlagSet.Output())
		d, err := docs.ReadFile("docs/" + cmd.FlagSet.Name() + ".md")
		if err == nil {
			cmd.FlagSet.Output().Write(d)
		}
		cmd.FlagSet.PrintDefaults()
	}
	return
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
