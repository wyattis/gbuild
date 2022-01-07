package cmd

import (
	"flag"
	"fmt"
	"gbuild/lib"
)

type AliasesConfig struct {
	ShowTargets bool
}

var aliasesConfig = AliasesConfig{}
var aliasesCmd = lib.Cmd{
	Name:             "aliases",
	ShortDescription: "list available aliases",
	Init: func(set *flag.FlagSet) error {
		set.BoolVar(&aliasesConfig.ShowTargets, "targets", false, "include a list of targets for each alias")
		return nil
	},
	Exec: func(set *flag.FlagSet) (err error) {
		config := aliasesConfig
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
	lib.AddCmd(aliasesCmd)
}
