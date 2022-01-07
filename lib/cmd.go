package lib

import (
	"flag"
	"fmt"
	"os"
)

type Commands []Cmd

type Cmd struct {
	Name             string
	ShortDescription string
	LongDescription  string
	Init             func(*flag.FlagSet) error
	Exec             func(*flag.FlagSet) error
}

var commands = []Cmd{}

func AddCmd(cmd Cmd) {
	commands = append(commands, cmd)
}

var helpCmd = Cmd{
	Name:             "help",
	ShortDescription: "see more information about a command",
}

func Execute() (err error) {

	commands = append(commands, helpCmd)

	flag.CommandLine.SetOutput(os.Stdin)

	nameColSize := 0
	flagSets := []*flag.FlagSet{}
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
			if err = flagSet.Parse(flag.Args()[1:]); err != nil {
				return
			}
			return cmd.Exec(flagSet)
		}
	}

	fmt.Printf("\nmust supply a valid command\n\n")
	for _, cmd := range commands {
		fmt.Printf("  %-10s %s\n", cmd.Name, cmd.ShortDescription)
	}

	return nil
}
