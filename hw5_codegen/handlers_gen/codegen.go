package main

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
)

type Methods map[string][]MethodTemplate
type Params map[string][]StructField

type MethodTemplate struct {
	Name   string
	URL    string
	Param  string
	Return string
	Auth   bool
	Method string
}

type PackageTemplate struct {
	Name    string
	Imports []string
}

type StructField struct {
	ParamType  string
	ParamName  string
	ParamValue string
	Default    string
	Required   bool
	Min        bool
	MinValue   int
	Max        bool
	MaxValue   int
	Enum       bool
	Enums      []string
}

func main() {
	if len(os.Args) < 3 {
		panic("usage: codegen input.go output.go")
	}

	in := os.Args[1]
	out := os.Args[2]

	p, err := NewParser(in, "apigen", "apivalidator")
	if err != nil {
		panic(err)
	}

	methods, err := p.ParseMethods()
	if err != nil {
		panic(err)
	}

	params, err := p.ParseParams(methods)
	if err != nil {
		panic(err)
	}

	buf := new(bytes.Buffer)
	g := Generator{buf}

	g.DoPackage("main", "net/http", "net/url", "encoding/json", "io", "fmt", "strconv", "strings")
	g.DoWrappers(methods)
	g.DoHandlers(methods)
	g.DoParams(params)

	formated, err := format.Source(buf.Bytes())
	if err != nil {
		panic(err)
	}

	output, err := os.Create(out)
      	fmt.Fprintln(output, string(formated))
}
