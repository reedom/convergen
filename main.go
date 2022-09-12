package main

import (
	"fmt"
	"os"

	"github.com/reedom/loki/pkg/config"
	"github.com/reedom/loki/pkg/runner"
)

func main() {
	var conf config.lokiConfig
	if err := conf.ParseArgs(); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	if err := runner.Run(conf); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}
