package main

import (
	"io"
	"text/template"
)

type Generator struct {
	w io.Writer
}

func (g Generator) DoPackage(name string, imports ...string) {
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

func (g Generator) DoHandlers(methods Methods) {
	t := template.Must(template.New("ServeTemplate").Parse(`
	{{range $r, $m := .}}
func (srv *{{$r}}) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var out interface{}
	var err  error
	var status int

	switch r.URL.Path {
	{{range $m}}case "{{.URL}}":
		out, err = srv.handle{{.Name}}(w, r)
	{{end}}default:
		err = ApiError{http.StatusNotFound, fmt.Errorf("unknown method")}
	}

	response := struct{
		Err string 	` + "`" + `json:"error"` + "`" + `
		Res interface{} ` + "`" + `json:"response,omitempty"` + "`" + ` 
	}{}

	if err != nil {
		if api, ok := err.(ApiError); ok {
			response.Err = api.Error()
			status = api.HTTPStatus
		} else {
			response.Err = err.Error()
			status = http.StatusInternalServerError
		}
	} else {
		response.Res = out
		status = http.StatusOK
	}

	js, _ := json.Marshal(response)
	w.WriteHeader(status)
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}
	{{end}}
	`))

	t.Execute(g.w, methods)
}

func (g Generator) DoWrappers(methods Methods) {
	t := template.Must(template.New("WrapperTemplate").Parse(`
	{{range $r, $m := .}}
	{{range $m}}
func (srv *{{$r}}) handle{{.Name}}(w http.ResponseWriter, r *http.Request) (interface{}, error) {
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
		return nil, err
	}

	return srv.{{.Name}}(r.Context(), rv{{.Param}})
}
	{{end}}
	{{end}}
	`))

	t.Execute(g.w, methods)
}

func (g Generator) DoParams(params Params) {
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
		return rv, ApiError{http.StatusBadRequest, fmt.Errorf("{{.ParamValue}} must be one of [%s]", strings.Join(enum{{.ParamName}}, ", "))}
	}

	{{end -}}

	{{end}}
	return rv, err
}
	{{end}}
	`))

	t.Execute(g.w, params)
}
