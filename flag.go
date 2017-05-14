// Copyright 2015 Jeremy Wall (jeremy@marzhillstudios.com)
// Use of this source code is governed by the Artistic License 2.0.
// That License is included in the LICENSE file.
package main

import (
	"flag"
	"fmt"
	"path/filepath"
	"strings"

	homedir "github.com/mitchellh/go-homedir"
)

type StringMapFlag map[string]string

func (v StringMapFlag) String() string {
	return fmt.Sprint(map[string]string(v))
}

func (v StringMapFlag) Set(s string) error {
	parts := strings.SplitN(s, "=", 2)
	if len(parts) < 2 {
		return fmt.Errorf("Invalid mimetype mapping")
	}
	v[parts[0]] = parts[1]
	return nil
}

func mimeFlag(name, usage string) StringMapFlag {
	mimeTypeMappings := StringMapFlag{}
	flag.Var(mimeTypeMappings, name, usage)
	return mimeTypeMappings
}

var homeDir, _ = homedir.Dir()

var tessData = flag.String("tess_data_prefix", defaultTessData(), "Location of the tesseract data.")
var help = flag.Bool("help", false, "Show this help.")
var pdfDensity = flag.Int("pdfdensity", 300, "density to use when converting pdf's to tiffs.")
var indexLocation = flag.String("index_location", filepath.Join(homeDir, ".index.bleve"), "Location for the bleve index.")
var hashLocation = flag.String("hash_location", filepath.Join(homeDir, ".indexed_files"), "Location where the indexed file hashes are stored.")
var isQuery = flag.Bool("query", false, "Run a query instead of indexing")
var limit = flag.Int("limit", 10, "Limit query result to this number of item.")
var from = flag.Int("from", 0, "Start returning at this item.")
var isIndex = flag.Bool("index", false, "Run an indexing operation instead of querying")
var mimeTypeMappings = mimeFlag("mime", "Add a custom mime type mapping.")
var maxFileSize = flag.Int64("max_file_size", -1, "Maximum size of file to index. A size of -1 means no limit.")
var force = flag.Bool("force", false, "Force an index even if the file hasn't changed")
var useHighlight = flag.Bool("highlight", true, "Whether to highlight results in the output")
