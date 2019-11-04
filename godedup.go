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

// top level object so nothing is global
type analyzer struct {
	requestedpaths []string
	filemap map[string]string
	filesbylen []string
	dirmap map[string]string
	dirsbylen []string
}

// creates new analyzer object and starts populating it based on requested paths
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
	for k, _ := range a.filemap {
        a.filesbylen = append(a.filesbylen, k)
    }
    for k, _ := range a.dirmap {
        a.dirsbylen = append(a.dirsbylen, k)
    }
    // sort the dir and file names by length, longest first:
    sort.Slice(a.filesbylen, func(i, j int) bool { return len(a.filesbylen[i]) > len(a.filesbylen[j]) })
    sort.Slice(a.dirsbylen, func(i, j int) bool { return len(a.dirsbylen[i]) > len(a.dirsbylen[j]) })
	err = a.hashdirs()
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return &a, nil
}

// hashes a specific directory based on its contents
func (a analyzer) hashdir (dirname string) (error) {
	// find all the files and dirs directly contained herein

	dirnamelen := len(dirname)
	for filename, _ := range a.filemap {
		filenamelen := len(filename)
		if dirnamelen >= filenamelen {
			break
		}
		fmt.Println(filename + " may be a member of " + dirname)
	}
	// close the hash here
	fmt.Println("DIR:  " + dirname)
	return nil
}

// hashes all directories, starting with the longest first
func (a analyzer) hashdirs () (error) {

	for _, k := range a.dirsbylen {
		a.hashdir(k)
	}
	return nil
}

// hashes a specific file based on its contents
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

// the callback from filepath.walk to process both dirs and files
func (a analyzer) process(path string, entry os.FileInfo, err error) error {
	if err != nil {
	    return err
	}
	if entry.IsDir() {
		// ensure there is one and only one trailng slash on dirs
		sep := string(os.PathSeparator)
		path = strings.TrimRight(path, sep) + sep
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