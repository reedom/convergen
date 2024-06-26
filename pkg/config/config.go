package config

import (
	"flag"
	"fmt"
	"os"
	"path"
	"strings"
)

// Usage prints the usage of the tool.
func Usage() {
	var sb strings.Builder
	sb.WriteString("\nUsage: convergen [flags] <input path>\n\n")
	sb.WriteString("By default, the generated code is written to <input path>.gen.go\n\n")
	sb.WriteString("Flags:\n")
	_, _ = fmt.Fprint(os.Stderr, sb.String())
	flag.PrintDefaults()
}

type Config struct {
	// Input is the path of the input file.
	Input string
	// Output is the path where the generated code will be saved.
	// If empty, the generated code will be saved in the same directory as
	// the input file with the name "<basename>.gen.go".
	Output string
	// Log is the path of the log file where the tool writes logs.
	Log string
	// DryRun instructs convergen not to write the generated code to the output path.
	DryRun bool
	// Prints instructs convergen to print the generated code to stdout.
	Prints bool
}

// String returns the string representation of the config.
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

// ParseArgs parses the command line arguments.
func (c *Config) ParseArgs() error {
	output := flag.String("out", "", "Set the output file path")
	outputSuffix := flag.String("suffix", "", "Set the output suffix file path")
	logs := flag.Bool("log", false, "Write log messages to <output path>.log.")
	dryRun := flag.Bool("dry", false, "Perform a dry run without writing files.")
	prints := flag.Bool("print", false, "Print the resulting code to STDOUT as well.")

	flag.Usage = Usage
	flag.Parse()

	inputPath := flag.Arg(0)
	if inputPath == "" {
		inputPath = os.Getenv("GOFILE")
	}
	if inputPath == "" {
		flag.Usage()
		os.Exit(1)
	}
	c.Input = inputPath

	if *output != "" {
		c.Output = *output
	} else {
		ext := path.Ext(inputPath)
		suffix := ".gen"
		if outputSuffix != nil && *outputSuffix != "" {
			suffix = "." + *outputSuffix
		}
		c.Output = inputPath[0:len(inputPath)-len(ext)] + suffix + ext
	}

	if *logs {
		ext := path.Ext(c.Output)
		c.Log = c.Output[0:len(c.Output)-len(ext)] + ".log"
	}
	c.DryRun = *dryRun
	c.Prints = *prints

	return nil
}
