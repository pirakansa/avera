package main

import (
	"fmt"
	"os"
)

func main() {
	if _, err := fmt.Fprintln(os.Stdout, "Hello, world!"); err != nil {
		os.Exit(1)
	}
}
