package web

import "net/http"

type Deps struct {
	Token http.Handler
}

func NewHandler(deps Deps) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	if deps.Token != nil {
		mux.Handle("GET /token", deps.Token)
	}
	return mux
}
