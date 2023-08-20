package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSearchServer(t *testing.T) {
	ss, err := NewServer(AccessToken, "somewhere")
	assert.Nil(t, ss)
	assert.Error(t, err)

	ss, err = NewServer(AccessToken, "server.go")
	assert.Nil(t, ss)
	assert.Error(t, err)

	ss, err = NewServer(AccessToken, DataSetPath)
	assert.NotNil(t, ss)
	assert.Nil(t, err)
}

func TestSearchRequest(t *testing.T) {
	ss, err := NewServer(AccessToken, DataSetPath)
	assert.NotNil(t, ss)
	assert.Nil(t, err)

	tests := []struct {
		req string
		err string
	}{
		{"%%%%%%%!!!!!!!!!!", "could not parse query: query is wrong"},
		{"limit=-1&offset=0&order_by=0", "limit is negative: wrong value"},
		{"limit=1&offset=-1&order_by=0", "offset is negative: wrong value"},
		{"limit=1&offset=1000&order_by=0", "offset is too big: wrong value"},
		{"limit=1&offset=-1&order_by=\"str\"", "order_by: couldnt cast s to i"},
		{"limit=\"str\"&offset=-1&order_by=0", "limit: couldnt cast s to i"},
		{"limit=1&order_by=0", "offset: field missed"},
	}

	for i, test := range tests {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			_, err := ss.SearchRequest(test.req)
			assert.Equal(t, test.err, err.Error())
		})
	}
}
