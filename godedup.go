package main

import (  
    "fmt"
    "io"
    "os"
    "log"
    "sort"
    "errors"
    "strings"
    "path/filepath"
    "crypto/sha1"
    "encoding/hex"
)

type analyzer struct {
	requestedpaths []string
	filemap map[string]string
	filesbylen []string
	dirmap map[string]string
	dirsbylen []string
}

func NewAnalyzer(paths []string) (*analyzer, error) {
	a := analyzer {
		requestedpaths:paths,
		filemap:make(map[string]string),
		filesbylen:make([]string,0),
		dirmap:make(map[string]string),
		dirsbylen:make([]string,0),
	}
	var err error
	for _, element := range paths {
		fileclosure := func(path string, fileInfo os.FileInfo, e error) (err error) {
			e = a.process(path, fileInfo, e)
			if e != nil {
			    log.Println(err)
			    return e
			}
			return nil
		}
		err = filepath.Walk(element, fileclosure)
		if err != nil {
		    log.Println(err)
		    return nil, err
		}
	}
	err = a.hashdirs()
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return &a, nil
}

func (a analyzer) hashdir (path string) (error) {
	// find all the files and dirs directly contained herein
	/*var files []string
	var dirs []string
	var rollingHash string

	dirlen := len(path)
	for k, v := range a.filemap {
		//fmt.Println("FILE: " + v + " " + k)
		if substr(k,path) {
			//
		}
	}
	*/
	fmt.Println("DIR:  " + path)
	return nil
}

func (a analyzer) hashdirs () (error) {
	//for k, v := range a.filemap {
	//	fmt.Println("FILE: " + v + " " + k)
	//}
    for k, _ := range a.filemap {
        a.filesbylen = append(a.filesbylen, k)
    }
    for k, _ := range a.dirmap {
        a.dirsbylen = append(a.dirsbylen, k)
    }
    // sort the dir and file names by length, longest first:
    sort.Slice(a.filesbylen, func(i, j int) bool { return len(a.filesbylen[i]) > len(a.filesbylen[j]) })
    sort.Slice(a.dirsbylen, func(i, j int) bool { return len(a.dirsbylen[i]) > len(a.dirsbylen[j]) })
	for _, k := range a.dirsbylen {
		a.hashdir(k)
	}
	return nil
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
		// ensure there is one and only one trailng slash on dirs
		path = strings.TrimRight(path, "/") + "/"
		// just make a note of this dir for now
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