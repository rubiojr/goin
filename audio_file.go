package main

import (
	"os"

	// id3 "github.com/mikkyang/id3-go"
	"github.com/dhowden/tag"
)

type AudioData struct {
	*FileData
	Artist string `json:"artist"`
	Album  string `json:"album"`
	Genre  string `json:"genre"`
	Title  string `json:"title"`
	Track  int    `json:"track"`
	Year   int    `json:"year"`
}

func (data *AudioData) Type() string {
	return "audio"
}

func (data *AudioData) Path() string {
	return data.FullPath
}

func (data *AudioData) Analyse() {
	//audioFile, err := id3.Open(mp3.FullPath)
	f, err := os.Open(data.FullPath)
	audioFile, err := tag.ReadFrom(f)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	data.Artist = audioFile.Artist()
	data.Title = audioFile.Title()
	data.Album = audioFile.Album()
	data.Genre = audioFile.Genre()
	data.Year = audioFile.Year()
	n, _ := audioFile.Track()
	data.Track = n
}
