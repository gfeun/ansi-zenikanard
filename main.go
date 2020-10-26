package main

import (
	"flag"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/schollz/progressbar/v3"
	log "github.com/sirupsen/logrus"

	"github.com/gfeun/ansi-zenikanard/handler"
	"github.com/gfeun/ansi-zenikanard/scrape"
	"github.com/gfeun/ansi-zenikanard/worker"
	"github.com/gfeun/ansi-zenikanard/zenikanard"
)

var (
	zenikanards = zenikanard.Zenikanards{}
	bar         *progressbar.ProgressBar
)

// CLI Flags
var (
	listenAddr               string
	cacheEnabled             bool
	cacheOnly                bool
	cacheDir                 string
	timeBetweenZenikanard    int
	imageTranscoder          string
	playwrightBrowserInstall bool
	verbose                  bool
	help                     bool
)

var (
	downloadTasksChan    = make(chan *zenikanard.Zenikanard, 8)
	transcodingTasksChan = make(chan *zenikanard.Zenikanard, 8)
	doneChan             = make(chan *zenikanard.Zenikanard, 1)
)

// downloadZenikanard gets a zenikanard image from its URL then forwards it to the transcoding worker
// if the cache is enabled, it first tries to get the zenikanard from a local file and if successfull goes directly to the done worker
func downloadZenikanard() {
	for z := range downloadTasksChan {
		if cacheEnabled && z.LoadFromCache(cacheDir) {
			log.Debug("got ", z.Name, " from cache")
			doneChan <- z
			continue
		}
		if err := z.DownloadImage(); err != nil {
			log.Errorf("could not download: %v", err)
			continue
		}
		transcodingTasksChan <- z
	}
}

// transcodeImage gets a zenikanard png image and transform it to an ANSI commands sequence
func transcodeImage() {
	for z := range transcodingTasksChan {
		if err := z.TranscodePNGToANSI(cacheEnabled, cacheDir); err != nil {
			log.Errorf("transcode error: %v", err)
			continue
		}
		doneChan <- z
	}
}

// doneWorker just increments the progress bar
func done() {
	for range doneChan {
		_ = bar.Add(1)
	}
}

func main() {
	flag.StringVar(&listenAddr, "listen-addr", ":8080", "adress and port fed to http.ListenAndServe")
	flag.BoolVar(&cacheEnabled, "cache", true, "use local cache")
	flag.BoolVar(&cacheOnly, "cache-only", false, "don't scrape website, use only local cache. Useful on machine where playwright is not supported, raspi for example. Assumes cache enabled")
	flag.StringVar(&cacheDir, "cache-dir", "./cache", "cache directory to store zenikanards")
	flag.IntVar(&timeBetweenZenikanard, "transition-time", 500, "time to sleep between zenikanard in millisecond")
	flag.StringVar(&imageTranscoder, "image-transcoder", "viu", "program to transcode png to ansi. one of viu, img2txt, pixterm")
	flag.BoolVar(&playwrightBrowserInstall, "playwright-install", false, "install browsers")
	flag.BoolVar(&verbose, "v", false, "enable debug output")
	flag.BoolVar(&help, "h", false, "help")

	flag.Parse()

	if help {
		flag.Usage()
		return
	}

	if verbose {
		log.SetLevel(log.DebugLevel)
	}

	if playwrightBrowserInstall {
		if err := scrape.InitPlaywright(); err != nil {
			log.Fatal(err)
		}
		return
	}

	if cacheOnly {
		cacheEnabled = true
	}

	if cacheEnabled {
		err := os.Mkdir(cacheDir, 0755)
		if err != nil && !os.IsExist(err) {
			log.Fatal(err)
		}
	}

	switch imageTranscoder {
	case "viu":
		zenikanard.Transcoder = zenikanard.ViuTranscoder
	case "pixterm":
		zenikanard.Transcoder = zenikanard.PixtermTranscoder
	case "img2txt":
		zenikanard.Transcoder = zenikanard.Img2txtTranscoder
	default:
		log.Fatal("-image-transcoder takes one of viu, pixterm, img2txt")
	}

	// Setup background workers
	downloadWorkers := worker.New(downloadZenikanard, 8)
	transcodingWorkers := worker.New(transcodeImage, 8)
	doneWorker := worker.New(done, 1)

	downloadWorkers.Run()
	transcodingWorkers.Run()
	doneWorker.Run()

	if cacheOnly {
		log.Info("Get zenikanards list from local cache: " + cacheDir)
		// enumerate file in cache directory, create zenikanard with file name
		dir, err := ioutil.ReadDir(cacheDir)
		if err != nil {
			log.Fatal(err)
		}
		zenikanards.Lock()
		for _, file := range dir {
			zenikanards.List = append(zenikanards.List, &zenikanard.Zenikanard{Name: file.Name()})
		}
		zenikanards.Unlock()
	} else {
		log.Info("Get zenikanard list from zenikanard gallery website")
		err := scrape.FetchZenikanards(&zenikanards)
		if err != nil {
			log.Fatal(err)
		}
	}

	bar = progressbar.Default(int64(len(zenikanards.List)))

	for i := range zenikanards.List {
		downloadTasksChan <- zenikanards.List[i]
	}

	// All work items have been sent, close channels and wait for the workers to finish
	close(downloadTasksChan)
	downloadWorkers.Wait()

	close(transcodingTasksChan)
	transcodingWorkers.Wait()

	close(doneChan)
	doneWorker.Wait()

	log.Info("Init done, all zenikanards loaded, start webserver...")
	handler := handler.NewZenikanardHandler(&zenikanards, timeBetweenZenikanard)
	http.DefaultServeMux.Handle("/", handler)

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
