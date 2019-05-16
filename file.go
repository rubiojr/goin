// Copyright 2015 Jeremy Wall (jeremy@marzhillstudios.com)
// Use of this source code is governed by the Artistic License 2.0.
// That License is included in the LICENSE file.
package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	lpt "gopkg.in/GeertJohan/go.leptonica.v1"
	gts "gopkg.in/GeertJohan/go.tesseract.v1"
)

var (
	red = color.New(color.FgRed).SprintFunc()
)

func init() {
	// Ensure that org-mode is registered as a mime type.
	mime.AddExtensionType(".org", "text/x-org")
	mime.AddExtensionType(".org_archive", "text/x-org")
	mime.AddExtensionType(".mp3", "audio/mp3")
	mime.AddExtensionType(".m4a", "audio/mp4a-latm")
}

func defaultTessData() (possible string) {
	possible = os.Getenv("TESSDATA_PREFIX")
	if possible == "" {
		possible = "/usr/local/share"
	}
	return
}

func hashFileName(file string) string {
	dirPath := filepath.Dir(file)
	prefix := strings.Replace(dirPath, string(filepath.Separator), "_", -1)
	return prefix + filepath.Base(file)
}

// FileTranslators turn a file into text. The get registered in a FileProcessor
// using the FileProcessor.Register method call.
type FileTranslator func(string) (string, error)

// TODO(jwall): Okay large file support without having to load the entire file
// into memory would be nice.
func getPixImage(f string) (*lpt.Pix, error) {
	//log.Print("extension: ", filepath.Ext(f))
	if filepath.Ext(f) == ".pdf" {
		if cmdName, err := exec.LookPath("convert"); err == nil {
			tmpFName := filepath.Join(os.TempDir(), filepath.Base(f)+".tif")
			Debugf("converting %q to %q", f, tmpFName)
			cmd := exec.Command(cmdName, "-background", "white", "-flatten", "-alpha", "Off", "-density", fmt.Sprint(*pdfDensity), f, "-depth", "8", tmpFName)
			out, err := cmd.CombinedOutput()
			if err != nil {
				log.Printf("output: %q", out)
				return nil, printError("converting pdf with %q err: %v", cmd.Args, err)
			}
			f = tmpFName
		} else {
			return nil, printError("Unable to find convert binary %v", err)
		}
	}
	Debugf("getting pix from %q", f)
	return lpt.NewPixFromFile(f)
}

func ocrImageFile(file string) (string, error) {
	// Create new tess instance and point it to the tessdata location.
	// Set language to english.
	t, err := gts.NewTess(filepath.Join(*tessData, "tessdata"), *tesseractLang)
	if err != nil {
		log.Fatalf("Error while initializing Tess: %s\n", err)
	}
	defer t.Close()

	pix, err := getPixImage(file)
	if err != nil {
		return "", printError("while getting pix from file: %s (%s)", file, err)
	}
	defer pix.Close()

	t.SetPageSegMode(gts.PSM_AUTO_OSD)

	// TODO(jwall): What is this even?
	err = t.SetVariable("tessedit_char_whitelist", ` !"#$%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\]^_abcdefghijklmnopqrstuvwxyz{|}~`+"`")
	if err != nil {
		return "", printError("Failed to set variable: %s\n", err)
	}

	if !*isDebug {
		err = t.SetVariable("debug_file", *tessDebugFile)
		if err != nil {
			return "", printError("Failed to set variable: %s\n", err)
		}
	}
	t.SetImagePix(pix)

	return t.Text(), nil
}

func getPlainTextContent(file string) (string, error) {
	fd, err := os.Open(file)
	defer fd.Close()
	if err != nil {
		return "", err
	}
	bs, err := ioutil.ReadAll(fd)
	if err != nil {
		return "", err
	}
	return string(bs), nil
}

type IFile interface {
	Type() string
	Path() string
}

// FileData represents the data about a file to be indexed.

type FileData struct {
	// Full path to the file on disk.
	FullPath string `json:"FullPath"`
	// Basename of the file.
	FileName string `json:"FileName"`
	// MimeType of the file.
	MimeType string `json:"MimeType"`
	// Time of last index.
	IndexTime time.Time `json:"IndexTime"`
	// Text content of the file.
	Text string `json:"Text"`
	// Size of the file.
	Size int64 `json:"Size"`
}

// Type satisifies the bleve.Classifier interface for FileData.
func (fd *FileData) Type() string {
	return fd.MimeType
}

func (fd *FileData) Path() string {
	return fd.FullPath
}

// FileProcessor is the interface FileProcessors must implement to handle a file.
type FileProcessor interface {
	ShouldProcess(file string) (bool, error)
	Process(file string) error
	Register(mime string, ft FileTranslator) error
	// FileProcessors also implement the Index interface.
	Index
}

type processor struct {
	defaultMimeTypeHandlers map[string]FileTranslator
	hashDir                 string
	force                   bool
	Index
}

func getAudioText(file string) (string, error) {
	return "audio", nil
}

func getPdfText(file string) (string, error) {
	// 1. try pdftotext if it exists.
	if cmdName, err := exec.LookPath("pdftotext"); err == nil {
		tmpName := filepath.Join(os.TempDir(), filepath.Base(file)+".txt")
		cmd := exec.Command(cmdName, file, tmpName)
		out, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("output: %q", out)
			log.Printf("Error converting pdf with %q err: %v", cmd.Args, err)
		}
		bs, err := ioutil.ReadFile(tmpName)
		if err == nil && len(bs) > 80 { // Sanity check that at least 80 characters where found.
			Debugf("Found text of length %d in pdf", len(bs))
			return string(bs), nil
		}
	}
	Debugf("Unable to get text from %q with pdftotext", file)
	return ocrImageFile(file)
}

func (p *processor) registerDefaults() {
	p.defaultMimeTypeHandlers = map[string]FileTranslator{
		"text":                   getPlainTextContent,
		"image":                  ocrImageFile,
		"application/javascript": getPlainTextContent,
		"application/json":       getPlainTextContent,
		"application/pdf":        getPdfText,
		"audio/mp3":              getAudioText,
		"audio/mp4a-latm":        getAudioText,
	}

}

func NewProcessor(hashDir string, index Index, force bool) FileProcessor {
	p := &processor{hashDir: hashDir, Index: index, force: force}
	p.registerDefaults()
	return p
}

// Register registers a mime type with a FileTranslator.
func (p *processor) Register(mime string, ft FileTranslator) error {
	if _, exists := p.defaultMimeTypeHandlers[mime]; exists {
		return fmt.Errorf("Attempt to register already existing mime type FileTranslator %q", mime)
	}
	p.defaultMimeTypeHandlers[mime] = ft
	return nil
}

func hashFile(file string) ([]byte, error) {
	h := sha256.New()
	f, err := os.Open(file)
	defer f.Close()
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(h, f)
	if err != nil {
		return nil, err
	}
	return h.Sum([]byte{}), nil
}

func (p *processor) checkHash(file string, hash []byte) (bool, error) {
	hashFile := path.Join(p.hashDir, hashFileName(file))
	Debugf("Checking for hashfile %q", hashFile)
	if _, err := os.Stat(hashFile); os.IsNotExist(err) {
		return false, nil
	}
	f, err := os.Open(hashFile)
	defer f.Close()
	if err != nil {
		return false, err
	}
	bs, err := ioutil.ReadAll(f)
	if err != nil {
		return false, err
	}
	if len(bs) != len(hash) {
		return false, nil
	}
	for i, b := range bs {
		if b != hash[i] {
			return false, nil
		}
	}
	return true, nil
}

func (p *processor) finishFile(file string) error {
	h, err := hashFile(file)
	if err != nil {
		return err
	}

	if _, err := os.Stat(p.hashDir); os.IsNotExist(err) {
		if err := os.MkdirAll(p.hashDir, os.ModeDir|os.ModePerm); err != nil {
			return err
		}
	}

	fd, err := os.Create(filepath.Join(p.hashDir, hashFileName(file)))
	defer fd.Close()
	if err != nil {
		return err
	}

	_, err = fd.Write(h)
	return err
}

// ShouldProcess returns true, nil if the file should be processed.
// false, error if it should not be processed.
func (p *processor) ShouldProcess(file string) (bool, error) {
	if strings.HasPrefix(file, ".") {
		return false, printError("not processing hidden file %q", file)
	}
	fi, err := os.Stat(file)
	if _, mt, ok := p.checkMimeType(file); !ok {
		return ok, printError("unhandled FileType: %q", mt)
	}
	if p.force && fi.Size() > *maxFileSize {
		return false, printError("file too large to index %q size=(%d)", file, fi.Size())
	}

	h, err := hashFile(file)
	if err != nil {
		return false, err
	}
	if ok, _ := p.checkHash(file, h); ok {
		Debugf("Already indexed %q", file)
		return false, nil
	}
	return true, nil
}

func (p *processor) checkMimeType(file string) (FileTranslator, string, bool) {
	// TODO(jwall): Do I want to do anything with the params?
	// Check to see if this is a handleable file
	mt, _, err := mime.ParseMediaType(mime.TypeByExtension(filepath.Ext(file)))
	if err != nil {
		return nil, mt, false
	}
	parts := strings.SplitN(mt, "/", 2)
	if ft, exists := p.defaultMimeTypeHandlers[mt]; exists {
		return ft, mt, exists
	} else if ft, exists := p.defaultMimeTypeHandlers[parts[0]]; exists {
		return ft, mt, exists
	} else {
		return nil, mt, false //fmt.Errorf("Unhandled file format %q", mt)
	}
}

// Process indexes a file.
func (p *processor) Process(file string) error {
	fi, err := os.Stat(file)
	if os.IsNotExist(err) {
		return err // In theory this will never happen
	}
	ft, mt, ok := p.checkMimeType(file)
	if !ok {
		return printError("unhandled file format %q", mt)
	}

	var ifile IFile
	fd := FileData{}
	fd.FileName = filepath.Base(file)
	fd.FullPath = path.Clean(file)
	fd.IndexTime = time.Now()
	fd.Size = fi.Size()

	fd.MimeType = mt
	fd.Text, err = ft(file)
	if err != nil {
		return err
	}

	if mt == "audio/mp3" || mt == "audio/mp4a-latm" {
		audio := AudioData{}
		audio.FileData = &fd
		audio.Analyse()
		ifile = &audio
	} else {
		ifile = &fd
	}

	parts := strings.SplitN(mt, "/", 2)
	Debugf("Detected mime category: %q", parts[0])
	Debugf("Indexing %q", ifile.Path())
	if err := p.Put(&ifile); err != nil {
		return err
	}
	return p.finishFile(ifile.Path())
}
