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

func (a analyzer) processtree(path string, entry os.FileInfo, err error) error {
	if err != nil {
	    return err
	}
	if entry.IsDir() {
		//fmt.Println("dir " + path, entry.Size())
	} else {
		//fmt.Println("file " + path, entry.Size())
		a.pathmap[path] = "woo"
	}
	return nil	
}

func NewAnalyzer(paths []string) (*analyzer, error) {
	a := analyzer{paths: paths}

	a.pathmap = make(map[string]string)
	a.hashmap = make(map[string]string)

	for _, element := range paths {
		processfunc := func(path string, fileInfo os.FileInfo, e error) (err error) {
			e = a.processtree(path, fileInfo, e)
			if e != nil {
			    log.Println(err)
			    return e
			}
			return nil
		}
		err := filepath.Walk(element, processfunc)
		if err != nil {
		    log.Println(err)
		    return nil, err
		}
	}
	// do more synthesis here
	fmt.Println(a)
	return &a, nil
}

func main() {
	var a *analyzer
	var err error
	a, err = NewAnalyzer(os.Args[1:])
	if err != nil {
	    log.Println(err)
	}
	log.Println(a.paths)
}