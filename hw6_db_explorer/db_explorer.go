package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

type DbExplorer struct {
	r  *Router
	v  *Validator
	db *sql.DB
}

func NewDbExplorer(db *sql.DB) (*DbExplorer, error) {
	v, err := NewValidator(db)
	if err != nil {
		return nil, err
	}

	r := NewRouter()

	e := &DbExplorer{r, v, db}

	r.Route("GET", "/", e.tablesList)
	r.Route("GET", "/$table", e.recordsList)
	r.Route("GET", "/$table/$id", e.recordInfo)
	r.Route("PUT", "/$table/", e.recordCreate)
	r.Route("POST", "/$table/$id", e.recordUpdate)
	r.Route("DELETE", "/$table/$id", e.recordDelete)

	return e, nil
}

func JSONParams(r *http.Request) (map[string]interface{}, error) {
	rv := map[string]interface{}{}
	err := json.NewDecoder(r.Body).Decode(&rv)
	if err != nil {
		return nil, err
	}

	return rv, nil
}

func writeResponse(w http.ResponseWriter, status int, data any, err string) {
	resp := struct {
		Error string      `json:"error,omitempty"`
		Data  interface{} `json:"response,omitempty"`
	}{err, data}

	js, _ := json.Marshal(resp)
	w.WriteHeader(status)
	w.Header().Add("Content-Type", "application/json")
	w.Write(js)
}

func (e *DbExplorer) tablesList(w http.ResponseWriter, r *http.Request) {
	tables := []string{}

	for t := range e.v.tables {
		tables = append(tables, t)
	}

	sort.Strings(tables)

	data := map[string][]string{"tables": tables}
	writeResponse(w, http.StatusOK, data, "")
}

func (e *DbExplorer) recordsList(w http.ResponseWriter, r *http.Request) {
	tname := RouteParam(r, "table")
	table, ok := e.v.tables[tname]
	if !ok {
		writeResponse(w, http.StatusNotFound, nil, "unknown table")
		return
	}

	limit, err := strconv.Atoi(URLParam(r, "limit"))
	if err != nil {
		limit = 5
	}

	offset, err := strconv.Atoi(URLParam(r, "offset"))
	if err != nil {
		offset = 0
	}

	rows, err := e.db.Query("SELECT * FROM "+tname+" LIMIT ? OFFSET ?", limit, offset)
	if err != nil {
		writeResponse(w, http.StatusInternalServerError, nil, "")
		return
	}
	defer rows.Close()

	records := []interface{}{}
	for rows.Next() {
		row := table.NewRow()
		err := rows.Scan(row...)
		if err != nil {
			writeResponse(w, http.StatusInternalServerError, nil, "")
			return
		}

		record := map[string]interface{}{}
		for i, c := range table.cols {
			record[c.Name()] = row[i]
		}

		records = append(records, record)
	}

	data := map[string]interface{}{"records": records}
	writeResponse(w, http.StatusOK, data, "")
}

func (e *DbExplorer) recordInfo(w http.ResponseWriter, r *http.Request) {
	tname := RouteParam(r, "table")
	table, ok := e.v.tables[tname]
	if !ok {
		writeResponse(w, http.StatusNotFound, nil, "unknown table")
		return
	}

	id, err := strconv.Atoi(RouteParam(r, "id"))
	if err != nil {
		writeResponse(w, http.StatusNotFound, nil, "bad id param")
		return
	}

	rows, err := e.db.Query("SELECT * FROM "+tname+" WHERE "+table.key+" = ?", id)
	if err != nil {
		fmt.Println(err)
		writeResponse(w, http.StatusInternalServerError, nil, "")
		return
	}
	defer rows.Close()

	record := map[string]interface{}{}
	for rows.Next() {
		row := table.NewRow()
		err := rows.Scan(row...)
		if err != nil {
			writeResponse(w, http.StatusInternalServerError, nil, "")
			return
		}

		for i, c := range table.cols {
			record[c.Name()] = row[i]
		}
	}

	if len(record) == 0 {
		writeResponse(w, http.StatusNotFound, nil, "record not found")
		return
	}

	data := map[string]interface{}{"record": record}
	writeResponse(w, http.StatusOK, data, "")
}

func (e *DbExplorer) recordCreate(w http.ResponseWriter, r *http.Request) {
	tname := RouteParam(r, "table")
	table, ok := e.v.tables[tname]
	if !ok {
		writeResponse(w, http.StatusNotFound, nil, "unknown table")
		return
	}

	params, err := JSONParams(r)
	if err != nil {
		writeResponse(w, http.StatusBadRequest, nil, err.Error())
		return
	}

	vals := make([]interface{}, 0, len(params))
	cols := make([]string, 0, len(params))
	plhd := make([]string, 0, len(params))
	for _, c := range table.cols {
		if param, ok := params[c.Name()]; ok {
			if c.Name() == table.key {
				continue
			}
			if !c.Equal(param) {
				writeResponse(w, http.StatusBadRequest, nil, fmt.Sprintf("field %s have invalid type", c.Name()))
				return
			}
			vals = append(vals, param)
		} else {
			vals = append(vals, c.Type())
		}

		cols = append(cols, c.Name())
		plhd = append(plhd, "?")
	}

	q := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", tname, strings.Join(cols, ", "), strings.Join(plhd, ", "))
	res, err := e.db.Exec(q, vals...)
	if err != nil {
		writeResponse(w, http.StatusInternalServerError, nil, err.Error())
		return
	}

	id, err := res.LastInsertId()
	if err != nil {
		writeResponse(w, http.StatusInternalServerError, nil, err.Error())
		return
	}

	data := map[string]int64{table.key: id}
	writeResponse(w, http.StatusOK, data, "")
}

func (e *DbExplorer) recordUpdate(w http.ResponseWriter, r *http.Request) {
	tname := RouteParam(r, "table")
	id := RouteParam(r, "id")
	table, ok := e.v.tables[tname]
	if !ok {
		writeResponse(w, http.StatusNotFound, nil, "unknown table")
		return
	}

	params, err := JSONParams(r)
	if err != nil {
		writeResponse(w, http.StatusBadRequest, nil, err.Error())
		return
	}

	vals := make([]interface{}, 0, len(params))
	cols := make([]string, 0, len(params))
	for _, c := range table.cols {
		if param, ok := params[c.Name()]; ok {
			if c.Name() == table.key {
				writeResponse(w, http.StatusBadRequest, nil, fmt.Sprintf("field %s have invalid type", c.Name()))
				return
			}
			if !c.Equal(param) {
				writeResponse(w, http.StatusBadRequest, nil, fmt.Sprintf("field %s have invalid type", c.Name()))
				return
			}
			vals = append(vals, param)
			cols = append(cols, fmt.Sprintf("%s = ?", c.Name()))
		}
	}

	q := fmt.Sprintf("UPDATE %s SET %s WHERE %s = %s", tname, strings.Join(cols, ", "), table.key, id)
	res, err := e.db.Exec(q, vals...)
	if err != nil {
		writeResponse(w, http.StatusInternalServerError, nil, err.Error())
		return
	}

	row, err := res.RowsAffected()
	if err != nil {
		writeResponse(w, http.StatusInternalServerError, nil, err.Error())
		return
	}

	data := map[string]int64{"updated": row}
	writeResponse(w, http.StatusOK, data, "")
}

func (e *DbExplorer) recordDelete(w http.ResponseWriter, r *http.Request) {
	tname := RouteParam(r, "table")
	id := RouteParam(r, "id")
	table, ok := e.v.tables[tname]
	if !ok {
		writeResponse(w, http.StatusNotFound, nil, "unknown table")
		return
	}

	q := fmt.Sprintf("DELETE FROM %s WHERE %s = ?", tname, table.key)
	res, err := e.db.Exec(q, id)
	if err != nil {
		writeResponse(w, http.StatusInternalServerError, nil, err.Error())
		return
	}

	del, err := res.RowsAffected()
	if err != nil {
		writeResponse(w, http.StatusInternalServerError, nil, err.Error())
		return
	}

	data := map[string]int64{"deleted": del}
	writeResponse(w, http.StatusOK, data, "")

}

func (e *DbExplorer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	e.r.Serve(w, r)
}
