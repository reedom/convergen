package config

import (
	"os"
	"path"
)

type ConvergenConfig struct {
	// Output is the file name or path for the generated code.
	// If empty, it generates to the original source code dir with the name <basename>.gen.go".
	Output string
}

func (c ConvergenConfig) OutputPath(srcPath string) string {
	pwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	if c.Output != "" {
		return path.Join(pwd, c.Output)
	}

	var fullPath string
	if path.IsAbs(srcPath) {
		fullPath = path.Dir(srcPath)
	} else {
		fullPath = path.Join(pwd, srcPath)
	}
	ext := path.Ext(srcPath)
	return fullPath[0:len(fullPath)-len(ext)] + ".gen" + ext
}
