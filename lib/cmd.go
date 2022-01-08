package lib

import (
	"flag"
	"fmt"
	"io"
	"os"
)

type Commands []Cmd

type Cmd struct {
	Name             string
	ShortDescription string
	LongDescription  string
	Parse            func(set *flag.FlagSet, args []string) error
	Init             func(*flag.FlagSet) error
	Exec             func(*flag.FlagSet) error
}

var commands = []Cmd{}
var flagSets = []*flag.FlagSet{}

func AddCmd(cmd Cmd) {
	commands = append(commands, cmd)
}

var helpCmd = Cmd{
	Name:             "help",
	ShortDescription: "See more information about a command",
	Exec: func(set *flag.FlagSet) (err error) {
		nonHelpCmds := commands[:len(commands)-1]
		for i, cmd := range nonHelpCmds {
			if len(set.Args()) > 0 && cmd.Name == set.Args()[0] {
				_, err = fmt.Fprintf(set.Output(), "\n%s\n", cmd.LongDescription)
				if err != nil {
					return
				}
				_, err = fmt.Fprintf(set.Output(), "\n%s options:\n", cmd.Name)
				if err != nil {
					return
				}
				flagSets[i].PrintDefaults()
				return
			}
		}

		fmt.Fprintln(set.Output(), "supply a command to learn more about it. Example: gbuild help list")
		return printCommands(set.Output(), nonHelpCmds)
	},
}

func Execute() (err error) {

	AddCmd(helpCmd)

	flag.CommandLine.SetOutput(os.Stdout)

	nameColSize := 0
	// Initialize all commands
	for _, cmd := range commands {
		if len(cmd.Name) > nameColSize {
			nameColSize = len(cmd.Name)
		}
		flagSet := flag.NewFlagSet(cmd.Name, flag.ExitOnError)
		flagSets = append(flagSets, flagSet)
		if cmd.Init != nil {
			if err := cmd.Init(flagSet); err != nil {
				return err
			}
		}
	}

	flag.Parse()
	for i, cmd := range commands {
		if cmd.Name == flag.Arg(0) {
			flagSet := flagSets[i]
			if cmd.Parse == nil {
				err = flagSet.Parse(flag.Args()[1:])
			} else {
				err = cmd.Parse(flagSet, flag.Args())
			}
			if err != nil {
				return
			}
			return cmd.Exec(flagSet)
		}
	}

	fmt.Printf("\nmust supply a valid command\n\n")
	printCommands(flag.CommandLine.Output(), commands)

	return nil
}

func printCommands(out io.Writer, commands []Cmd) (err error) {
	for _, cmd := range commands {
		_, err = fmt.Fprintf(out, "  %-10s %s\n", cmd.Name, cmd.ShortDescription)
		if err != nil {
			return
		}
	}
	return
}
