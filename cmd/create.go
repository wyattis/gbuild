//go:build experimental
// +build experimental

package cmd

import (
	"bytes"
	_ "embed"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/wyattis/gbuild/lib"
	"github.com/wyattis/z/zslice/zstrings"
)

//go:embed manual/create.md
var createDescription string

type ActionsConfig struct {
	lib.BuildConfig
	CreateRelease    bool
	WorkflowDispatch bool
	BuildBinUrl      string
	BuildBinName     string
	PreRelease       bool
	Draft            bool
	Args             []string
}

var createConfig = ActionsConfig{}
var createCmd = lib.Cmd{
	Name:             "create",
	ShortDescription: "generate workflow and bash scripts using the same settings",
	LongDescription:  createDescription,
	Init:             initCreate,
	Parse: func(set *flag.FlagSet, args []string) (err error) {
		args, buildArgs, _ := zstrings.Cut(args, "--")
		set = flag.NewFlagSet("create", flag.ContinueOnError)
		if err = initCreate(set); err != nil {
			return
		}
		if err = set.Parse(args); err != nil {
			return
		}
		buildSet := flag.NewFlagSet("", flag.ExitOnError)
		if err = initBuild(buildSet); err != nil {
			return
		}
		args = append(set.Args(), "--")
		if err = buildCommand.Parse(buildSet, append(args, buildArgs...)); err != nil {
			return
		}
		createConfig.BuildConfig = buildConfig
		return
	},
	Exec: func(set *flag.FlagSet) (err error) {
		tmpl := template.New("")
		tmpl.Funcs(makeFuncMap(tmpl))
		tmpl, err = tmpl.ParseFS(os.DirFS("templates/github"), "*.tmpl")
		fmt.Printf("creating Github Actions workflow with %d targets\n", len(createConfig.DistributionSet))
		if err = os.MkdirAll(".github/workflows", os.ModeDir); err != nil {
			return
		}
		f, err := os.Create(filepath.Join(".github/workflows", fmt.Sprintf("release-%s.yml", createConfig.Name)))
		if err != nil {
			return
		}
		defer f.Close()
		return tmpl.ExecuteTemplate(f, "actions", createConfig)
	},
}

func initCreate(set *flag.FlagSet) error {
	set.BoolVar(&createConfig.CreateRelease, "release", false, "automatically create a release via the action")
	set.BoolVar(&createConfig.WorkflowDispatch, "workflow-dispatch", true, "allow dispatching the workflow manually")
	set.BoolVar(&createConfig.PreRelease, "prerelease", false, "mark the release as a pre-release")
	set.BoolVar(&createConfig.Draft, "draft", false, "mark the release as a draft")
	set.StringVar(&createConfig.BuildBinUrl, "build-bin-url", "github.com/wyattis/gbuild@latest", "change the location of the build binary")
	set.StringVar(&createConfig.BuildBinName, "build-bin-name", "gbuild", "change the name of the binary to execute")
	return nil
}

func makeFuncMap(t *template.Template) template.FuncMap {
	return template.FuncMap{
		"join": strings.Join,
		"add": func(num int, add int) int {
			return num + add
		},
		"indent": func(padding int, val string) (res string) {
			res = "\n"
			lines := strings.Split(val, "\n")
			for _, line := range lines {
				res += strings.Repeat(" ", padding) + line + "\n"
			}
			return
		},
		"include": func(name string, data interface{}) (string, error) {
			buf := bytes.NewBuffer(nil)
			if err := t.ExecuteTemplate(buf, name, data); err != nil {
				return "", err
			}
			return buf.String(), nil
		},
		"filename": func(path string) string {
			return filepath.FromSlash(filepath.Base(path))
		},
	}
}

func init() {
	lib.AddCmd(createCmd)
}
