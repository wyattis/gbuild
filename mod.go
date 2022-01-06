package main

import (
	"bufio"
	"errors"
	"os"
	"strings"
)

type ModConfig struct {
	Name      string
	GoVersion string
}

func parseMod(loc string) (mod ModConfig, err error) {
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
		if strings.HasPrefix(line, "module") {
			_, name, found := StringCut(line, " ")
			if !found {
				err = errors.New("invalid module name in go.mod")
				return
			}
			mod.Name = name
			return
		}
	}
	return
}
