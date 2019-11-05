package main

import (  
    "fmt"
    "io"
    "os"
    "log"
    //"sort"
    "errors"
    "bytes"
    "strings"
    "path/filepath"
    "crypto/sha1"
    //"encoding/hex"
)

type File struct {
    Name string
}

type Dir struct {
    Name  string
    Files []File
    Dirs  map[string]*Dir
}

func canonicalpath(pathname string) string {
	var buf bytes.Buffer
	var last rune
	for i, r := range pathname {
		if r != last || i == 0 || r != os.PathSeparator {
			buf.WriteRune(r)
			last = r
		}
	}
	pathname = buf.String()
	if (pathname[:2] == "./"){
		pathname=pathname[2:]
	}
	return pathname
}

func newDir(name string) *Dir {
	fmt.Println("creating new dir: " + name)
	d := Dir{name, []File{}, make(map[string]*Dir)}
    return &d
}

func (f *Dir) addDir(path []string) (error) {
	fmt.Printf("addDir on \"%s\"",strings.Join(path, "/"))
	fmt.Printf(" with len %d\n", len(path))

	next := path[0]
	rem := path[1:]

	fmt.Println("placing " + next + " in " + f.Name)
	if val, ok := f.Dirs[next]; ok {
		fmt.Println("found " + next + " in " + f.Name)
		return (val.addDir(rem))
	}
	fmt.Println("couldnt find " + next + " in " + f.Name)
	f.Dirs[next] = newDir(next)
	if len(rem) > 1 {
		fmt.Println("moving on to " + strings.Join(rem, "/") + " in " + next)
		return (f.Dirs[next].addDir(rem))
	}
	// directory already exists, do nothing
	return nil
}

func (d *Dir) addFile(path []string) error {
	//fmt.Printf("adding file %s\n", strings.Join(path, "/") )
	next := path[0]
	rem := path[1:]
	if len(rem) == 0 {
		// this is the file
		d.Files = append(d.Files, File{next})
		return nil
	}	
	// still finding the containing dir
	if val, ok := d.Dirs[next]; ok {
		return (val.addFile(rem))
	}
	return errors.New("somehow can't find a subdir for file placement")
}

func (currdir *Dir) Dump(preface string) {
    for _, file := range currdir.Files {
        fmt.Printf("%s%s%c%s\n", preface, currdir.Name, os.PathSeparator, file.Name)
    }
    for _, subdir := range currdir.Dirs {
        subdir.Dump(preface + currdir.Name + string(os.PathSeparator))
    }
    fmt.Printf("%s%s\n", preface, currdir.Name)
}

// top level object so nothing is global
type analyzer struct {
	rootdirs       []*Dir
}

// creates new analyzer object and starts populating it based on requested paths
func NewAnalyzer(rootpaths []string) (*analyzer, error) {
	a := analyzer {
		rootdirs:make([]*Dir,0),
	}
	var err error
	for _, rootpathname := range rootpaths {
		// fold any duplicate slashes down to just one
		rootpathname = canonicalpath(rootpathname)
		// add this dir to the requested top level dirs
		a.rootdirs = append(a.rootdirs, newDir(rootpathname))

		processclosure := func(path string, fileInfo os.FileInfo, e error) (err error) {
			e = a.process(path, fileInfo, e)
			if e != nil {
			    log.Println("error in process closure:")
			    log.Println(e)
			    os.Exit(-1)
			}
			return nil
		}
		err = filepath.Walk(rootpathname, processclosure)
		if err != nil {
		    log.Println(err)
		    return nil, err
		}
	}
	return &a, nil
}

// hashes a specific directory based on its contents
func (a analyzer) hashdir (dirname string) (error) {
	// find all the files and dirs directly contained herein
	return nil
}

// hashes all directories, starting with the longest first
func (a analyzer) hashdirs () (error) {
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
	//a.filemap[path] = hex.EncodeToString(h.Sum(nil))
	return nil
}

// the callback from filepath.walk to process both dirs and files
func (a analyzer) process(path string, entry os.FileInfo, err error) error {
	if err != nil {
		fmt.Println("process being called with an error")
		fmt.Println(err)
	    return err
	}
	for _, rootdir := range a.rootdirs {
		rootpath := rootdir.Name
		if rootpath == path {
			// already added the root level dirs
			return nil
		}
		// which rootpath are we in?
		if rootpath == path[:len(rootpath)] {
			trimpath := path[len(rootpath)+1:]
			segments := strings.Split(trimpath, string(os.PathSeparator))
			if entry.IsDir() {
				return rootdir.addDir(segments)
			}
			if !entry.Mode().IsRegular() {
				return errors.New("irregular files not handled: " + path)
			}
			// must be a file:
			//fmt.Println("adding a file " + path)
			return rootdir.addFile(segments)
		}
	}
	return errors.New("path outside requested roots: " + path)
}

func main() {
	var a *analyzer
	var err error
	a, err = NewAnalyzer(os.Args[1:])
	if err != nil {
	    log.Println(err)
	}
	for _, v := range a.rootdirs {
		fmt.Println()
		v.Dump("")
	}
}