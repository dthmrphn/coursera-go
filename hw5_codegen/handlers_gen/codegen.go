package main

import (
	"bytes"
	"fmt"
	"go/format"
	"io"
	"os"
	"text/template"
)

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

func GeneratePackage(out io.Writer, name string, imports ...string) {
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

	t.Execute(out, temp)
}

func GenerateHandlers(out io.Writer, methods Methods) {
	t := template.Must(template.New("ServeTemplate").Parse(`
	{{range $r, $m := .}}
func (srv *{{$r}}) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var data []byte
	var err  error

	switch r.URL.Path {
	{{range $m}}case "{{.URL}}":
		data, err = srv.handle{{.Name}}(w, r)
	{{end}}default:
		err = ApiError{http.StatusNotFound, fmt.Errorf("unknow method")}
	}

	status := http.StatusOK
	
	if err != nil {
		rv := struct{
			Err string ` + "`" + `json:"error"` + "`" + `
		}{}

		if api, ok := err.(ApiError); ok {
			status = api.HTTPStatus 
			rv.Err = api.Error()
			data, err = json.Marshal(rv)
		} else {
			status = http.StatusInternalServerError
		}
	}

	w.WriteHeader(status)
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}
	{{end}}
	`))

	t.Execute(out, methods)
}

func GenerateWrappers(out io.Writer, methods Methods) {
	t := template.Must(template.New("WrapperTemplate").Parse(`
	{{range $r, $m := .}}
	{{range $m}}
func (srv *{{$r}}) handle{{.Name}}(w http.ResponseWriter, r *http.Request) ([]byte, error) {
	{{if .Auth -}}
	if r.Header.Get("X-Auth") != "100500" {
		return nil, ApiError{http.StatusForbidden, fmt.Errorf("unauthorized")}
	}
	{{end -}}

	{{if .Method -}}
	if r.Method != "{{.Method}}" {
		return nil, ApiError{http.StatusNotAcceptable , fmt.Errorf("bad method")}
	}
	{{end -}}

	params := url.Values{}
	if r.Method == "GET" {
		params = r.URL.Query()
	} else {
		body, _ := io.ReadAll(r.Body)
		params, _ = url.ParseQuery(string(body))
	}

	rv{{.Param}}, err := New{{.Param}}(params)
	if err != nil {
		return nil, ApiError{http.StatusBadRequest  , fmt.Errorf("bad method")}
	}

	rv{{.Return}}, err := srv.{{.Name}}(r.Context(), rv{{.Param}})
	if err != nil {
		return nil, err
	}

	response := struct{
		Json *{{.Return}} ` + "`" + `json:"response"` + "`" + `
	}{}

	response.Json = rv{{.Return}}

	rv, err := json.Marshal(response)
	if err != nil {
		return nil, ApiError{http.StatusInternalServerError  , fmt.Errorf("internal error")} 
	}

	return rv, nil
}
	{{end}}
	{{end}}
	`))

	t.Execute(out, methods)
}

func GenerateParams(out io.Writer, params Params) {
	t := template.Must(template.New("ParamTemplate").Parse(`
	{{range $r, $m := .}}
func New{{$r}}(values url.Values) ({{$r}}, error){
	rv := {{$r}}{}
	var err error = nil
	{{range $m}}
	{{- if eq .ParamType "int"}}
	rv.{{.ParamName}}, err = strconv.Atoi(values.Get("{{.ParamValue}}"))
	if err != nil {
		return rv, ApiError{http.StatusBadRequest, fmt.Errorf("{{.ParamValue}} must be int")}
	}
	{{else}}
	rv.{{.ParamName}} = values.Get("{{.ParamValue}}")
	{{end -}}

	{{- if .Default -}}
	if rv.{{.ParamName}} == "" {
		rv.{{.ParamName}} = "{{.Default}}"
	}
	{{end -}}

	{{- if .Required -}}
	if rv.{{.ParamName}} == "" {
		return rv, ApiError{http.StatusBadRequest, fmt.Errorf("{{.ParamValue}} must me not empty")}
	}
	{{end -}}

	{{- if and .Max (eq .ParamType "int") -}}
	if rv.{{.ParamName}} > {{.MaxValue}} {
		return rv, ApiError{http.StatusBadRequest, fmt.Errorf("{{.ParamValue}} must be <= {{.MaxValue}}")}
	}
	{{end -}}

	{{- if and .Min (eq .ParamType "int") -}}
	if rv.{{.ParamName}} < {{.MinValue}} {
		return rv, ApiError{http.StatusBadRequest, fmt.Errorf("{{.ParamValue}} must be >= {{.MinValue}}")}
	}
	{{end -}}

	{{- if and .Max (eq .ParamType "string") -}}
	if len(rv.{{.ParamName}}) > {{.MaxValue}} {
		return rv, ApiError{http.StatusBadRequest, fmt.Errorf("{{.ParamValue}} len must be <= {{.MaxValue}}")}
	}
	{{end -}}

	{{- if and .Min (eq .ParamType "string") -}}
	if len(rv.{{.ParamName}}) < {{.MinValue}} {
		return rv, ApiError{http.StatusBadRequest, fmt.Errorf("{{.ParamValue}} len must be >= {{.MinValue}}")}
	}
	{{end -}}

	{{- if .Enum -}}
	enum{{.ParamName}}Valid := false
	enum{{.ParamName}} := []string{ {{range $i, $e := .Enums}} {{if $i}}, {{end}} "{{$e}}"{{end -}} }
	for _, valid := range enum{{.ParamName}} {
		if valid == rv.{{.ParamName}} {
			  enum{{.ParamName}}Valid = true
			  break
		}
	}
	if !enum{{.ParamName}}Valid {
		return rv, ApiError{http.StatusBadRequest, fmt.Errorf("{{.ParamName}} be one of [%s]", strings.Join(enum{{.ParamName}}, ","))}
	}

	{{end -}}


	{{end}}
	return rv, err
}
	{{end}}
	`))

	t.Execute(out, params)
}

func main() {
	if len(os.Args) < 2 {
		panic("usage: codegen <in>.go")
	}

	var out *os.File
	var err error
	if len(os.Args) == 3 {
		out, err = os.Create(os.Args[2])
		if err != nil {
			panic(err)
		}
	} else {
		out = os.Stdout
	}

	p, err := NewParser(os.Args[1], "apigen", "apivalidator")
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

	GeneratePackage(buf, "main", "net/http", "net/url", "encoding/json", "io", "fmt", "strconv", "strings")
	GenerateWrappers(buf, methods)
	GenerateHandlers(buf, methods)
	GenerateParams(buf, params)

	formated, err := format.Source(buf.Bytes())
	if err != nil {
		panic(err)
	}

	fmt.Fprintln(out, string(formated))
}
