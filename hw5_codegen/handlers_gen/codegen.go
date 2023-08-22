package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"strings"
	"text/template"
)

type MethodTemplate struct {
	Name   string
	URL    string
	Args   string
	Return string
	Auth   bool
	Method string
}

type PackageTemplate struct {
	Name    string
	Imports []string
}

type StructTemplate struct{}

type Generator struct {
	w io.Writer

	generator string
	validator string

	methods map[string][]MethodTemplate
	structs map[string]StructTemplate
}

func NewGenerator(w io.Writer, generator, validator string) Generator {
	return Generator{
		w:         w,
		generator: generator,
		validator: validator,
		methods:   map[string][]MethodTemplate{},
		structs:   map[string]StructTemplate{},
	}
}

func (g *Generator) ParseFile(file *ast.File) error {
	for _, decl := range file.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if ok && strings.HasPrefix(fd.Doc.Text(), g.generator) && fd.Recv != nil {
			err := g.parseHandler(fd)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (g *Generator) parseHandler(fd *ast.FuncDecl) error {
	if len(fd.Recv.List) != 1 {
		return fmt.Errorf("must be 1 receiver")
	}

	re, ok := fd.Recv.List[0].Type.(*ast.StarExpr)
	if !ok {
		return fmt.Errorf("must be pointer type")
	}

	rn, ok := re.X.(*ast.Ident)
	if !ok {
		return fmt.Errorf("illegal receiver name")
	}

	args := make([]string, 0)
	for _, param := range fd.Type.Params.List {
		switch t := param.Type.(type) {
		case *ast.Ident:
			args = append(args, t.Name)
		case *ast.StarExpr:
			if se, ok := t.X.(*ast.Ident); ok {
				args = append(args, "*"+se.Name)
			}
		}
	}

	rets := make([]string, 0)
	for _, param := range fd.Type.Results.List {
		switch t := param.Type.(type) {
		case *ast.Ident:
			rets = append(rets, t.Name)
		case *ast.StarExpr:
			if se, ok := t.X.(*ast.Ident); ok {
				args = append(args, "*"+se.Name)
			}
		}
	}

	api := struct {
		Url    string `json:"url"`
		Method string `json:"method"`
		Auth   bool   `json:"auth"`
	}{}

	com := strings.TrimPrefix(fd.Doc.Text(), g.generator+":api")
	err := json.Unmarshal([]byte(com), &api)
	if err != nil {
		return fmt.Errorf("bad json api description")
	}

	m := MethodTemplate{
		Name:   fd.Name.String(),
		Args:   strings.Join(args, ", "),
		Return: strings.Join(rets, ", "),
		Auth:   api.Auth,
		URL:    api.Url,
		Method: api.Method,
	}

	g.methods[rn.Name] = append(g.methods[rn.Name], m)

	return nil
}

func (g *Generator) PrintPackage(name string, imports ...string) {
	t := template.Must(template.New("PackageTemplate").Parse(`
package {{.Name}}

import (
{{range .Imports}}"{{.}}"
{{end}})
	`))
	temp := PackageTemplate{
		Name:    name,
		Imports: imports,
	}

	t.Execute(g.w, temp)
}

func (g *Generator) PrintHandlers() {
	t := template.Must(template.New("ServeTemplate").Parse(`
	{{range $r, $m := .}}
func (srv *{{$r}}) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	    switch r.URL.Path {
	    {{range $m}}case "{{.URL}}":
		srv.handle{{.Name}}(w, r)
	    {{end}}
	    default:
		w.WriteHeader(404)
	    }
}
	{{end}}
	`))

	t.Execute(g.w, g.methods)
}

func (g *Generator) PrintWrappers() {
	t := template.Must(template.New("WrapperTemplate").Parse(`
func handleApiError(w http.ResponseWriter, code int, err string) {
	      e := struct {
		  err string ` + "`" + `json:"error"` + "`" + `
	      }{}

	      e.err = err
	      r, _ := json.Marshal(e)
	      w.WriteHeader(code)
	      w.Header().Set("Content-Type", "application/json")
	      w.Write(r)
}

	{{range $r, $m := .}}
	{{range $m}}
func (srv *{{$r}}) handle{{.Name}}(w http.ResponseWriter, r *http.Request) {

	{{if .Auth -}}
	if r.Header.Get("X-Auth") != "100500" {
	      handleApiError(w, http.StatusForbidden, "unauthorized")
	      return
	}
	{{end -}}

	{{if .Method -}}
	if r.Method != "{{.Method}}" {
	      handleApiError(w, http.StatusNotAcceptable, "bad method") 
	      return
	}
	{{end -}}

	params := url.Values{}
	if r.Method == "GET" {
		params = r.URL.Query()
	} else {
		body, _ := io.ReadAll(r.Body)
		params, _ = url.ParseQuery(string(body))
	}

	in, err := New{{.Args}}(params)
	if err != nil {
		handleApiError(w, http.StatusBadRequest, err.Error())
		return
	}

	r, _ := json.Marshal(in)
	w.WriteHeader(code)
	w.Header().Set("Content-Type", "application/json")
	w.Write(r)
	{{end}}
	{{end}}
}
	`))

	t.Execute(g.w, g.methods)
}

func main() {
	if len(os.Args) < 2 {
		panic("usage: codegen <in>.go")
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, os.Args[1], nil, parser.ParseComments)
	if err != nil {
		panic(err)
	}

	gen := NewGenerator(os.Stdout, "apigen", "apivalidator")

	err = gen.ParseFile(file)
	if err != nil {
		panic(err)
	}

	gen.PrintPackage("someapi", "http", "json")
	gen.PrintHandlers()
	gen.PrintWrappers()
}
