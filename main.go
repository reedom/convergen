package main

import (
	"fmt"
	"os"

	"github.com/reedom/convergen/pkg/config"
	"github.com/reedom/convergen/pkg/runner"
)

func main() {
	var conf config.ConvergenConfig
	if err := conf.ParseArgs(); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	if err := runner.Run(conf); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
