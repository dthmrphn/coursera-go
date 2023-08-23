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

func (srv *MyApi) handleProfile(w http.ResponseWriter, r *http.Request) ([]byte, error) {
	params := url.Values{}
	if r.Method == "GET" {
		params = r.URL.Query()
	} else {
		body, _ := io.ReadAll(r.Body)
		params, _ = url.ParseQuery(string(body))
	}

	rvProfileParams, err := NewProfileParams(params)
	if err != nil {
		return nil, ApiError{http.StatusBadRequest, fmt.Errorf("bad method")}
	}

	rvUser, err := srv.Profile(r.Context(), rvProfileParams)
	if err != nil {
		return nil, err
	}

	response := struct {
		Json *User `json:"response"`
	}{}

	response.Json = rvUser

	rv, err := json.Marshal(response)
	if err != nil {
		return nil, ApiError{http.StatusInternalServerError, fmt.Errorf("internal error")}
	}

	return rv, nil
}

func (srv *MyApi) handleCreate(w http.ResponseWriter, r *http.Request) ([]byte, error) {
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
		return nil, ApiError{http.StatusBadRequest, fmt.Errorf("bad method")}
	}

	rvNewUser, err := srv.Create(r.Context(), rvCreateParams)
	if err != nil {
		return nil, err
	}

	response := struct {
		Json *NewUser `json:"response"`
	}{}

	response.Json = rvNewUser

	rv, err := json.Marshal(response)
	if err != nil {
		return nil, ApiError{http.StatusInternalServerError, fmt.Errorf("internal error")}
	}

	return rv, nil
}

func (srv *OtherApi) handleCreate(w http.ResponseWriter, r *http.Request) ([]byte, error) {
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
		return nil, ApiError{http.StatusBadRequest, fmt.Errorf("bad method")}
	}

	rvOtherUser, err := srv.Create(r.Context(), rvOtherCreateParams)
	if err != nil {
		return nil, err
	}

	response := struct {
		Json *OtherUser `json:"response"`
	}{}

	response.Json = rvOtherUser

	rv, err := json.Marshal(response)
	if err != nil {
		return nil, ApiError{http.StatusInternalServerError, fmt.Errorf("internal error")}
	}

	return rv, nil
}

func (srv *MyApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var data []byte
	var err error

	switch r.URL.Path {
	case "/user/profile":
		data, err = srv.handleProfile(w, r)
	case "/user/create":
		data, err = srv.handleCreate(w, r)
	default:
		err = ApiError{http.StatusNotFound, fmt.Errorf("unknow method")}
	}

	status := http.StatusOK

	if err != nil {
		rv := struct {
			Err string `json:"error"`
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

func (srv *OtherApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var data []byte
	var err error

	switch r.URL.Path {
	case "/user/create":
		data, err = srv.handleCreate(w, r)
	default:
		err = ApiError{http.StatusNotFound, fmt.Errorf("unknow method")}
	}

	status := http.StatusOK

	if err != nil {
		rv := struct {
			Err string `json:"error"`
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

func NewCreateParams(values url.Values) (CreateParams, error) {
	rv := CreateParams{}
	var err error = nil

	rv.Login = values.Get("login")
	if rv.Login == "" {
		return rv, ApiError{http.StatusBadRequest, fmt.Errorf("login must me not empty")}
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
		return rv, ApiError{http.StatusBadRequest, fmt.Errorf("Status be one of [%s]", strings.Join(enumStatus, ","))}
	}

	rv.Age, err = strconv.Atoi(values.Get("age"))
	if err != nil {
		return rv, ApiError{http.StatusBadRequest, fmt.Errorf("age must be int")}
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
		return rv, ApiError{http.StatusBadRequest, fmt.Errorf("Class be one of [%s]", strings.Join(enumClass, ","))}
	}

	rv.Level, err = strconv.Atoi(values.Get("level"))
	if err != nil {
		return rv, ApiError{http.StatusBadRequest, fmt.Errorf("level must be int")}
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

