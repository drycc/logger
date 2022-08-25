package weblog

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"

	"github.com/drycc/logger/storage"
)

var (
	DRYCC_LOGS_MAXIMUM_LINES   = 300
	DRYCC_LOGS_MAXIMUM_TIMEOUT = 300
)

func init() {
	lines, err := strconv.Atoi(os.Getenv("DRYCC_LOGS_MAXIMUM_LINES"))
	if err == nil && lines > 0 {
		DRYCC_LOGS_MAXIMUM_LINES = lines
	}
	timeout, err := strconv.Atoi(os.Getenv("DRYCC_LOGS_MAXIMUM_TIMEOUT"))
	if err == nil && timeout > 0 {
		DRYCC_LOGS_MAXIMUM_TIMEOUT = timeout
	}
}

type requestHandler struct {
	storageAdapter storage.Adapter
}

func newRequestHandler(storageAdapter storage.Adapter) *requestHandler {
	return &requestHandler{
		storageAdapter: storageAdapter,
	}
}

func (h requestHandler) getHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (h requestHandler) getLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Connection", "Keep-Alive")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	flusher, ok := w.(http.Flusher)
	if !ok {
		panic("expected http.ResponseWriter to be an http.Flusher")
	}

	app := mux.Vars(r)["app"]
	var logLines int
	logLinesStr := r.URL.Query().Get("log_lines")
	if logLinesStr == "" {
		log.Printf("The number of lines to return was not specified. Defaulting to 100 lines.")
		logLines = DRYCC_LOGS_MAXIMUM_LINES
	} else {
		var err error
		logLines, err = strconv.Atoi(logLinesStr)
		if err != nil || logLines > DRYCC_LOGS_MAXIMUM_LINES || logLines < 0 {
			log.Printf("The specified number of log lines was invalid. Defaulting to 100 lines.")
			logLines = DRYCC_LOGS_MAXIMUM_LINES
		}
	}
	logs, err := h.storageAdapter.Read(app, logLines)
	if err != nil {
		log.Println(err)
		if strings.HasPrefix(err.Error(), "Could not find logs for") {
			w.WriteHeader(http.StatusNoContent)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}
	log.Printf("Returning the last %v lines for %s", logLines, app)
	for _, line := range logs {
		// strip any trailing newline characters from the logs
		fmt.Fprintf(w, "%s\n", strings.TrimSuffix(line, "\n"))
	}
	flusher.Flush()

	follow, err := strconv.ParseBool(r.URL.Query().Get("follow"))
	if err == nil && follow {
		timeout, err := strconv.Atoi(r.URL.Query().Get("timeout"))
		if err != nil || timeout > DRYCC_LOGS_MAXIMUM_TIMEOUT || timeout <= 0 {
			timeout = DRYCC_LOGS_MAXIMUM_TIMEOUT
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
		defer cancel()
		if channel, err := h.storageAdapter.Chan(ctx, app, 100); err == nil {
			for {
				line := <-channel
				if line == "" {
					break
				}
				fmt.Fprintf(w, "%s\n", strings.TrimSuffix(line, "\n"))
				flusher.Flush()
			}
		}
	}
	w.Header().Set("Content-Length", "0")
}

func (h requestHandler) deleteLogs(w http.ResponseWriter, r *http.Request) {
	app := mux.Vars(r)["app"]
	if err := h.storageAdapter.Destroy(app); err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}
