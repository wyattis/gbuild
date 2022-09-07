package cmd

import (
	"context"
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/wyattis/gbuild/lib"
)

//go:embed manual/build.md
var buildDescription string
var buildConfig lib.BuildConfig

var (
	ErrBuildName = errors.New("must define the executable name or have a go.mod file present")
)

var buildCommand = lib.Cmd{
	Name:             "build",
	ShortDescription: "Cross compile for multiple platforms using a set of aliases.",
	LongDescription:  buildDescription,
	Init: func(set *flag.FlagSet) error {
		set.BoolVar(&buildConfig.Verbose, "v", false, "verbose output")
		set.StringVar(&buildConfig.OutputDir, "o", "release", "output directory")
		set.StringVar(&buildConfig.Name, "name", "", "executable name")
		set.StringVar(&buildConfig.NameTemplate, "name-template", "{{.NAME}}{{.EXT}}", "template to use for each file")
		set.StringVar(&buildConfig.BundleTemplate, "bundle-template", "{{.NAME}}_{{.GOOS}}_{{.GOARCH}}{{.ZIP}}", "template to use for each bundle")
		set.BoolVar(&buildConfig.Clean, "clean", false, "clean the output directory before building")
		set.BoolVar(&buildConfig.Generate, "generate", false, "run go generate before building")
		set.BoolVar(&buildConfig.Dry, "dry", false, "run without actually doing anything")
		set.BoolVar(&buildConfig.CGO, "cgo", false, "enabled cgo by setting CGO_ENABLED=1 for each build")
		set.StringVar(&buildConfig.LdFlags, "ldflags", "", "pass ldflags to build command")
		set.BoolVar(&buildConfig.Debug, "debug", false, "include debug symbols in build")
		return nil
	},
	Parse: func(set *flag.FlagSet, args []string) (err error) {
		args, buildConfig.BuildArgs, _ = lib.StringSliceCut(args, "--")
		if err = set.Parse(args[1:]); err != nil {
			return
		}
		buildConfig.OutputDir = filepath.Clean(buildConfig.OutputDir)
		if _, err = lib.ApplyModule(&buildConfig); err != nil {
			return
		}
		if buildConfig.Name == "" {
			return ErrBuildName
		}
		buildConfig.Aliases = set.Args()
		buildConfig.DistributionSet, err = lib.GetBuildTargets(buildConfig)
		return
	},
	Exec: runBuild,
}

func runGenerate(config lib.BuildConfig) (err error) {
	fmt.Println("running go generate")
	cmd := exec.Command("go", "generate")
	cmd.Env = os.Environ()
	cmd.Dir, err = os.Getwd()
	if err != nil {
		return
	}
	return cmd.Run()
}

func runBuild(set *flag.FlagSet) (err error) {
	config := buildConfig
	if config.Dry {
		fmt.Println("** dry run **")
	}
	fmt.Printf("preparing to build %d packages\n", len(config.DistributionSet))

	if !config.Dry && config.Clean {
		if err = lib.CleanDirGlob(config.OutputDir, "*.zip"); err != nil {
			return
		}
	}

	if config.Generate {
		if err = runGenerate(config); err != nil {
			return
		}
	}

	args := []string{"build"}
	for _, dist := range config.DistributionSet {
		outPath := filepath.Join(config.OutputDir, config.Name)
		cmdArgs := append(args, "-o", outPath)
		ldFlags := config.LdFlags
		if !config.Debug {
			ldFlags += " -s"
		}
		if ldFlags != "" {
			cmdArgs = append(cmdArgs, "-ldflags", ldFlags)
		}
		cmdArgs = append(cmdArgs, config.BuildArgs...)
		cmd := exec.CommandContext(context.Background(), "go", cmdArgs...)
		cmd.Env = append(os.Environ(), []string{
			fmt.Sprintf("GOOS=%s", dist.GOOS),
			fmt.Sprintf("GOARCH=%s", dist.GOARCH),
		}...)
		if config.CGO {
			cmd.Env = append(cmd.Env, "CGO_ENABLED=1")
		}
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		fmt.Printf("building %s\\%s\n", dist.GOOS, dist.GOARCH)
		if config.Verbose {
			fmt.Printf("%+v\n", dist)
		}
		if config.Dry {
			continue
		}
		if err = cmd.Run(); err != nil {
			fmt.Println(err)
			// return err
		}
		if err == nil {
			if err = lib.BundleFile(outPath, dist, config); err != nil {
				return
			}
			if err = os.Remove(outPath); err != nil {
				return
			}
		}
	}
	return nil
}

func init() {
	lib.AddCmd(buildCommand)
}
