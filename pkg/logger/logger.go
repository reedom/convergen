package logger

import (
	"fmt"
	"io"
	"log"
	"os"
)

// LoggerOpt is a function that modifies the logger options.
type LoggerOpt func(*option)

// option is a structure that holds the logger options.
type option struct {
	enabled bool      // enabled is a flag that determines whether the logger is enabled or not.
	forTest bool      // forTest is a flag that determines whether the logger is used for testing or not.
	out     io.Writer // out is the output destination of the logger.
}

var logger = log.New(io.Discard, "", 0) // logger is the info logger instance.
var elogger = log.New(os.Stderr, "", 0) // elogger is the error logger instance.

// Enable sets the enabled flag to true.
func Enable() LoggerOpt {
	return func(opt *option) {
		opt.enabled = true
	}
}

// Output sets the output destination of the logger.
func Output(out io.Writer) LoggerOpt {
	return func(opt *option) {
		opt.out = out
	}
}

// ForTest sets the forTest flag to true.
func ForTest() LoggerOpt {
	return func(opt *option) {
		opt.forTest = true
	}
}

// SetupLogger sets up the logger with the provided options.
func SetupLogger(options ...LoggerOpt) {
	opt := option{}
	for _, o := range options {
		o(&opt)
	}

	if !opt.enabled {
		logger = log.New(io.Discard, "", 0)
		elogger = log.New(os.Stderr, "", 0)
	} else if opt.out != nil {
		//f, err := os.OpenFile(opt.outputPath, os.O_WRONLY | os.O_CREATE | os.O_TRUNC, 0644)
		//if err != nil {
		//	return err
		//}
		logger = log.New(opt.out, "", log.LstdFlags)
		elogger = log.New(os.Stderr, "", 0)
	} else {
		logger = log.New(os.Stdout, "", log.LstdFlags)
		elogger = log.New(io.Discard, "", 0)
	}

	if opt.forTest {
		elogger = log.New(io.Discard, "", 0)
	}
}

// Errorf logs the formatted error message.
func Errorf(format string, a ...any) error {
	err := fmt.Errorf(format, a...)
	logger.Println(err.Error())
	elogger.Println(err.Error())
	return err
}

// Warnf logs the formatted warning message.
func Warnf(format string, a ...any) {
	logger.Printf(format, a...)
	elogger.Printf(format, a...)
}

// Printf logs the formatted message.
func Printf(format string, a ...any) {
	logger.Printf(format, a...)
}
