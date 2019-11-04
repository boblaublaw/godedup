package main

import (  
    "fmt";
    "os";
    "log";
    "path/filepath"
)

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

func addPaths(paths []string) error {
	for _, element := range paths {
		err := filepath.Walk(element, processentry)
		if err != nil {
		    log.Println(err)
		    return err
		}
	}
	return nil
}

func main() {
	argsWithoutProg := os.Args[1:]
	fmt.Println(argsWithoutProg)

	err := addPaths(argsWithoutProg)
	if err != nil {
	    log.Println(err)
	}
}