package runner

import (
	"os"

	"github.com/reedom/convergen/pkg/config"
	"github.com/reedom/convergen/pkg/generator"
	"github.com/reedom/convergen/pkg/logger"
	"github.com/reedom/convergen/pkg/model"
	"github.com/reedom/convergen/pkg/parser"
)

func Run(conf config.Config) error {
	if conf.Log != "" {
		f, err := os.OpenFile(conf.Log, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0644)
		if err != nil {
			return err
		}
		logger.SetupLogger(logger.Enable(), logger.Output(f))
	}

	p, err := parser.NewParser(conf.Input)
	if err != nil {
		return err
	}

	methods, err := p.Parse()
	if err != nil {
		return err
	}

	builder := p.CreateBuilder()
	functions, err := builder.CreateFunctions(methods)
	if err != nil {
		return err
	}

	pre, post, err := p.GenerateBaseCode()
	if err != nil {
		return err
	}

	code := model.Code{
		Pre:       pre,
		Post:      post,
		Functions: functions,
	}

	g := generator.NewGenerator(code)
	_, err = g.Generate(conf.Output, conf.Prints, conf.DryRun)
	if err != nil {
		return err
	}

	return nil
}
