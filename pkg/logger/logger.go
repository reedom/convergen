package logger

import (
	"fmt"
	"io"
	"log"
	"os"
)

type loggerOpt func(*option)

type option struct {
	enabled bool
	out     io.Writer
}

var logger *log.Logger = log.New(io.Discard, "", 0)

func Enable() loggerOpt {
	return func(opt *option) {
		opt.enabled = true
	}
}

func Output(out io.Writer) loggerOpt {
	return func(opt *option) {
		opt.out = out
	}
}

func SetupLogger(options ...loggerOpt) {
	opt := option{}
	for _, o := range options {
		o(&opt)
	}

	if !opt.enabled {
		logger = log.New(io.Discard, "", 0)
	} else if opt.out != nil {
		//f, err := os.OpenFile(opt.outputPath, os.O_WRONLY | os.O_CREATE | os.O_TRUNC, 0644)
		//if err != nil {
		//	return err
		//}
		logger = log.New(opt.out, "", log.LstdFlags)
	} else {
		logger = log.New(os.Stdout, "", log.LstdFlags)
	}
}

func Errorf(format string, a ...any) error {
	err := fmt.Errorf(format, a...)
	logger.Println(err.Error())
	return err
}

func Printf(format string, a ...any) {
	logger.Printf(format, a...)
}
