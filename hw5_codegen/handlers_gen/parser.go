package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"strings"
)

type Methods map[string][]MethodTemplate
type Structs map[string]StructTemplate

type StructFields struct {
	Min      bool
	MinValue int
	Max      bool
	MaxValue int
	Enum     bool
	Enums    []string
}

type Parser struct {
	generator string
	validator string

	file *ast.File
}

func NewParser(path, generator, validator string) (Parser, error) {
	p := Parser{}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return p, err
	}

	p.file = file
	p.generator = generator
	p.validator = validator

	return p, nil
}

func codegen(pref string, api string) (string, bool) {
	return strings.TrimPrefix(api, pref), strings.HasPrefix(api, pref)
}

func functionRecv(fd *ast.FuncDecl) (string, error) {
	if fd.Recv == nil {
		return "", fmt.Errorf("must be method")
	}

	re, ok := fd.Recv.List[0].Type.(*ast.StarExpr)
	if !ok {
		return "", fmt.Errorf("must be pointer type")
	}

	rn, ok := re.X.(*ast.Ident)
	if !ok {
		return "", fmt.Errorf("illegal receiver name")
	}

	return rn.Name, nil
}

func functionArgs(fd *ast.FuncDecl) []string {
	args := make([]string, 0)
	for _, param := range fd.Type.Params.List {
		switch t := param.Type.(type) {
		case *ast.Ident:
			args = append(args, t.Name)
		case *ast.StarExpr:
			if se, ok := t.X.(*ast.Ident); ok {
				args = append(args, se.Name)
			}
		}
	}
	return args
}

func functionRets(fd *ast.FuncDecl) []string {
	rets := make([]string, 0)
	for _, param := range fd.Type.Results.List {
		switch t := param.Type.(type) {
		case *ast.Ident:
			rets = append(rets, t.Name)
		case *ast.StarExpr:
			if se, ok := t.X.(*ast.Ident); ok {
				rets = append(rets, se.Name)
			}
		}
	}

	return rets
}

func (p Parser) ParseMethods() (Methods, error) {
	ms := map[string][]MethodTemplate{}

	for _, decl := range p.file.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		cg, ok := codegen(p.generator+":api", fd.Doc.Text())
		if !ok {
			continue
		}

		recv, err := functionRecv(fd)
		if err != nil {
			return ms, err
		}

		args := functionArgs(fd)
		rets := functionRets(fd)

		api := struct {
			Url    string `json:"url"`
			Method string `json:"method"`
			Auth   bool   `json:"auth"`
		}{}

		err = json.Unmarshal([]byte(cg), &api)
		if err != nil {
			return ms, fmt.Errorf("bad api description")
		}

		m := MethodTemplate{
			Name:   fd.Name.String(),
			Args:   strings.Join(args, ", "),
			Return: strings.Join(rets, ", "),
			Auth:   api.Auth,
			URL:    api.Url,
			Method: api.Method,
		}

		ms[recv] = append(ms[recv], m)
	}

	return ms, nil
}

func validate(tag, validator string) (string, bool) {
	return strings.TrimPrefix(tag, validator), strings.HasPrefix(tag, validator)
}

func structFields(st *ast.StructType, validator string) ([]StructFields, error) {
	fields := make([]StructFields, 0)

	for _, field := range st.Fields.List {
		if field.Tag == nil {
			continue
		}

		tag, ok := validate(field.Tag.Value, validator+":")
		if !ok {
			continue
		}

		// parse fields
	}

	return fields, nil
}

func (p Parser) ParseStructs(names ...string) (Structs, error) {
	ss := Structs{}

	for _, decl := range p.file.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}

		ts, ok := gd.Specs[0].(*ast.TypeSpec)
		if !ok {
			continue
		}

		st, ok := ts.Type.(*ast.StructType)
		if !ok {
			continue
		}

		if !strings.Contains(strings.Join(names, ""), ts.Name.Name) {
			continue
		}
	}

	return ss, nil
}
