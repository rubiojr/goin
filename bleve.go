// Copyright 2015 Jeremy Wall (jeremy@marzhillstudios.com)
// Use of this source code is governed by the Artistic License 2.0.
// That License is included in the LICENSE file.
package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/analysis"
	"github.com/blevesearch/bleve/analysis/char/html"
	"github.com/blevesearch/bleve/analysis/lang/en"
	"github.com/blevesearch/bleve/mapping"
	"github.com/blevesearch/bleve/registry"
	"github.com/blevesearch/bleve/search/highlight/highlighter/ansi"
)

const htmlMimeType = "text/html"

// handle text/html types
func init() {
	registry.RegisterAnalyzer(htmlMimeType, func(config map[string]interface{}, cache *registry.Cache) (*analysis.Analyzer, error) {
		a, err := en.AnalyzerConstructor(config, cache)
		if err != nil {
			if cf, err := cache.CharFilterNamed(html.Name); err == nil {
				a.CharFilters = []analysis.CharFilter{cf}
			} else {
				return nil, err
			}
		}

		return a, err
	})
}

func buildHtmlDocumentMapping() *mapping.DocumentMapping {
	dm := bleve.NewDocumentMapping()
	dm.DefaultAnalyzer = htmlMimeType
	return dm
}

type Index interface {
	Put(data *IFile) error
	Query(terms []string) (*bleve.SearchResult, error)
	Close() error
}

type bleveIndex struct {
	index bleve.Index
}

func (i *bleveIndex) Put(data *IFile) error {
	if err := i.index.Index((*data).Path(), data); err != nil {
		return fmt.Errorf("Error writing to index: %q", err)
	}
	return nil
}

func (i *bleveIndex) Query(terms []string) (*bleve.SearchResult, error) {
	searchQuery := strings.Join(terms, " ")
	query := bleve.NewQueryStringQuery(searchQuery)
	// TODO(jwall): limit, skip, and explain should be configurable.
	request := bleve.NewSearchRequestOptions(query, *limit, *from, false)
	if *useHighlight {
		request.Highlight = bleve.NewHighlightWithStyle(ansi.Name)
	} else {
		request.Highlight = bleve.NewHighlight()
	}

	result, err := i.index.Search(request)
	if err != nil {
		log.Printf("Search Error: %q", err)
		return nil, err
	}
	return result, nil
}

func (i *bleveIndex) Close() error {
	return i.index.Close()
}
func NewIndex(indexLocation string) (Index, error) {
	// TODO(jwall): An abstract indexing interface?
	var index bleve.Index
	if _, err := os.Stat(indexLocation); os.IsNotExist(err) {
		mapping := bleve.NewIndexMapping()
		mapping.DefaultAnalyzer = "en"
		mapping.AddDocumentMapping(htmlMimeType, buildHtmlDocumentMapping())
		// TODO(jwall): Create document mappings for our custom types.
		log.Printf("Creating new index %q", indexLocation)
		if index, err = bleve.NewUsing(indexLocation, mapping, "scorch", "scorch", nil); err != nil {
			return nil, fmt.Errorf("Error creating index %q\n", err)
		}
	} else {
		readOnly := false
		if *isQuery {
			readOnly = true
		}
		opts := map[string]interface{}{
			"read_only": readOnly,
		}
		Debugf("Opening index %q (readonly: %t)\n", indexLocation, readOnly)
		if index, err = bleve.OpenUsing(indexLocation, opts); err != nil {
			return nil, fmt.Errorf("Error opening index %q\n", err)
		}
	}
	return &bleveIndex{index}, nil
}
