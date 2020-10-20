package zenikanard

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"sync"

	log "github.com/sirupsen/logrus"
)

type imageTranscoder struct {
	command     string
	commandArgs []string
	inputType   int
}

const (
	fileInput int = iota
	stdInput
)

var (
	ViuTranscoder     = imageTranscoder{command: "viu", commandArgs: []string{"-w", "40", "-t"}, inputType: stdInput}
	PixtermTranscoder = imageTranscoder{command: "pixterm", commandArgs: []string{"-tc", "30", "-s", "2"}, inputType: fileInput}
	Img2txtTranscoder = imageTranscoder{command: "img2txt", commandArgs: []string{"-d", "none", "-W", "30", "-f", "ansi"}, inputType: fileInput}
	Transcoder        imageTranscoder
)

// Zenikanard is the struct passed around the workers
type Zenikanard struct {
	URL      string
	Name     string
	PNGData  []byte
	ANSIData []byte
}

// Zenikanards represents a list of Zenikanards
type Zenikanards struct {
	sync.RWMutex
	List []*Zenikanard
}

// DownloadImage fetches zenikanard png image
func (z *Zenikanard) DownloadImage() error {
	log.Debug("download: ", z.Name, z.URL)
	response, err := http.Get(z.URL)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return errors.New(z.URL + " -> status != 200")
	}

	z.PNGData, err = ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	log.Debugf("downloaded: %s %s %d", z.Name, z.URL, len(z.PNGData))
	return nil
}

// LoadFromCache tries to get zenikanard from local cache
func (z *Zenikanard) LoadFromCache(cacheDir string) bool {
	data, err := ioutil.ReadFile(cacheDir + "/" + z.Name)
	if err != nil {
		return false
	}

	z.ANSIData = data
	return true
}

// TranscodePNGToANSI takes png image input and outputs ansi for the terminal
// It launches a transcoder program and pass the png image
// The output is recovered from the program stdout
func (z *Zenikanard) TranscodePNGToANSI(cacheEnabled bool, cacheDir string) error {
	log.Debug("transcode: ", z.Name, z.URL, len(z.PNGData))

	var out io.Writer
	ansiData := new(bytes.Buffer)

	// If cache enabled, write transcoding output to both cache file and zenikanard struct
	if cacheEnabled {
		file, err := os.Create(cacheDir + "/" + z.Name)
		if err != nil {
			return err
		}
		out = io.MultiWriter(file, ansiData)
	} else {
		out = ansiData
	}

	var cmdArgs []string

	// If the transcoder program supports passing image via stdin append -
	// If it supports passing the image via a file we need to create a temporary file first
	switch Transcoder.inputType {
	case stdInput:
		cmdArgs = append(Transcoder.commandArgs, "-")
	case fileInput:
		// https://golangcode.com/creating-temp-files/
		tmpFile, err := ioutil.TempFile(os.TempDir(), "prefix-")
		if err != nil {
			return err
		}

		defer os.Remove(tmpFile.Name())

		if _, err = tmpFile.Write(z.PNGData); err != nil {
			return err
		}

		// Close the file
		if err := tmpFile.Close(); err != nil {
			return err
		}

		cmdArgs = append(Transcoder.commandArgs, tmpFile.Name())
	}

	p := exec.Command(Transcoder.command, cmdArgs...)
	if Transcoder.inputType == stdInput {
		p.Stdin = bytes.NewReader(z.PNGData)
	}

	p.Stdout = out
	p.Stderr = os.Stderr

	// Some programs base their transcoding option on some terminal capabilities
	// viu in particular will use certain control sequences according to the COLORTERM env var
	// To make sure we output sequences compatible with most terminals, clear this variable
	p.Env = append(os.Environ(), "COLORTERM=")
	err := p.Run()
	if err != nil {
		return err
	}

	z.ANSIData = ansiData.Bytes()

	log.Debug("transcoded: ", z.Name, z.URL, len(z.PNGData), len(z.ANSIData), ansiData.Len())
	return nil
}
