package main

import (  
    "fmt"
    "os"
    "log"
    "path/filepath"
)

type analyzer struct {
	paths []string
	pathmap map[string]string
	hashmap map[string]string
}

func processentry(path string, entry os.FileInfo, err error) error {
	if err != nil {
	    return err
	}
	if entry.IsDir() {
		fmt.Println("dir " + path, entry.Size())
	} else {
		fmt.Println("file " + path, entry.Size())
	}
	return nil	
}

func NewAnalyzer(paths []string) (*analyzer, error) {
	a := analyzer{paths: paths }

	for _, element := range paths {
		err := filepath.Walk(element, processentry)
		if err != nil {
		    log.Println(err)
		    return nil, err
		}
	}
	return &a, nil
}

func main() {
	a, err := NewAnalyzer(os.Args[1:])
	if err != nil {
	    log.Println(err)
	}
	log.Println(a.paths)
}