package main

import (
	"database/sql"
	"fmt"
	"strings"
)

type column interface {
	Name() string
	Equal(val interface{}) bool
	Type() interface{}
}

type columnInt struct {
	null bool
	name string
}

func (c *columnInt) Name() string {
	return c.name
}

func (c *columnInt) Equal(val interface{}) bool {
	if val == nil {
		return c.null
	}

	_, ok := val.(int)
	return ok
}

func (c *columnInt) Type() interface{} {
	if c.null {
		return new(*int)
	}

	return new(int)
}

type columnString struct {
	null bool
	name string
}

func (c *columnString) Name() string {
	return c.name
}

func (c *columnString) Equal(val interface{}) bool {
	if val == nil {
		return c.null
	}

	_, ok := val.(string)
	return ok
}

func (c *columnString) Type() interface{} {
	if c.null {
		return new(*string)
	}

	return new(string)
}

type table struct {
	key  string
	cols []column
}

func (t table) NewRow() []interface{} {
	rv := make([]interface{}, 0)

	for _, c := range t.cols {
		rv = append(rv, c.Type())
	}

	return rv
}

type Validator struct {
	tables map[string]table
}

func NewValidator(db *sql.DB) (*Validator, error) {
	names, err := getTablesNames(db)
	if err != nil {
		return nil, err
	}

	tables := map[string]table{}
	for _, name := range names {
		cols, key, err := getTableColumns(db, name)
		if err != nil {
			return nil, err
		}

		tables[name] = table{key, cols}
	}

	return &Validator{tables}, nil
}

func getTablesNames(db *sql.DB) ([]string, error) {
	tables, err := db.Query("SHOW TABLES")
	if err != nil {
		return nil, err
	}
	defer tables.Close()

	rv := make([]string, 0)
	t := ""

	for tables.Next() {
		err := tables.Scan(&t)
		if err != nil {
			return nil, err
		}
		rv = append(rv, t)
	}

	return rv, nil
}

func getTableColumns(db *sql.DB, name string) ([]column, string, error) {
	cols, err := db.Query("SHOW COLUMNS FROM " + name)
	if err != nil {
		fmt.Println(err)
		return nil, "", err
	}

	rv := []column{}

	var Field, Type, Null, Key, Extra, key string
	var Default interface{}
	for cols.Next() {
		err := cols.Scan(&Field, &Type, &Null, &Key, &Default, &Extra)
		if err != nil {
			return nil, "", err
		}

		null := Null == "YES"
		if strings.HasPrefix(Type, "int") {
			rv = append(rv, &columnInt{null, Field})
		} else {
			rv = append(rv, &columnString{null, Field})
		}

		if Key == "PRI" {
			key = Field
		}
	}

	return rv, key, nil
}
