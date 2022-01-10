package cmd

import (
	_ "embed"
	"flag"
	"fmt"

	"github.com/wyattis/gbuild/lib"
)

//go:embed manual/list.md
var listLongDescription string

type ListConfig struct {
	ShowTargets bool
}

var listConfig = ListConfig{}
var listCmd = lib.Cmd{
	Name:             "list",
	ShortDescription: "List available aliases",
	LongDescription:  listLongDescription,
	Init: func(set *flag.FlagSet) error {
		set.BoolVar(&listConfig.ShowTargets, "targets", false, "include a list of targets for each alias")
		return nil
	},
	Exec: func(set *flag.FlagSet) (err error) {
		config := listConfig
		distributions, err := lib.GetAllDistributions()
		if err != nil {
			return
		}

		aliases := lib.GetAliases(distributions)
		keys := set.Args()
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
	},
}

func init() {
	lib.AddCmd(listCmd)
}
