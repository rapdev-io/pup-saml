package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	subcmd := "discover"
	org := os.Getenv("DD_ORG")

	args := os.Args[1:]
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		subcmd = args[0]
		args = args[1:]
	}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--org":
			if i+1 < len(args) {
				org = args[i+1]
				i++
			}
		case "--help", "-h":
			printHelp()
			os.Exit(0)
		}
	}

	switch subcmd {
	case "discover":
		runDiscover(org)
	case "mappings":
		runMappings(org)
	default:
		fmt.Fprintf(os.Stderr, "pup-saml: unknown subcommand %q\n", subcmd)
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Fprintln(os.Stderr, "Usage: pup saml <subcommand> [--org <name>]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Subcommands:")
	fmt.Fprintln(os.Stderr, "  discover   Full SAML & auth discovery (default)")
	fmt.Fprintln(os.Stderr, "  mappings   List authn_mappings with resolved role names")
}
