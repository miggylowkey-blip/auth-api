package middleware

import "net/http"

type responseRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (rr *responseRecorder) WriteHeader(code int) {
	rr.status = code
	rr.ResponseWriter.WriteHeader(code)
}

func (rr *responseRecorder) Write(b []byte) (int, error) {
	if rr.status == 0 {
		rr.status = 200
	}
	n, err := rr.ResponseWriter.Write(b)
	rr.bytes += n
	return n, err
}
