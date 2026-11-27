package upgrade

import (
	"io"
	"net/http"
)

const MaxBytes = 64 * 1024 * 1024
const ExitCode = 75

func NewHandler(runtimeID, program string, exit func(int)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed\n", http.StatusMethodNotAllowed)
			return
		}
		if r.Header.Get("X-Mx-Runtime") != runtimeID {
			http.Error(w, "runtime 不一致\n", http.StatusBadRequest)
			return
		}
		if !FilenameOK(runtimeID, r.Header.Get("X-Mx-Filename")) {
			http.Error(w, "文件名不正确\n", http.StatusBadRequest)
			return
		}
		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, MaxBytes))
		if err != nil {
			http.Error(w, "包的内容不正确\n", http.StatusBadRequest)
			return
		}
		if CheckBinary(runtimeID, body) != nil {
			http.Error(w, "包的内容不正确\n", http.StatusBadRequest)
			return
		}
		if ApplyFile(program, body) != nil {
			http.Error(w, "写不进去\n", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		exit(ExitCode)
	})
}

func Routes(status http.Handler, runtimeID, program string, exit func(int)) http.Handler {
	upgrade := NewHandler(runtimeID, program, exit)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/upgrade" {
			upgrade.ServeHTTP(w, r)
			return
		}
		if status == nil {
			http.NotFound(w, r)
			return
		}
		status.ServeHTTP(w, r)
	})
}
