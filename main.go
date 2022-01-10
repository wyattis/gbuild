package main

import (
	"fmt"
	"os"

	_ "github.com/wyattis/gbuild/cmd"
	"github.com/wyattis/gbuild/lib"
)

func main() {
	if err := lib.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
