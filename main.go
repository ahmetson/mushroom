package main

import (
	"fmt"
	"io"
	"os"
)

func main() {
	if err := run(os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "mushroom: %v\n", err)
		os.Exit(1)
	}
}

func run(stdout io.Writer) error {
	_, err := fmt.Fprintln(stdout, "mushroom")
	return err
}
