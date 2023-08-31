package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
)

const (
	fieldRequired  = "required"
	fieldDefault   = "default"
	fieldParamName = "paramname"
	fieldMinValue  = "min"
	fieldMaxValue  = "max"
	fieldEnum      = "enum"
)

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

func codegen(tag, pref string) (string, bool) {
	return strings.TrimPrefix(tag, pref), strings.HasPrefix(tag, pref)
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

func (p *Parser) ParseMethods() (Methods, error) {
	ms := Methods{}

	for _, decl := range p.file.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		cg, ok := codegen(fd.Doc.Text(), p.generator+":api")
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
			Param:  args[0],
			Return: rets[0],
			Auth:   api.Auth,
			URL:    api.Url,
			Method: api.Method,
		}

		ms[recv] = append(ms[recv], m)
	}

	return ms, nil
}

func methodsParams(methods Methods) Params {
	ps := Params{}

	for _, ms := range methods {
		for _, m := range ms {
			ps[m.Param] = []StructField{}
		}
	}

	return ps
}

func structField(field *ast.Field, tags string) (StructField, error) {
	sf := StructField{}

	for _, tag := range strings.Split(tags, ",") {
		tag = strings.Trim(tag, "\",`")
		if strings.HasPrefix(tag, fieldRequired) {
			sf.Required = true
		}

		if strings.HasPrefix(tag, fieldDefault) {
			s := strings.Split(tag, "=")
			if len(s) != 2 {
				return sf, fmt.Errorf("default without value")
			}

			sf.Default = s[1]
		}

		if strings.HasPrefix(tag, fieldMinValue) {
			s := strings.Split(tag, "=")
			if len(s) != 2 {
				return sf, fmt.Errorf("min without value")
			}

			v, err := strconv.Atoi(s[1])
			if err != nil {
				return sf, err
			}
			sf.Min = true
			sf.MinValue = v
		}

		if strings.HasPrefix(tag, fieldMaxValue) {
			s := strings.Split(tag, "=")
			if len(s) != 2 {
				return sf, fmt.Errorf("max without value")
			}

			v, err := strconv.Atoi(s[1])
			if err != nil {
				return sf, err
			}
			sf.Max = true
			sf.MaxValue = v
		}

		if strings.HasPrefix(tag, fieldEnum) {
			s := strings.Split(tag, "=")
			if len(s) != 2 {
				return sf, fmt.Errorf("enum without value")
			}

			sf.Enum = true
			sf.Enums = strings.Split(s[1], "|")
		}

		if strings.HasPrefix(tag, fieldParamName) {
			sf.ParamValue = strings.Split(tag, "=")[1]
		} else {
			sf.ParamValue = strings.ToLower(field.Names[0].Name)
		}
	}

	sf.ParamName = field.Names[0].Name
	sf.ParamType = field.Type.(*ast.Ident).Name

	return sf, nil
}

func (p Parser) ParseParams(methods Methods) (Params, error) {
	ps := methodsParams(methods)

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

		_, ok = ps[ts.Name.Name]
		if !ok {
			continue
		}

		for _, field := range st.Fields.List {
			tags, ok := codegen(field.Tag.Value, "`"+p.validator+":")
			if !ok {
				continue
			}

			sf, err := structField(field, tags)
			if err != nil {
				return ps, err
			}

			ps[ts.Name.Name] = append(ps[ts.Name.Name], sf)
		}
	}

	return ps, nil
}
