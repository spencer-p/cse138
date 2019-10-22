// Package leader implements handlers for follower instances.
package follower

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/spencer-p/cse138/pkg/types"

	"github.com/gorilla/mux"
)

const (
	TIMEOUT = 5 * time.Second

	MainFailure = "Main instance is down"
)

// follower holds all state that a follower needs to operate.
type follower struct {
	client http.Client
	addr   *url.URL
}

func (f *follower) indexHandler(w http.ResponseWriter, r *http.Request) {
	requestBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println("Failed to read body:", err)
		http.Error(w, "Failed to read request", http.StatusInternalServerError)
		return
	}

	target := *f.addr
	target.Path = path.Join(target.Path, r.URL.Path)

	request, err := http.NewRequest(r.Method,
		target.String(),
		bytes.NewBuffer(requestBody))
	if err != nil {
		log.Println("Failed to make proxy request:", err)
		http.Error(w, "Failed to make request", http.StatusInternalServerError)
		return
	}

	request.Header = r.Header.Clone()

	resp, err := f.client.Do(request)
	if err != nil {
		log.Println("Failed to do proxy request:", err)
		// Presumably the leader is down.
		result := types.Response{
			Status: http.StatusServiceUnavailable,
			Error:  MainFailure,
		}
		result.Serve(w, request)
		return
	}

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func Route(r *mux.Router, fwd string) error {
	addr, err := url.Parse(fwd)
	if err != nil {
		return fmt.Errorf("Bad forwarding address %q: %v\n", fwd, addr)
	}

	f := follower{
		client: http.Client{
			Timeout: TIMEOUT,
		},
		addr: addr,
	}

	r.PathPrefix("/").Handler(http.HandlerFunc(f.indexHandler))
	return nil
}
