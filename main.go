package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
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

type rBody struct {
	Body string `json:"body"`
}

func FilterBadWords(chirp rBody) string {
	sl := strings.Split(chirp.Body, " ")
	for i, str := range sl {
		if str == "kerfuffle" || str == "sharbert" || str == "fornax" ||
			str == "Kerfuffle" || str == "Sharbert" || str == "Fornax" {
			sl[i] = "****"
		}
	}
	s := strings.Join(sl, " ")

	return s
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
	sm.HandleFunc("POST /api/validate_chirp", func(w http.ResponseWriter, r *http.Request) {
		// Decode request body
		decoder := json.NewDecoder(r.Body)
		body := rBody{}
		err := decoder.Decode(&body)
		if err != nil {
			log.Printf("Error decoding request body: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// If Chirp is too long
		if len(body.Body) > 140 {
			type wBody struct {
				Error string `json:"error"`
			}
			resposeBody := wBody{Error: "Chirp is too long"}
			dat, err := json.Marshal(resposeBody)
			if err != nil {
				log.Printf("Error marshalling JSON response: %s", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusBadRequest)
			w.Write(dat)
			return
		}

		cleanedString := FilterBadWords(body)

		// Happy path
		type wBody struct {
			CleanedBody string `json:"cleaned_body"`
		}
		resposeBody := wBody{CleanedBody: cleanedString}
		dat, err := json.Marshal(resposeBody)
		if err != nil {
			log.Printf("Error marshalling JSON response: %s", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Add("Content-type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(dat)
	})
	sm.HandleFunc("GET /admin/metrics", apiCfg.getMetrics)
	sm.HandleFunc("POST /admin/reset", apiCfg.resetMetrics)

	log.Fatal(server.ListenAndServe())
}
