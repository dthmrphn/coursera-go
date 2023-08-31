package main

import (
	"fmt"
	"io"
	"os"
	"path"
)

func format(dir os.DirEntry) string {
	if !dir.IsDir() {
		info, err := dir.Info()
		if err != nil {
			return ""
		}

		if info.Size() == 0 {
			return fmt.Sprintf("%s (empty)", dir.Name())
		} else {
			return fmt.Sprintf("%s (%db)", dir.Name(), info.Size())
		}
	}

	return dir.Name()
}

func entries(dir string, files bool) ([]os.DirEntry, error) {
	var rv []os.DirEntry

	dirs, err := os.ReadDir(dir)
	if err != nil {
		return rv, err
	}

	for _, d := range dirs {
		if d.IsDir() == false && files == false {
			continue
		}
		rv = append(rv, d)
	}

	return rv, nil
}

func helper(out io.Writer, dir string, files bool, prefix string) error {
	entries, err := entries(dir, files)
	if err != nil {
		return err
	}

	for i, e := range entries {
		subpath := path.Join(dir, e.Name())

		if i == len(entries)-1 {
			fmt.Fprintf(out, "%s└───%s\n", prefix,  format(e))
			helper(out, subpath, files, prefix+"\t")
		} else {
			fmt.Fprintf(out, "%s├───%s\n", prefix,  format(e))
			helper(out, subpath, files, prefix+"│\t")
		}
	}

	return nil
}

func dirTree(out io.Writer, path string, files bool) error {
	err := helper(out, path, files, "")
	if err != nil {
		return err
	}

	return nil
}

func main() {
	out := os.Stdout
	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		panic("usage go run main.go . [-f]")
	}
	path := os.Args[1]
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"
	err := dirTree(out, path, printFiles)
	if err != nil {
		panic(err.Error())
	}
}
