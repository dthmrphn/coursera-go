package main

import (
	"coursera/hw3_bench/fast"
	"io"
	"os"
)

func FastSearch(out io.Writer) {
	data, err := os.ReadFile(filePath)

	if err != nil {
		panic(err)
	}

	fast.Default(out, data)
}

func FastSearchEasyJson(out io.Writer) {
	data, err := os.ReadFile(filePath)

	if err != nil {
		panic(err)
	}

	fast.EasyJson(out, data)
}

