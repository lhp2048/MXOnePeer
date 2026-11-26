package clientapi

import (
	"net/http"
	"strings"
)

func StatusURL(ip string) string {
	host := ip
	if strings.Contains(ip, ":") {
		host = "[" + ip + "]"
	}
	return "http://" + host + ":18020/status"
}

func ReadStatus(code int, body []byte) ([]byte, bool) {
	if code != http.StatusOK || !jsonObject(body) {
		return nil, false
	}
	return body, true
}
