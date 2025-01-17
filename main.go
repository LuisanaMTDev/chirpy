package main

import (
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)

		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) getMetrics(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Add("Content-type", "text/html; charset=utf-8")
	_, err := w.Write([]byte(fmt.Sprintf(`
		<html>
		  <body>
			<h1>Welcome, Chirpy Admin</h1>
			<p>Chirpy has been visited %d times!</p>
		  </body>
		</html>
		`, cfg.fileserverHits.Load())))

	if err != nil {
		log.Printf("Error while writing response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (cfg *apiConfig) resetMetrics(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)
	w.WriteHeader(http.StatusOK)
}

func main() {
	sm := http.NewServeMux()
	server := http.Server{Addr: ":8080", Handler: sm}
	apiCfg := apiConfig{}

	sm.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(".")))))
	sm.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("OK"))

		if err != nil {
			log.Printf("Error while writing response: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	})
	sm.HandleFunc("GET /admin/metrics", apiCfg.getMetrics)
	sm.HandleFunc("POST /admin/reset", apiCfg.resetMetrics)

	log.Fatal(server.ListenAndServe())
}
