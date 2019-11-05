// Copyright 2015 Jeremy Wall (jeremy@marzhillstudios.com)
// Use of this source code is governed by the Artistic License 2.0.
// That License is included in the LICENSE file.
package main

import (
	"flag"
	"fmt"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// IndexFile indexes a single file using the provided FileProcessor
func IndexFile(file string, p FileProcessor) {
	Debugf("Processing file: %q", file)
	if !*isDebug {
		fmt.Printf(".")
	}
	if ok, err := p.ShouldProcess(file); !ok {
		if err != nil {
			log.Print(err)
		}
		return
	}
	err := p.Process(file)
	if err != nil {
		log.Printf("Error Processing file %q, %v\n", file, err)
		return
	}
	return
}

// IndexFile indexes all the files in a directory recursively using
// the provided FileProcessor. It skips the directories it uses for storage.
func IndexDirectory(dir string, p FileProcessor) {
	log.Printf("Processing directory: %q", dir)
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") ||
				path == *indexLocation || path == *hashLocation {
				return filepath.SkipDir
			}
			return nil
		}
		IndexFile(path, p)
		return nil
	})
}

func formatFragment(frag string) string {
	content := fmt.Sprintf("%s", frag)
	lines := strings.Split(content, "\n")
	rv := ""
	for _, l := range lines {
		rv += fmt.Sprintf(" %s", l)
	}
	return rv
}

func usage() string {
	return fmt.Sprintln("") +
		fmt.Sprintln("Indexing: \n\tgoindexer [options] --index <locations to index>") +
		fmt.Sprintln("Querying: \n\tgoindexer [options] --query <search query>") +
		fmt.Sprintln("") +
		fmt.Sprintln("The locations to index can be a list of directories or files.") +
		fmt.Sprintln("") +
		fmt.Sprintln("The search query is in the syntax documented at: https://github.com/blevesearch/bleve/wiki/Query%20String%20Query.") +
		fmt.Sprintln("")
}

func main() {
	flag.Parse()

	// Write annoying tesseract stderr to a file
	// This means everything logged to stderr goes to that file unfortunately.
	// Setting debug_file for tesseract doesn't seem to work for everything.
	if !*isDebug {
		os.Stderr.Close()
		stderr, err := os.Create("/tmp/goin.err")
		if err != nil {
			panic(err)
		}
		os.Stderr = stderr
		defer stderr.Close()
	}

	if *help {
		fmt.Println(usage())
		flag.PrintDefaults()
		os.Exit(0)
	}

	if *serveHTTP {
		startServer()
		log.Fatal(http.ListenAndServe(":8080", nil))
	}

	if !(*isQuery) && !(*isIndex) {
		fmt.Println("One of --query or --index must be passed")
		flag.PrintDefaults()
		os.Exit(1)
	}

	for k, v := range mimeTypeMappings {
		log.Printf("Adding mime-type mapping for extension %q=%q", k, v)
		mime.AddExtensionType(k, v)
	}

	index, err := NewIndex(*indexLocation)
	if err != nil {
		log.Fatalln(err)
	}
	defer index.Close()

	if *isQuery {
		result, err := index.Query(flag.Args())
		if err != nil {
			log.Printf("Error: %q", err)
			os.Exit(1)
		}
		for i, match := range result.Hits {
			fmt.Println("")
			fmt.Printf("%d. %q (%f)\n", i+1, match.ID, match.Score)
			for field, fragments := range match.Fragments {
				fmt.Printf("%s: ", field)
				for _, frag := range fragments {
					fmt.Println(formatFragment(frag))
				}
				for fieldName, fieldValue := range match.Fields {
					if _, ok := match.Fragments[fieldName]; !ok {
						fmt.Printf("%s:\n", fieldName)
						fmt.Println(formatFragment(fmt.Sprint(fieldValue)))
					}
				}
			}
		}
		// TODO(jwall): handle facet outputs?
		fmt.Printf("\nTotal results: %d Retrieved %d to %d in %s.", result.Total, result.Request.From+1, result.Request.From+len(result.Hits), result.Took)
		return
	} else if *isIndex {
		p := NewProcessor(*hashLocation, index, *force)
		for _, file := range flag.Args() {
			fi, err := os.Stat(file)
			if os.IsNotExist(err) {
				continue
			}
			if err != nil {
				log.Printf("Error Stat(ing) file %q", err)
			}
			if fi.IsDir() {
				IndexDirectory(file, p)
			} else {
				IndexFile(file, p)
			}
		}
	}
}
