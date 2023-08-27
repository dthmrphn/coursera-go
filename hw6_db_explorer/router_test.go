package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestMatch(t *testing.T) {
	tests := []struct {
		cr      map[string]string
		path    string
		pattern string
		ok      bool
	}{
		{map[string]string{}, "/", "/", true},
		{map[string]string{}, "/", "/table/42", false},
		{map[string]string{"table": "users"}, "/users", "/$table", true},
		{map[string]string{"table": "users", "id": "42"}, "/users/42", "/$table/$id", true},
		{map[string]string{"table": "users", "id": "42"}, "/users/42/delete", "/$table/$id/del", false},
	}

	for i, test := range tests {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			rv, ok := match(test.path, test.pattern)
			if ok != test.ok {
				t.Fatalf("[%s] failed: exp %#v, got %#v\n", t.Name(), test.ok, ok)
			}

			if !reflect.DeepEqual(rv, test.cr) {
				t.Fatalf("[%s] failed: exp %#v, got %#v\n", t.Name(), test.cr, rv)
			}
		})
	}
}

func newRequest(method, url string, body any) (*http.Request, error) {
	var req *http.Request
	var err error
	var data []byte

	if method == http.MethodGet {
		req, err = http.NewRequest(method, url, nil)
		if err != nil {
			return nil, err
		}
	} else {
		data, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}

		req, err = http.NewRequest(method, url, bytes.NewReader(data))
		if err != nil {
			return nil, err
		}

		req.Header.Add("Content-Type", "application/json")
	}

	return req, nil
}

func TestRouter(t *testing.T) {
	stubTables := func(w http.ResponseWriter, r *http.Request) {
		tables := []string{"users", "items", "products"}
		fmt.Fprintf(w, "list of tables: [%s]", strings.Join(tables, ", "))
	}

	stubRecord := func(w http.ResponseWriter, r *http.Request) {
		table := RouteParam(r, "table")
		id := RouteParam(r, "id")
		fmt.Fprintf(w, "record [%s] of table [%s]", id, table)
	}

	stubRecords := func(w http.ResponseWriter, r *http.Request) {
		limit := URLParam(r, "limit")
		offset := URLParam(r, "offset")
		table := RouteParam(r, "table")
		fmt.Fprintf(w, "records of table [%s] at [%s:%s]", table, offset, limit)
	}

	stubCreate := func(w http.ResponseWriter, r *http.Request) {
		table := RouteParam(r, "table")
		user_id := JSONParam(r, "user_id")
		fmt.Fprintf(w, "creating record of table [%s] with 'user_id=%s'", table, user_id)
	}

	r := NewRouter()
	r.Route("GET", "/", stubTables)
	r.Route("GET", "/$table", stubRecords)
	r.Route("GET", "/$table/$id", stubRecord)
	r.Route("PUT", "/$table", stubCreate)

	tests := []struct {
		method string
		path   string
		query  string
		body   map[string]string
		resp   string
	}{
		{method: http.MethodGet, path: "/", resp: "list of tables: [users, items, products]"},
		{method: http.MethodGet, path: "/users", resp: "records of table [users] at [5:10]", query: "?limit=10&offset=5"},
		{method: http.MethodGet, path: "/users/42", resp: "record [42] of table [users]"},
		{method: http.MethodPut, path: "/items", body: map[string]string{"user_id": "42"}, resp: "creating record of table [items] with 'user_id=42'"},
	}

	srv := httptest.NewServer(http.HandlerFunc(r.Serve))
	cli := &http.Client{}

	for i, test := range tests {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			req, err := newRequest(test.method, srv.URL+test.path+test.query, test.body)
			if err != nil {
				t.Fatalf("request error at case[%s]: %s\n", t.Name(), err.Error())
			}

			resp, err := cli.Do(req)
			if err != nil {
				t.Fatalf("response error at case[%s]: %s\n", t.Name(), err.Error())
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("body read error at case[%s]: %s\n", t.Name(), err.Error())
			}

			bodystr := string(body)
			if bodystr != test.resp {
				t.Fatalf("case[%s]: exp: [%s], got: [%s]\n", t.Name(), test.resp, bodystr)
			}
		})
	}

}
