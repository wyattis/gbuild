package main

import (
	"fmt"
	_ "gbuild/cmd"
	"gbuild/lib"
	"os"
)

func main() {
	if err := lib.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
