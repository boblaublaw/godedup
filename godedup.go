package main

import (  
    "fmt"
    "io"
    "os"
    "log"
    "errors"
    "path/filepath"
    "crypto/sha1"
    "encoding/hex"
)

type analyzer struct {
	paths []string
	pathmap map[string]string
	hashmap map[string]string
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
	for k, v := range a.pathmap {
		fmt.Println(v + " " + k)
	}
	return &a, nil
}

func (a analyzer) hashfile(path string) (error) {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	a.pathmap[path] = hex.EncodeToString(h.Sum(nil))
	return nil
}

func (a analyzer) processtree(path string, entry os.FileInfo, err error) error {
	if err != nil {
	    return err
	}
	if entry.IsDir() {
		//fmt.Println("dir " + path, entry.Size())
	} else if entry.Mode().IsRegular() {
		//fmt.Println("file " + path, entry.Size())
		e := a.hashfile(path)
		if e != nil {
			return e
		}
	} else {
		return errors.New("Only works on regular files: " + path)
	}
	return nil
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