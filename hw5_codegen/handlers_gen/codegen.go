package main

import (
	"io"
	"os"
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

	methods map[string][]MethodTemplate
	structs map[string]StructTemplate
}

func NewGenerator(w io.Writer) Generator {
	return Generator{
		w:         w,
		methods:   map[string][]MethodTemplate{},
		structs:   map[string]StructTemplate{},
	}
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

	g := NewGenerator(os.Stdout)
	p, err := NewParser(os.Args[1], "apigen", "apivalidator")
	if err != nil {
		panic(err)
	}

	p.ParseStructs()

	g.PrintPackage("someapi", "http", "json")
	g.PrintHandlers()
	g.PrintWrappers()
}
