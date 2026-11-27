package clientapi

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/lhp2048/MXOnePeer/go/internal/upgrade"
)

var (
	ErrCannotUpgrade = errors.New("这个节点不能升级")
	ErrBadFilename   = errors.New("文件名不正确")
	ErrUpload        = errors.New("传不上去")
)

func CheckUpgrade(ip, runtimeID, filename string, nodes []Node) error {
	if !Allowed(ip, nodes) {
		return ErrNodeAddress
	}
	if IsSelf(ip, nodes) || (!strings.HasPrefix(runtimeID, "go/") && !strings.HasPrefix(runtimeID, "python/")) {
		return ErrCannotUpgrade
	}
	if !upgrade.FilenameOK(runtimeID, filename) {
		return ErrBadFilename
	}
	return nil
}

func UpgradeURL(ip string) string {
	host := ip
	if strings.Contains(ip, ":") {
		host = "[" + ip + "]"
	}
	return "http://" + host + ":18020/upgrade"
}

func PostUpgrade(ctx context.Context, dial func(context.Context, string, string) (net.Conn, error), ip, runtimeID, filename string, payload []byte) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, UpgradeURL(ip), strings.NewReader(string(payload)))
	if err != nil {
		return http.StatusBadGateway, nil, ErrUpload
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("X-Mx-Runtime", runtimeID)
	req.Header.Set("X-Mx-Filename", filename)
	client := &http.Client{
		Transport: &http.Transport{DialContext: dial},
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return http.StatusBadGateway, nil, ErrUpload
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	switch resp.StatusCode {
	case http.StatusNoContent:
		return http.StatusNoContent, nil, nil
	case http.StatusBadRequest, http.StatusInternalServerError:
		return resp.StatusCode, body, nil
	default:
		return http.StatusBadGateway, nil, ErrUpload
	}
}
