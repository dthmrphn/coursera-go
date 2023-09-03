package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const AccessToken string = "accesstoken"
const DataSetPath string = "dataset.xml"

func TestTimeout(t *testing.T) {
	h := func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Second * 2)
	}

	srv := httptest.NewServer(http.HandlerFunc(h))
	cli := SearchClient{
		AccessToken: AccessToken,
		URL:         srv.URL,
	}

	_, err := cli.FindUsers(SearchRequest{})
	assert.Equal(t, err.Error(), "timeout for limit=1&offset=0&order_by=0&order_field=&query=")

}

func TestBrokenJson(t *testing.T) {
	h := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("json"))
	}

	srv := httptest.NewServer(http.HandlerFunc(h))
	cli := SearchClient{
		AccessToken: AccessToken,
		URL:         srv.URL,
	}

	_, err := cli.FindUsers(SearchRequest{})
	assert.Equal(t, err.Error(), "cant unpack result json: invalid character 'j' looking for beginning of value")

	h = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("json"))
	}

	srv = httptest.NewServer(http.HandlerFunc(h))
	cli = SearchClient{
		AccessToken: AccessToken,
		URL:         srv.URL,
	}

	_, err = cli.FindUsers(SearchRequest{})
	assert.Equal(t, err.Error(), "cant unpack error json: invalid character 'j' looking for beginning of value")
}

func TestInternalError(t *testing.T) {
	h := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}

	srv := httptest.NewServer(http.HandlerFunc(h))
	cli := SearchClient{
		AccessToken: AccessToken,
		URL:         srv.URL,
	}

	_, err := cli.FindUsers(SearchRequest{})
	assert.Equal(t, err.Error(), "SearchServer fatal error")
}

func TestAccessToken(t *testing.T) {
	ss, _ := NewServer(AccessToken, DataSetPath)

	srv := httptest.NewServer(ss)
	cli := SearchClient{
		AccessToken: "BadAccessToken",
		URL:         srv.URL,
	}

	_, err := cli.FindUsers(SearchRequest{})
	assert.Equal(t, err.Error(), "Bad AccessToken")
}

func TestUnknownError(t *testing.T) {
	cli := SearchClient{
		AccessToken: AccessToken,
		URL:         "http://somewhere",
	}

	_, err := cli.FindUsers(SearchRequest{})
	assert.Equal(t, true, strings.HasPrefix(err.Error(), "unknown error Get \"http://somewhere"))
}

func TestSearchParams(t *testing.T) {
	ss, e := NewServer(AccessToken, DataSetPath)
	assert.NotNil(t, ss)
	assert.NoError(t, e)

	srv := httptest.NewServer(ss)

	cli := &SearchClient{
		AccessToken: AccessToken,
		URL:         srv.URL,
	}

	tests := []struct {
		n string
		r SearchRequest
		e string
	}{
		{"WrongLimit", SearchRequest{Limit: -1}, "limit must be > 0"},
		{"WrongOffset", SearchRequest{Offset: -1}, "offset must be > 0"},
		{"WrongOrder", SearchRequest{OrderField: "any"}, "OrderField any invalid"},
		{"WrongOrderBy", SearchRequest{OrderBy: 2}, "unknown bad request error: should be [-1:1]"},
	}

	for _, test := range tests {
		t.Run(test.n, func(t *testing.T) {
			_, err := cli.FindUsers(test.r)
			assert.Equal(t, test.e, err.Error())
		})
	}

	users, err := cli.FindUsers(SearchRequest{Limit: 100})
	assert.NoError(t, err)
	assert.Equal(t, 25, len(users.Users))

	users, err = cli.FindUsers(SearchRequest{Limit: 5})
	assert.NoError(t, err)
	assert.Equal(t, 5, len(users.Users))
}

func TestSearchUsers(t *testing.T) {
	ss, e := NewServer(AccessToken, "dataset.xml")
	assert.NotNil(t, ss)
	assert.NoError(t, e)

	srv := httptest.NewServer(ss)
	cli := SearchClient{
		AccessToken: AccessToken,
		URL:         srv.URL,
	}

	tests := []struct {
		r SearchRequest
		u []User
	}{
		{SearchRequest{Limit: 4, OrderField: "Id", OrderBy: OrderByDesc}, []User{ss.users[0], ss.users[1], ss.users[2], ss.users[3]}},
		{SearchRequest{Limit: 4, OrderField: "Id", OrderBy: OrderByAsc}, []User{ss.users[4], ss.users[3], ss.users[2], ss.users[1]}},
		{SearchRequest{Limit: 4, OrderField: "Age", OrderBy: OrderByDesc}, []User{ss.users[1], ss.users[0], ss.users[2], ss.users[3]}},
		{SearchRequest{Limit: 4, OrderField: "Age", OrderBy: OrderByAsc}, []User{ss.users[4], ss.users[3], ss.users[2], ss.users[0]}},
		{SearchRequest{Limit: 4, OrderField: "Name", OrderBy: OrderByDesc}, []User{ss.users[0], ss.users[2], ss.users[3], ss.users[1]}},
		{SearchRequest{Limit: 4, OrderField: "Name", OrderBy: OrderByAsc}, []User{ss.users[4], ss.users[1], ss.users[3], ss.users[2]}},
		{SearchRequest{Limit: 4, OrderField: "", OrderBy: OrderByAsc}, []User{ss.users[4], ss.users[1], ss.users[3], ss.users[2]}},
		{SearchRequest{Limit: 4, OrderField: "", OrderBy: OrderByAsc, Query: "uSeR"}, []User{}},
	}

	for i, test := range tests {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			users, err := cli.FindUsers(test.r)
			assert.Nil(t, err)
			assert.Equal(t, test.u, users.Users)
		})
	}
}
