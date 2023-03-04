// MIT License
//
// Copyright (c) 2022-2023 HANAI Tohru

/*
Convergen is a code generator that creates functions for type-to-type copy.
It generates functions that copy field to field between two types.

Notation Table
--------------

| notation                                  | location           | summary                                                                               |
|-------------------------------------------|--------------------|---------------------------------------------------------------------------------------|
| :match &lt;`name` &#124; `none`>          | interface, method  | Sets the field matcher algorithm (default: `name`).                                   |
| :style &lt;`return` &#124; `arg`>         | interface, method  | Sets the style of the assignee variable input/output (default: `return`).             |
| :recv &lt;_var_>                          | method             | Specifies the source value as a receiver of the generated function.                   |
| :reverse                                  | 	method            | Reverses the copy direction. Might be useful with receiver form.                      |
| :case	                                    | interface, method  | Sets case-sensitive for name match (default).                                         |
| :case:off	                                | interface, method  | Sets case-insensitive for name match.                                                 |
| :getter	                                  | interface, method  | Includes getters for name match.                                                      |
| :getter:off	                              | interface, method  | Excludes getters for name match (default).                                            |
| :stringer                                 | 	interface, method | Calls String() if appropriate in name match.                                          |
| :stringer:off                             | 	interface, method | Calls String() if appropriate in name match (default).                                |
| :typecast	                                | interface, method	 | Allows type casting if appropriate in name match.                                     |
| :typecast:off                             | 	interface, method | Suppresses type casting if appropriate in name match (default).                       |
| :skip &lt;_dst field pattern_>            | method             | Marks the destination field to skip copying. Regex is allowed in /…/ syntax.          |
| :map &lt;_src_> &lt;_dst field_>          | method             | the pair as assign source and destination.                                            |
| :conv &lt;_func_> &lt;_src_> [_to field_] | method             | Converts the source value by the converter and assigns its result to the destination. |
| :literal &lt;_dst_> &lt;_literal_>        | method             | Assigns the literal expression to the destination.                                    |
| :preprocess &lt;_func_>                   | method             | Calls the function at the beginning of the convergen func.                            |
| :postprocess &lt;_func_>                  | method             | Calls the function at the end of the convergen function.                              |

Installation and Introduction
-----------------------------

### Use as a Go generator

To use Convergen as a Go generator, install the module in your Go project directory via go get:

```shell
$ go get -u github.com/reedom/convergen@latest
```

Then, write a generator as follows:

```go
//go:generate go run github.com/reedom/convergen@v0.6.1
type Convergen interface {
    …
}
````

### Use as a CLI command

To use Convergen as a CLI command, install the command via go install:

```shell
$ go install github.com/reedom/convergen@latest
```

You can then generate code by calling:

```shell
$ convergen any-codegen-defined-code.go
```

The CLI help shows:

```shell
Usage: convergen [flags] <input path>

By default, the generated code is written to <input path>.gen.go

Flags:
  -dry
        Perform a dry run without writing files.
  -log
        Write log messages to <output path>.log.
  -out string
        Set the output file path.
  -print
        Print the resulting code to STDOUT as well.
```
*/

package main
