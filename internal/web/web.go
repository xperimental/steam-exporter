package web

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

var (
	//go:embed home.html
	homepageContents string
)

func Handler(log logrus.FieldLogger, gitCommit, version string) http.Handler {
	r := mux.NewRouter()
	r.Handle("/", homeHandler())
	r.Handle("/metrics", promhttp.Handler())
	r.Handle("/version", versionHandler(log, gitCommit, version))
	return r
}

func homeHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, homepageContents)
	})
}

func versionHandler(log logrus.FieldLogger, gitCommit, version string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		version := struct {
			Commit  string `json:"commit"`
			Version string `json:"version"`
		}{
			Commit:  gitCommit,
			Version: version,
		}
		if err := json.NewEncoder(w).Encode(version); err != nil {
			log.Errorf("Error writing version information: %s", err)
		}
	})
}
