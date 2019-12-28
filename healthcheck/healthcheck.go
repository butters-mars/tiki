package healthcheck

import "net/http"

var offline = false

// SetOffline set offline mode of healthcheck
func SetOffline(off bool) {
	offline = off
}

// Handler returns a "/healthcheck" HTTP GET handler
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if offline {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("OFFLINE"))
		} else {
			w.Write([]byte("OK"))
		}
	})
}
