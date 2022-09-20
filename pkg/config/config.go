package config

import (
	"flag"
	"fmt"
	"os"
	"path"
	"strings"
)

type Config struct {
	// Input is the input file path.
	Input string
	// Output is the output file path.
	// If empty, it generates to the original source code dir with the name <basename>.gen.go".
	Output string
	// Log is the log file path to where the tool write logs.
	Log string
	// DryRun instructs convergen not to write the result code to the output path.
	DryRun bool
	// Prints instructs convergen to print the result code to stdout.
	Prints bool
}

func (c *Config) String() string {
	var sb strings.Builder
	sb.WriteString("config.Config{\n\tInput: \"")
	sb.WriteString(c.Input)
	sb.WriteString("\"\n\tOutput: \"")
	sb.WriteString(c.Output)
	sb.WriteString("\"\n\tLog: \"")
	sb.WriteString(c.Log)
	sb.WriteString("\"\n}")
	return sb.String()
}

func (c *Config) ParseArgs() error {
	output := flag.String("out", "", "output file path")
	logs := flag.Bool("log", false, "write log to <output path>.log")
	dryRun := flag.Bool("dry", false, "dry run")
	prints := flag.Bool("print", false, "print result to STDOUT as well")
	flag.Parse()

	err := c.populateDefaults()
	if err != nil {
		return err
	}

	if *output != "" {
		c.Output = *output
	}
	if *logs {
		ext := path.Ext(c.Output)
		c.Log = c.Output[0:len(c.Output)-len(ext)] + ".log"
	}
	c.DryRun = *dryRun
	c.Prints = *prints

	return nil
}

func (c *Config) populateDefaults() error {
	inputPath := flag.Arg(0)
	if inputPath == "" {
		inputPath = os.Getenv("GOFILE")
	}
	if inputPath == "" {
		return fmt.Errorf("ERR: no input path specified")
	}

	c.Input = inputPath
	ext := path.Ext(inputPath)
	c.Output = inputPath[0:len(inputPath)-len(ext)] + ".gen" + ext

	return nil
}
