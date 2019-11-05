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

type entry interface {
    pathname() string
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

func newDir(name string) *Dir {
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
	firstpart := path[0]
	remainder := path[1:]
	if len(remainder) == 0 {
		// firstpart is the file
		f := File{Name:firstpart}
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
// notifies the requester when it is done
func (currfile *File) calcdigest(wg *sync.WaitGroup) {
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
}

// walks the dirtree, populating Pathname values and calculating hashes
func (currdir *Dir) InformLineage(level int, preface string) {
    for _, file := range currdir.Files {
		// print file info
        file.Pathname = fmt.Sprintf("%s%s%c%s", preface, currdir.Name,
			os.PathSeparator, file.Name)
		file.Level = level + 1
    }
    for _, subdir := range currdir.Dirs {
        subdir.InformLineage(level + 1, preface + currdir.Name + string(os.PathSeparator))
    }
    // print dir info
    currdir.Pathname = fmt.Sprintf("%s%s", preface, currdir.Name)
	currdir.Level = level
}

// recursively hashes file and dir contents
func (currdir *Dir) calcdigests() error {
	// we need a canonical list of the hashes of the files and subdirs
	hashlist := make([]string,0)

	// launch file hashing goroutines
    var wg sync.WaitGroup
    for _, file := range currdir.Files {
        wg.Add(1)
		go file.calcdigest(&wg)
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
        e := subdir.calcdigests()
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

type EntryList *[]entry

type LevelMap struct {
	entrymap map[int]EntryList
}

func newLevelMap() *LevelMap {
	lm := LevelMap{ entrymap:make(map[int]EntryList) }
    return &lm
}

type Analyzer struct {
	topdirs []*Dir
	digestmap map[string]*LevelMap
}

// creates new Analyzer object and starts populating it based on requested paths
func NewAnalyzer(toppaths []string) (*Analyzer, error) {
	a := Analyzer {
		topdirs:make([]*Dir,0),
		digestmap:make(map[string]*LevelMap),
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
	}
	return &a, nil
}

func (a Analyzer) analyze() {
	for _, topdir := range a.topdirs {
		// set pathnames (Go has no optional parameters)
		topdir.InformLineage(1, "")
		// calculate all the file hashes and dir metahashes
		topdir.calcdigests()
	}
}

func (a Analyzer) dump()  {
	for _, v := range a.topdirs {
		v.dump()
	}
}

// the callback from filepath.walk to process both dirs and files
func (a Analyzer) process(path string, entry os.FileInfo, err error) error {
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
	var a *Analyzer
	var err error
	a, err = NewAnalyzer(os.Args[1:])
	if err != nil {
	    log.Println(err)
	}
	a.analyze()
	a.dump()
}