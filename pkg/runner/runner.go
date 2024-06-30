package runner

import (
	"os"

	"github.com/reedom/convergen/pkg/config"
	"github.com/reedom/convergen/pkg/generator"
	"github.com/reedom/convergen/pkg/generator/model"
	"github.com/reedom/convergen/pkg/logger"
	"github.com/reedom/convergen/pkg/parser"
)

// Run runs the convergen code generator using the provided configuration.
// If a log file path is specified in the configuration, the logger will output to that file.
// It creates a parser instance from the input and output paths in the configuration,
// and then generates a list of methods from the parsed source code. Using a function builder,
// the generator creates a block of functions for each set of methods and combines them with
// the parsed base code. Finally, it generates the output files using the generated code and
// the provided configuration options.
func Run(conf config.Config) error {
	if conf.Log != "" {
		f, err := os.OpenFile(conf.Log, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0644)
		if err != nil {
			return err
		}
		logger.SetupLogger(logger.Enable(), logger.Output(f))
	}

	p, err := parser.NewParser(&conf)
	if err != nil {
		return err
	}

	methods, err := p.Parse()
	if err != nil {
		return err
	}

	builder := p.CreateBuilder()

	var funcBlocks []model.FunctionsBlock
	for _, info := range methods {
		functions, err := builder.CreateFunctions(info.Methods)
		if err != nil {
			return err
		}
		block := model.FunctionsBlock{
			Marker:    info.Marker,
			Functions: functions,
		}
		funcBlocks = append(funcBlocks, block)
	}

	baseCode, err := p.GenerateBaseCode()
	if err != nil {
		return err
	}

	code := model.Code{
		BaseCode:       baseCode,
		FunctionBlocks: funcBlocks,
	}

	g := generator.NewGenerator(code)
	_, err = g.Generate(conf.Output, conf.Prints, conf.DryRun)
	if err != nil {
		return err
	}

	return nil
}
