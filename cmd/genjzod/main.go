package main

import (
	"fmt"
	"os"

	"github.com/miroir-framework/miroir/go/jzodgen"
)

func main() {
	path, err := jzodgen.WriteBootstrapPackage()
	if err != nil {
		fmt.Fprintf(os.Stderr, "genjzod: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("wrote", path)
}
