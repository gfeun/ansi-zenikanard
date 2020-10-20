package handler

import (
	"bytes"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/gfeun/ansi-zenikanard/zenikanard"
)

// Escape code to start a control sequence can be \x1B (for hexadecimal code of ESC) or unicode code u+241B: ␛ 
var (
	resetSequence      = []byte("\x1B[?25l\x1B[2J\x1B[H\x1B[38;5;16m")
	lineReturnSequence = []byte("\x1B[E\x1B[100D")
)

// ZenikanardHandler implements ServeHTTP to serve zenikanards
type ZenikanardHandler struct {
	zenikanards           *zenikanard.Zenikanards
	timeBetweenZenikanard int
}

func NewZenikanardHandler(z *zenikanard.Zenikanards, timeBetweenZenikanard int) *ZenikanardHandler {
	return &ZenikanardHandler{zenikanards: z, timeBetweenZenikanard: timeBetweenZenikanard}
}

func writeZenikanard(w http.ResponseWriter, z *zenikanard.Zenikanard) error {
	log.Debug("write zenikanard")
	dataBuff := bytes.NewBuffer(resetSequence)

	// create a single []byte from several ones, including control sequences to reset the terminal and the zenikanard
	for _, arr := range [][]byte{resetSequence, lineReturnSequence, []byte("   " + z.Name), lineReturnSequence, z.ANSIData} {
		if _, err := dataBuff.Write(arr); err != nil {
			return err
		}
	}
	if _, err := w.Write(dataBuff.Bytes()); err != nil {
		return err
	}
	return nil
}

// ServeHTTP receives HTTP requests and answers either with one specific Zenikanard if url contains a zenikanard name
// or with all the zenikanards if requesting "/"
func (h *ZenikanardHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		if _, err := w.Write([]byte("Method not allowed")); err != nil {
			log.Error(err)
		}
		return
	}

	h.zenikanards.RLock()
	defer h.zenikanards.RUnlock()

	requestedZenikanardName := r.URL.Path[1:]
	// Check if a specific zenikanard has been requested
	if requestedZenikanardName != "" {
		// TODO: optim - use map instead of looping over every item
		for _, zenikanard := range h.zenikanards.List {
			if requestedZenikanardName == zenikanard.Name {
				_ = writeZenikanard(w, zenikanard)
				return
			}
		}

		w.WriteHeader(http.StatusNotFound)
		if _, err := w.Write([]byte("Zenikanard " + requestedZenikanardName + " not found")); err != nil {
			log.Error(err)
		}
		return
	}

	// Send all zenikanards
	for _, zenikanard := range h.zenikanards.List {
		if err := writeZenikanard(w, zenikanard); err != nil {
			return
		}
		if f, ok := w.(http.Flusher); ok {
			f.Flush() // Avoid buffering
		}
		time.Sleep(time.Duration(h.timeBetweenZenikanard) * time.Millisecond)
	}
}
