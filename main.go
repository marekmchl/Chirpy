package main

import "net/http"

func main() {
	serveMux := http.ServeMux{}
	serveMux.Handle("/app/", http.StripPrefix("/app/", http.FileServer(http.Dir("."))))
	serveMux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(200)
		w.Write([]byte{'O', 'K'})
	})
	server := http.Server{
		Addr:    ":8080",
		Handler: &serveMux,
	}
	server.ListenAndServe()
}
