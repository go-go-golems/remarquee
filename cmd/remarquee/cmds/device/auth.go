package device

import "net/http"

func basicAuthMiddleware(next http.Handler, username, password string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != username || pass != password {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("Unauthorized"))
			return
		}
		next.ServeHTTP(w, r)
	})
}
