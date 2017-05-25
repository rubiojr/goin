package main

import (
	"os"

	"github.com/blevesearch/bleve"
	"github.com/blevesearch/bleve/analysis/lang/en"
	"github.com/blevesearch/bleve/mapping"
	// id3 "github.com/mikkyang/id3-go"
	"github.com/dhowden/tag"
)

type MP3 struct {
	*FileData
	Artist string `json:"artist"`
	Album  string `json:"album"`
	Genre  string `json:"genre"`
	Title  string `json:"title"`
	Track  int    `json:"track"`
	Year   int    `json:"year"`
}

func (mp3 *MP3) Type() string {
	return "mp3"
}

func (fd *MP3) Path() string {
	return fd.FullPath
}

func (mp3 *MP3) Analyse() {
	//mp3File, err := id3.Open(mp3.FullPath)
	f, err := os.Open(mp3.FullPath)
	mp3File, err := tag.ReadFrom(f)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	mp3.Artist = mp3File.Artist()
	mp3.Title = mp3File.Title()
	mp3.Album = mp3File.Album()
	mp3.Genre = mp3File.Genre()
	mp3.Year = mp3File.Year()
	n, _ := mp3File.Track()
	mp3.Track = n
}

func buildMapping() mapping.IndexMapping {
	enFieldMapping := bleve.NewTextFieldMapping()
	enFieldMapping.Analyzer = en.AnalyzerName

	//mp3Mapping := bleve.NewDocumentMapping()
	//mp3Mapping.AddFieldMappingsAt("artist", enFieldMapping)
	//mp3Mapping.AddFieldMappingsAt("title", enFieldMapping)

	mapping := bleve.NewIndexMapping()
	//mapping.DefaultMapping = mp3Mapping
	mapping.DefaultAnalyzer = en.AnalyzerName

	return mapping
}
