package config

import (
	"flag"
	"fmt"
)

const HelpMessage = `
  Help Flag info
`

func PrintHelp() {
	if HelpMessage != "" {
		fmt.Printf("%s", HelpMessage)
	} else {
		flag.Usage()
	}
}
