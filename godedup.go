package main

import (  
    "fmt"
    "io"
    "os"
    "log"
    "sort"
    "errors"
    "path/filepath"
    "crypto/sha1"
    "encoding/hex"
)

type analyzer struct {
	requestedpaths []string
	filemap map[string]string
	dirmap map[string]string
}

func NewAnalyzer(paths []string) (*analyzer, error) {
	a := analyzer {
		requestedpaths:paths,
		filemap:make(map[string]string),
		dirmap:make(map[string]string),
	}

	for _, element := range paths {
		fileclosure := func(path string, fileInfo os.FileInfo, e error) (err error) {
			e = a.process(path, fileInfo, e)
			if e != nil {
			    log.Println(err)
			    return e
			}
			return nil
		}
		err := filepath.Walk(element, fileclosure)
		if err != nil {
		    log.Println(err)
		    return nil, err
		}
	}
	for k, v := range a.filemap {
		fmt.Println("FILE: " + v + " " + k)
	}
	var keys []string
    for k, _ := range a.dirmap {
        keys = append(keys, k)
    }
    sort.Strings(keys)
	for _, k := range keys {
		fmt.Println("DIR:  " + k)
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
	a.filemap[path] = hex.EncodeToString(h.Sum(nil))
	return nil
}

func (a analyzer) process(path string, entry os.FileInfo, err error) error {
	if err != nil {
	    return err
	}
	if entry.IsDir() {
		// just make a note of the dirs for now
		// we'll generate hashes for these later
		a.dirmap[path] = ""
	} else if entry.Mode().IsRegular() {
		// hash the file contents
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
	log.Println(a.requestedpaths)
}