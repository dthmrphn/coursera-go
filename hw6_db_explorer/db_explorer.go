package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
)

type DbExplorer struct {
	r  *Router
	db *sql.DB

	tables []Table
}

type DbResponse struct {
	Error string      `json:"error,omitempty"`
	Data  interface{} `json:"response,omitempty"`
}

type Column struct{}

type Table struct {
	Name string
	Key  string
	Cols []Column
}

func getTables(db *sql.DB) ([]Table, error) {
	rows, err := db.Query("SHOW TABLES")
	if err != nil {
		return nil, err
	}

	rv := make([]Table, 0)

	for rows.Next() {
		t := Table{}
		err := rows.Scan(&t.Name)
		if err != nil {
			return nil, err
		}
		rv = append(rv, t)
	}

	return rv, nil
}

func getColumns(db *sql.DB, name string) ([]Column, error) {
	cols, err := db.Query("SHOW COLUMNS FROM " + name)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	for cols.Next() {
		Field := ""
		Type := ""
		Null := ""
		Key := ""
		Default := ""
		Extra := ""

		cols.Scan(&Field, &Type, &Null, &Key, &Default, &Extra)
		fmt.Println(Field, Type, Null, Key, Default, Extra)
	}

	return nil, nil
}

func NewDbExplorer(db *sql.DB) (*DbExplorer, error) {
	e := &DbExplorer{r: NewRouter(), db: db}

	tables, err := getTables(db)
	if err != nil {
		return nil, err
	}

	for _, t := range tables {
		getColumns(db, t.Name)
	}

	e.tables = tables

	e.r.Route("GET", "/", e.handleGetTables)
	e.r.Route("GET", "/$table", e.handleGetRecords)
	e.r.Route("GET", "/$table/$id", e.handlePostRecord)

	return e, nil
}

func (e *DbExplorer) handleGetTables(w http.ResponseWriter, r *http.Request) {
	rv := struct {
		Tables []string `json:"tables"`
	}{}

	rv.Tables = make([]string, len(e.tables))
	for i, t := range e.tables {
		rv.Tables[i] = t.Name
	}

	resp := DbResponse{Data: rv}

	js, _ := json.Marshal(resp)

	w.WriteHeader(http.StatusOK)
	w.Header().Add("Content-Type", "application/json")
	w.Write(js)
}

func (e *DbExplorer) handleGetRecord(w http.ResponseWriter, r *http.Request)  {}
func (e *DbExplorer) handleGetRecords(w http.ResponseWriter, r *http.Request) {}
func (e *DbExplorer) handlePutRecord(w http.ResponseWriter, r *http.Request)  {}
func (e *DbExplorer) handlePostRecord(w http.ResponseWriter, r *http.Request) {}
func (e *DbExplorer) handleDelRecord(w http.ResponseWriter, r *http.Request)  {}

func (e *DbExplorer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	e.r.Serve(w, r)
}
