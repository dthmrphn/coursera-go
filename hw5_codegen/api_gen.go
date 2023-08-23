package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func (srv *MyApi) handleProfile(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	params := url.Values{}
	if r.Method == "GET" {
		params = r.URL.Query()
	} else {
		body, _ := io.ReadAll(r.Body)
		params, _ = url.ParseQuery(string(body))
	}

	rvProfileParams, err := NewProfileParams(params)
	if err != nil {
		return nil, err
	}

	return srv.Profile(r.Context(), rvProfileParams)
}

func (srv *MyApi) handleCreate(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	if r.Header.Get("X-Auth") != "100500" {
		return nil, ApiError{http.StatusForbidden, fmt.Errorf("unauthorized")}
	}
	if r.Method != "POST" {
		return nil, ApiError{http.StatusNotAcceptable, fmt.Errorf("bad method")}
	}
	params := url.Values{}
	if r.Method == "GET" {
		params = r.URL.Query()
	} else {
		body, _ := io.ReadAll(r.Body)
		params, _ = url.ParseQuery(string(body))
	}

	rvCreateParams, err := NewCreateParams(params)
	if err != nil {
		return nil, err
	}

	return srv.Create(r.Context(), rvCreateParams)
}

func (srv *OtherApi) handleCreate(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	if r.Header.Get("X-Auth") != "100500" {
		return nil, ApiError{http.StatusForbidden, fmt.Errorf("unauthorized")}
	}
	if r.Method != "POST" {
		return nil, ApiError{http.StatusNotAcceptable, fmt.Errorf("bad method")}
	}
	params := url.Values{}
	if r.Method == "GET" {
		params = r.URL.Query()
	} else {
		body, _ := io.ReadAll(r.Body)
		params, _ = url.ParseQuery(string(body))
	}

	rvOtherCreateParams, err := NewOtherCreateParams(params)
	if err != nil {
		return nil, err
	}

	return srv.Create(r.Context(), rvOtherCreateParams)
}

func (srv *MyApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var out interface{}
	var err error
	var status int

	switch r.URL.Path {
	case "/user/profile":
		out, err = srv.handleProfile(w, r)
	case "/user/create":
		out, err = srv.handleCreate(w, r)
	default:
		err = ApiError{http.StatusNotFound, fmt.Errorf("unknown method")}
	}

	response := struct {
		Err string      `json:"error"`
		Res interface{} `json:"response,omitempty"`
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

func (srv *OtherApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var out interface{}
	var err error
	var status int

	switch r.URL.Path {
	case "/user/create":
		out, err = srv.handleCreate(w, r)
	default:
		err = ApiError{http.StatusNotFound, fmt.Errorf("unknown method")}
	}

	response := struct {
		Err string      `json:"error"`
		Res interface{} `json:"response,omitempty"`
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

func NewCreateParams(values url.Values) (CreateParams, error) {
	rv := CreateParams{}
	var err error = nil

	rv.Login = values.Get("login")
	if rv.Login == "" {
		return rv, ApiError{http.StatusBadRequest, fmt.Errorf("login must me not empty")}
	}
	if len(rv.Login) < 10 {
		return rv, ApiError{http.StatusBadRequest, fmt.Errorf("login len must be >= 10")}
	}

	rv.Name = values.Get("full_name")

	rv.Status = values.Get("status")
	if rv.Status == "" {
		rv.Status = "user"
	}
	enumStatusValid := false
	enumStatus := []string{"user", "moderator", "admin"}
	for _, valid := range enumStatus {
		if valid == rv.Status {
			enumStatusValid = true
			break
		}
	}
	if !enumStatusValid {
		return rv, ApiError{http.StatusBadRequest, fmt.Errorf("status must be one of [%s]", strings.Join(enumStatus, ", "))}
	}

	rv.Age, err = strconv.Atoi(values.Get("age"))
	if err != nil {
		return rv, ApiError{http.StatusBadRequest, fmt.Errorf("age must be int")}
	}
	if rv.Age > 128 {
		return rv, ApiError{http.StatusBadRequest, fmt.Errorf("age must be <= 128")}
	}
	if rv.Age < 0 {
		return rv, ApiError{http.StatusBadRequest, fmt.Errorf("age must be >= 0")}
	}

	return rv, err
}

func NewOtherCreateParams(values url.Values) (OtherCreateParams, error) {
	rv := OtherCreateParams{}
	var err error = nil

	rv.Username = values.Get("username")
	if rv.Username == "" {
		return rv, ApiError{http.StatusBadRequest, fmt.Errorf("username must me not empty")}
	}
	if len(rv.Username) < 3 {
		return rv, ApiError{http.StatusBadRequest, fmt.Errorf("username len must be >= 3")}
	}

	rv.Name = values.Get("account_name")

	rv.Class = values.Get("class")
	if rv.Class == "" {
		rv.Class = "warrior"
	}
	enumClassValid := false
	enumClass := []string{"warrior", "sorcerer", "rouge"}
	for _, valid := range enumClass {
		if valid == rv.Class {
			enumClassValid = true
			break
		}
	}
	if !enumClassValid {
		return rv, ApiError{http.StatusBadRequest, fmt.Errorf("class must be one of [%s]", strings.Join(enumClass, ", "))}
	}

	rv.Level, err = strconv.Atoi(values.Get("level"))
	if err != nil {
		return rv, ApiError{http.StatusBadRequest, fmt.Errorf("level must be int")}
	}
	if rv.Level > 50 {
		return rv, ApiError{http.StatusBadRequest, fmt.Errorf("level must be <= 50")}
	}
	if rv.Level < 1 {
		return rv, ApiError{http.StatusBadRequest, fmt.Errorf("level must be >= 1")}
	}

	return rv, err
}

func NewProfileParams(values url.Values) (ProfileParams, error) {
	rv := ProfileParams{}
	var err error = nil

	rv.Login = values.Get("login")
	if rv.Login == "" {
		return rv, ApiError{http.StatusBadRequest, fmt.Errorf("login must me not empty")}
	}

	return rv, err
}

