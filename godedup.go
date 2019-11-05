package main

import (  
    "fmt"
    "io"
    "os"
    "log"
    "sort"
    "sync"
    "errors"
    "bytes"
    "strings"
    "strconv"
    "path/filepath"
    "crypto/sha1"
    "encoding/hex"
)

// helper function
func canonicalpath(inpath string) string {
	var buf bytes.Buffer
	var last rune
	for i, r := range inpath {
		if r != last || i == 0 || r != os.PathSeparator {
			buf.WriteRune(r)
			last = r
		}
	}
	outpath := buf.String()
	if (outpath[:2] == "./"){
		outpath=outpath[2:]
	}
	if (outpath[len(outpath)-1:] == "/") {
		outpath=outpath[:len(outpath)-1]
	}
	return outpath
}

type File struct {
    Name      string
    Pathname  string
    Digest	  string
    Level     int
    DigestErr error
}

type Dir struct {
    Name     string
    Pathname string
    Digest   string
    Level    int
    Files    []*File
    Dirs     map[string]*Dir
}

type entry interface {
    pathname() string
    isdir()    bool
    level()    int
    digest()   string
}

func (currfile *File) pathname () string {
	return currfile.Pathname
}

func (currdir *Dir) pathname () string {
	return currdir.Pathname
}

func (currfile *File) digest () string {
	return currfile.Digest
}

func (currdir *Dir) digest () string {
	return currdir.Digest
}

func (currfile *File) level () int {
	return currfile.Level
}

func (currdir *Dir) level () int {
	return currdir.Level
}

func (currfile *File) isdir () bool {
	return false
}

func (currdir *Dir) isdir () bool {
	return true
}

func newDir(name string, pathname string, level int) *Dir {
	d := Dir{
		Name:name,
		Pathname:pathname,
		Level:level,
		Files:[]*File{},
		Dirs:make(map[string]*Dir),
		Digest:"",
	}
    return &d
}

func (currdir *Dir) addDir(segments []string, pathname string, level int, info os.FileInfo) (error) {
	firstpart := segments[0]
	remainder := segments[1:]
	// already exists, recurse into it:
	if val, ok := currdir.Dirs[firstpart]; ok {
		return (val.addDir(remainder, pathname, level, info))
	}
	// create a new leaf and recurse into it
	if len(remainder) == 0 {
		currdir.Dirs[firstpart] = newDir(firstpart, pathname, level)
		return nil
	}
	return (currdir.Dirs[firstpart].addDir(remainder, pathname, level, info))
}

func (d *Dir) addFile(segments []string, pathname string, level int, info os.FileInfo) error {
	firstpart := segments[0]
	remainder := segments[1:]
	if len(remainder) == 0 {
		// firstpart is the file
		f := File{Name:firstpart,Pathname:pathname,Level:level}
		d.Files = append(d.Files, &f)
		return nil
	}	
	// still finding the containing dir
	if val, ok := d.Dirs[firstpart]; ok {
		return (val.addFile(remainder, pathname, level, info))
	}
	return errors.New("somehow can't find a subdir for file placement")
}

// hashes a specific file based on its contents
// notifies the requester when it is done
func (currfile *File) calcdigest(a *Analyzer, wg *sync.WaitGroup) {
	defer wg.Done()
	f, err := os.Open(currfile.Pathname)
	if err != nil {
		currfile.DigestErr = err
		return
	}
	defer f.Close()
	h := sha1.New()
	if _, err := io.Copy(h, f); err != nil {
		currfile.DigestErr = err
		return
	}
	currfile.Digest = hex.EncodeToString(h.Sum(nil))
	a.storedirentry(currfile)
}

// recursively hashes file and dir contents
func (currdir *Dir) calcdigests(a *Analyzer) error {
	// we need a canonical list of the hashes of the files and subdirs
	hashlist := make([]string,0)

	// launch file hashing goroutines
    var wg sync.WaitGroup
    for _, file := range currdir.Files {
        wg.Add(1)
		go file.calcdigest(a, &wg)
	}
    wg.Wait()
    // tally up the hashes from completed goroutines
	for _, file := range currdir.Files {
		if file.DigestErr != nil {
			return file.DigestErr
		}
		hashlist = append(hashlist, file.Digest)
    }
    for _, subdir := range currdir.Dirs {
        e := subdir.calcdigests(a)
        if e != nil {
			return e
        }
		hashlist = append(hashlist, subdir.Digest)
    }

    // sort digests of subdirs and files so we're not
    // sensitive to the order of traversal.
    sort.Strings(hashlist)

    h := sha1.New()
	// seed the hash with something to distinguish
	// an empty dir from an empty file:
	io.WriteString(h, "DIRECTORYSEED")
	for _, hash := range hashlist {
		io.WriteString(h, hash)
	}
    currdir.Digest = hex.EncodeToString(h.Sum(nil))
	a.storedirentry(currdir)
    return nil
}

// recursively hashes file and dir contents
func (currdir *Dir) dump() {
    for _, file := range currdir.Files {
        fmt.Printf("%s %03d %s\n", file.Digest, file.Level, file.Pathname)
    }
    for _, subdir := range currdir.Dirs {
        subdir.dump()
    }
    fmt.Printf("%s %03d %s\n", currdir.Digest, currdir.Level, currdir.Pathname)
}

type EntryList = []entry

func newEntryList() EntryList {
	el := make([]entry,0)
    return el
}

type LevelMap = map[int]EntryList

func newLevelMap() LevelMap {
	lm := make(map[int]EntryList)
    return lm
}

type Analyzer struct {
	topdirs []*Dir
	digestmap map[string]LevelMap
}

// creates new Analyzer object and starts populating it based on requested paths
func NewAnalyzer() (*Analyzer) {
	a := Analyzer {
		topdirs:make([]*Dir,0),
		digestmap:make(map[string]LevelMap),
	}
	return &a
}

func (a *Analyzer) analyze(toppaths []string) (error) {
	for _, toppathname := range toppaths {
		// fold any duplicate slashes down to just one
		toppathname = canonicalpath(toppathname)
		// add this dir to the requested top level dirs
		currdir := newDir(toppathname, toppathname, 1)
		a.topdirs = append(a.topdirs, currdir)

		processclosure := func(path string, fileInfo os.FileInfo, e error) (error) {
			e = a.process(path, fileInfo, e)
			if e != nil {
			    return e
			}
			return nil
		}
		// walk each requested top level dir for subdirs and files
		err := filepath.Walk(toppathname, processclosure)
		if err != nil {
		    return err
		}
		currdir.calcdigests(a)
	}
	return nil
}

// the callback from filepath.walk to process both dirs and files
func (a *Analyzer) process(path string, info os.FileInfo, err error) error {
	if err != nil {
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
			if info.IsDir() {
				return topdir.addDir(segments, path, len(segments) + 1, info)
			}
			if !info.Mode().IsRegular() {
				return errors.New("irregular files not handled: " + path)
			}
			// must be a file:
			return topdir.addFile(segments, path, len(segments) + 1, info)
		}
	}
	return errors.New("path outside requested tops: " + path)
}

func (a *Analyzer) storedirentry(e entry) {
	dig := e.digest()
	lev := e.level()

	if _, ok := a.digestmap[dig]; ok {
		if _, ok := a.digestmap[dig][lev]; ok {
			a.digestmap[dig][lev] = append(a.digestmap[dig][lev], e)
		} else {
			a.digestmap[dig][lev] = newEntryList()
			a.digestmap[dig][lev] = append(a.digestmap[dig][lev], e)
		}
	} else {
		a.digestmap[dig] = newLevelMap()
		a.digestmap[dig][lev] = newEntryList()
		a.digestmap[dig][lev] = append(a.digestmap[dig][lev], e)
	}
}

func (a *Analyzer) showduplicates()  {
	fmt.Println("#!/bin/sh\n# REVIEW ALL THESE COMMANDS BEFORE EXECUTION\n")
	for _, lm := range a.digestmap {
		count := 0
		for _, el := range lm {
			count = count + len(el)
		}
		// skip all unique files
		if count > 1 {
			index := 0
			levels := make([]int,0)
			for lev, _ := range lm {
				levels = append(levels, lev)
			}
			// prefer the files and dirs closest to the topdirs
			sort.Ints(levels)
			for _, lev := range levels {
				el := lm[lev]
				// at same level, prefer shorter pathnames
				// at same level and pathnamelen, prefer alphabetical
				sort.Slice(el, func(i, j int) bool {
					pa := el[i].pathname()
					pb := el[j].pathname()
					scorea := len(pa)
					scoreb := len(pb)
					if scorea != scoreb {
						return scorea < scoreb
					}
					return pa < pb
				})
				for _, e := range el {
					index = index + 1
					// the first instance is the keeper
					if index == 1 {
						fmt.Printf("# keep %s\n",strconv.Quote(e.pathname()))
					} else {
						if e.isdir() {
							fmt.Printf("rm -rf %s\n",strconv.Quote(e.pathname()))
						} else {
							fmt.Printf("rm     %s\n",strconv.Quote(e.pathname()))
						}
					}
				}
			}
			fmt.Println()
		}
	}
}

func main() {
	a := NewAnalyzer()

	err := a.analyze(os.Args[1:])
	if err != nil {
	    log.Println(err)
	}
	a.showduplicates()
}