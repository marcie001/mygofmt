package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"regexp"

	"golang.org/x/tools/go/loader"
	"golang.org/x/tools/imports"
)

var (
	reg = regexp.MustCompile(`(\{\n{2,}|\n{2,}\})`)
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: mygofmt [flags] [path ...]\n")
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage

	var d, w bool
	flag.BoolVar(&d, "d", false, "display diffs instead of rewriting files")
	flag.BoolVar(&w, "w", false, "write result to (source) file instead of stdout")
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		usage()
		os.Exit(1)
	}

	if errs := run(w, args); len(errs) > 0 {
		for _, err := range errs {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}
}

func run(writeToFile bool, files []string) []error {
	conf := &loader.Config{
		ParserMode: parser.ParseComments,
	}

	m := make(map[string][]byte)
	var errs []error
	for _, file := range files {
		code, err := processFile(conf, file)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		m[file] = code
	}

	if len(errs) > 0 {
		return errs
	}

	for k, v := range m {
		if writeToFile {
			err := ioutil.WriteFile(k, v, 0666)
			if err != nil {
				errs = append(errs, err)
			}
		} else {
			fmt.Println("// File: ", k)
			fmt.Println(string(v))
		}
	}

	return errs
}

func processFile(conf *loader.Config, file string) ([]byte, error) {
	astFile, err := conf.ParseFile(file, nil)
	if err != nil {
		return nil, err
	}

	tokenFile := conf.Fset.File(astFile.Pos())

	// import の内側の改行を削除する
	for _, v := range astFile.Imports {
		tokenFile.MergeLine(tokenFile.Line(v.End()))
	}

	ast.Inspect(astFile, func(n ast.Node) bool {
		if bl, ok := n.(*ast.BlockStmt); ok {
			rmEmptyRowInsideBrace(tokenFile, bl)
		}
		return true
	})

	buf := &bytes.Buffer{}
	format.Node(buf, conf.Fset, astFile)
	return imports.Process(file, buf.Bytes(), nil)
}

// rmEmptyRow はブロックの内側の空行を削除する
func rmEmptyRowInsideBrace(f *token.File, bl *ast.BlockStmt) {
	f.MergeLine(f.Line(bl.Pos()))
	f.MergeLine(f.Line(bl.End()) - 1)
}
