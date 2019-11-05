package main

import (  
    "fmt"
    "io"
    "os"
    "log"
    "errors"
    "bytes"
    "strings"
    "path/filepath"
    "crypto/sha1"
    "encoding/hex"
)

type File struct {
    Name     string
    Pathname string
    Digest	 string
}

type Dir struct {
    Name     string
    Pathname string
    Files    []*File
    Dirs     map[string]*Dir
    Digest   string
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
	//fmt.Println("creating new dir: " + name)
	d := Dir{
		Name:name,Pathname:"",
		Files:[]*File{},
		Dirs:make(map[string]*Dir),
		Digest:"",
	}
    return &d
}

func (currdir *Dir) addDir(path []string) (error) {
	firstpart := path[0]
	remainder := path[1:]
	if val, ok := currdir.Dirs[firstpart]; ok {
		return (val.addDir(remainder))
	}
	currdir.Dirs[firstpart] = newDir(firstpart)
	if len(remainder) > 1 {
		return (currdir.Dirs[firstpart].addDir(remainder))
	}
	// directory already exists, do nothing
	return nil
}

func (d *Dir) addFile(path []string) error {
	//fmt.Printf("adding file %s\n", strings.Join(path, "/") )
	firstpart := path[0]
	remainder := path[1:]
	if len(remainder) == 0 {
		// firstpart is the file
		f := File{Name:firstpart,Pathname:"",Digest:""}
		d.Files = append(d.Files, &f)
		return nil
	}	
	// still finding the containing dir
	if val, ok := d.Dirs[firstpart]; ok {
		return (val.addFile(remainder))
	}
	return errors.New("somehow can't find a subdir for file placement")
}

// hashes a specific file based on its contents
func (currfile *File) calcDigest() (error) {
	f, err := os.Open(currfile.Pathname)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	currfile.Digest = hex.EncodeToString(h.Sum(nil))
	return nil
}

func (topdir *Dir) Analyze() {
	// tell all the files and dirs what their pathnames are
	topdir.InformLineage("")
	// use those pathnames to calculate digests
	topdir.CalcDigest()
}

// walks the dirtree, populating Pathname values and calculating hashes
func (currdir *Dir) InformLineage(preface string) {
    for _, file := range currdir.Files {
		// print file info
        file.Pathname = fmt.Sprintf("%s%s%c%s", preface, currdir.Name,
			os.PathSeparator, file.Name)
    }
    for _, subdir := range currdir.Dirs {
        subdir.InformLineage(preface + currdir.Name + string(os.PathSeparator))
    }
    // print dir info
    currdir.Pathname = fmt.Sprintf("%s%s", preface, currdir.Name)
}

// recursively hashes file and dir contents
func (currdir *Dir) CalcDigest() error {
    for _, file := range currdir.Files {
		// print file info
		e := file.calcDigest()
		if e != nil {
			return e
		}
    }
    for _, subdir := range currdir.Dirs {
        e := subdir.CalcDigest()
        if e != nil {
			return e
        }
    }
    // print dir info
    //fmt.Println(currdir.Pathname)
    return nil
}

// recursively hashes file and dir contents
func (currdir *Dir) Dump() {
    for _, file := range currdir.Files {
		// print file info
        fmt.Println(file.Digest + " " + file.Pathname)
    }
    for _, subdir := range currdir.Dirs {
        subdir.Dump()
    }
    // print dir info
    fmt.Println(currdir.Digest + " " + currdir.Pathname)
}

// top level object so nothing is global
type analyzer struct {
	topdirs       []*Dir
}

// creates new analyzer object and starts populating it based on requested paths
func NewAnalyzer(toppaths []string) (*analyzer, error) {
	a := analyzer {
		topdirs:make([]*Dir,0),
	}
	var err error
	for _, toppathname := range toppaths {
		// fold any duplicate slashes down to just one
		toppathname = canonicalpath(toppathname)
		// add this dir to the requested top level dirs
		currdir := newDir(toppathname)
		a.topdirs = append(a.topdirs, currdir)

		processclosure := func(path string, fileInfo os.FileInfo, e error) (err error) {
			e = a.process(path, fileInfo, e)
			if e != nil {
			    log.Println(e)
			    os.Exit(-1)
			}
			return nil
		}
		// walk each requested top level dir for subdirs and files
		err = filepath.Walk(toppathname, processclosure)
		if err != nil {
		    log.Println(err)
		    return nil, err
		}
		currdir.Analyze()
	}
	return &a, nil
}

func (a analyzer) Dump()  {
	for _, v := range a.topdirs {
		v.Dump()
	}
}

// the callback from filepath.walk to process both dirs and files
func (a analyzer) process(path string, entry os.FileInfo, err error) error {
	if err != nil {
		fmt.Println(err)
	    return err
	}
	for _, topdir := range a.topdirs {
		toppath := topdir.Name
		if toppath == path {
			// already added the top level dirs
			return nil
		}
		// which toppath are we in?
		if toppath == path[:len(toppath)] {
			trimpath := path[len(toppath)+1:]
			segments := strings.Split(trimpath, string(os.PathSeparator))
			if entry.IsDir() {
				return topdir.addDir(segments)
			}
			if !entry.Mode().IsRegular() {
				return errors.New("irregular files not handled: " + path)
			}
			// must be a file:
			return topdir.addFile(segments)
		}
	}
	return errors.New("path outside requested tops: " + path)
}

func main() {
	var a *analyzer
	var err error
	a, err = NewAnalyzer(os.Args[1:])
	if err != nil {
	    log.Println(err)
	}
	a.Dump()
}