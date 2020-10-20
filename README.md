# Zenikanard terminal display

Transcode zenikanard png images to ansi sequences viewable in a terminal

![zenikanard-png](docs/zenikanard.png)

to

![zenikanard-ansi](docs/zenikanard-ansi.png)

## Description

This project first fetches the zenikanards from https://theduckgallery.zenika.com/ using https://github.com/mxschmitt/playwright-go

It then transcodes the png image using a transcoding program (see below).

Finally a webserver is started providing two routes:

- GET /[github username] -> Get zenikanard of the provided github username
- GET / -> circles through all the gallery zenikanard

Display depends on the terminal used.
The default transcoding program (viu) should output sequences viewable in most terminal.

## Quickstart

### Dependencies

You need one of the following transcoding backend installed:

- [viu](https://github.com/atanunq/viu) - Follow repo installation instructions
- [img2txt](https://github.com/atanunq/viu) - Debian: apt-get install caca-utils on Debian
- [pixterm](https://github.com/eliukblau/pixterm) - go get -u github.com/eliukblau/pixterm/cmd/pixterm

You also need an OS compatible with [playwright-go](https://github.com/mxschmitt/playwright-go).
x86-64 machines Windows/Mac/Linux should be ok.

Tested only on linux.

### Run

`go run main.go`

On another terminal:

`curl http://localhost:8080`

If using another backend then viu you need to specify at the cli:

`go run main.go -image-transcoder img2txt`

### Usage

```
Usage of ansi-zenikanard:
  -cache
    use local cache (default true)
  -cache-dir string
    cache directory to store zenikanards (default "./cache")
  -cache-only
    don't scrape website, use only local cache. Useful on machine where playwright is not supported, raspi for example. Assumes cache enabled
  -h help
  -image-transcoder string
    program to transcode png to ansi. one of viu, img2txt, pixterm (default "viu")
  -listen-addr string
    adress and port fed to http.ListenAndServe (default ":8080")
  -transition-time int
    time to sleep between zenikanard in millisecond (default 500)
  -v enable debug output
```

## Internals

playwright-go is used to [scrape](./scrape/scrape.go) the duck gallery.
Playwright launches a headless browser which loads the duck gallery.
A query selector is then used to get all img tag corresponding to zenikanards.

I first tried to download the main page and parse it manually but the list of zenikanard is downloaded with Javascript,
so it is not available in the base html.

I could have downloaded the https://theduckgallery.zenika.com/contributors.js and parsed the zenikanard list from there but it was less fun than using playwright :).

Once the list of zenikanard is obtained, each zenikanard is sent on a download channel.

A pool of download worker is started and each [worker](./worker/worker.go) listens on the download channel.
When it receives a zenikanard it first checks if it is available in a local cache.
If not it downloads it using its URL.

If the zenikanard is not recovered from cache, the downloaded image still needs to be converted to ANSI.
In this case it is sent through another channel: the transcoding channel.

Behind this channel is another pool of workers.
When a transcode worker receives a zenikanard it launches an external process and passes the png image to it.
The resulting ansi output is stored in the zenikanard struct and in a local cache file if the cache is enabled

Once all zenikanards have been processed, a webserver is started with a single handler.

The handler will respond to GET "/" by iterating over all zenikanard and sending them one by one to the client

The handler also responds to GET "/[github username]" with a specific zenikanard if found
