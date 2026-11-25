package mcpx

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/lhp2048/MXOnePeer/go/internal/adminboard"
	"github.com/lhp2048/MXOnePeer/go/internal/store"
)

const preferred = "2025-03-26"

var protocols = map[string]bool{
	"2024-11-05": true,
	"2025-03-26": true,
	"2025-06-18": true,
}

func Handle(w http.ResponseWriter, r *http.Request, board *adminboard.Board, stateDir string) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, "POST required\n", http.StatusMethodNotAllowed)
		return
	}
	if !strings.Contains(strings.ToLower(r.Header.Get("Content-Type")), "application/json") {
		writeError(w, http.StatusUnsupportedMediaType, nil, -32600, "application/json required", preferred)
		return
	}
	accept := r.Header.Get("Accept")
	if !strings.Contains(accept, "application/json") && !strings.Contains(accept, "text/event-stream") {
		writeError(w, http.StatusNotAcceptable, nil, -32600, "Accept must include application/json or text/event-stream", preferred)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<16))
	if err != nil {
		writeError(w, http.StatusBadRequest, nil, -32700, "Parse error", preferred)
		return
	}
	var message map[string]any
	if err := json.Unmarshal(body, &message); err != nil {
		writeError(w, http.StatusBadRequest, nil, -32700, "Parse error", preferred)
		return
	}
	method, _ := message["method"].(string)
	if message["jsonrpc"] != "2.0" || method == "" {
		writeError(w, http.StatusBadRequest, message["id"], -32600, "Invalid Request", preferred)
		return
	}
	requested := r.Header.Get("MCP-Protocol-Version")
	if method != "initialize" && requested != "" && !protocols[requested] {
		writeError(w, http.StatusBadRequest, message["id"], -32600, "Unsupported protocol version", preferred)
		return
	}
	version := versionOf(message, requested)
	if _, ok := message["id"]; !ok {
		w.Header().Set("MCP-Protocol-Version", version)
		w.WriteHeader(http.StatusAccepted)
		return
	}
	result, rpcErr := dispatch(method, message, board, stateDir)
	if rpcErr != nil {
		writeError(w, http.StatusOK, message["id"], rpcErr.code, rpcErr.message, version)
		return
	}
	writeResult(w, message["id"], result, version)
}

func dispatch(method string, message map[string]any, board *adminboard.Board, stateDir string) (any, *rpcError) {
	switch method {
	case "initialize":
		return map[string]any{
			"protocolVersion": agree(message),
			"capabilities":    map[string]any{"tools": map[string]any{"listChanged": false}},
			"serverInfo":      map[string]any{"name": "MXOnePeer", "version": "0.1.0"},
			"instructions":    "MXOnePeer 的本机登录控制。status 查看节点。set_hostname 保存节点名称。set_auth_key 保存 auth key。start_login 开始账号登录。",
		}, nil
	case "ping", "logging/setLevel":
		return map[string]any{}, nil
	case "tools/list":
		return tools(), nil
	case "tools/call":
		return call(message, board, stateDir)
	default:
		return nil, &rpcError{-32601, "Method not found"}
	}
}

func tools() map[string]any {
	empty := map[string]any{"type": "object", "properties": map[string]any{}, "additionalProperties": false}
	key := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"auth_key": map[string]any{"type": "string"},
		},
		"required":             []string{"auth_key"},
		"additionalProperties": false,
	}
	hostname := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"hostname": map[string]any{"type": "string"},
		},
		"required":             []string{"hostname"},
		"additionalProperties": false,
	}
	return map[string]any{"tools": []any{
		map[string]any{"name": "status", "description": "查看本机节点是否在线、当前名称、要申请的名称、Tailscale 地址、auth key 是否已配置，以及账号登录链接。", "inputSchema": empty},
		map[string]any{"name": "set_hostname", "description": "保存节点名称并重新连接。使用小写字母、数字和连字符，长度 1 到 63。", "inputSchema": hostname},
		map[string]any{"name": "set_auth_key", "description": "保存 Tailscale auth key。必须以 tskey-auth- 开头。", "inputSchema": key},
		map[string]any{"name": "start_login", "description": "开始 Tailscale 账号登录。登录链接可能要过几秒才出现，之后再调用 status。", "inputSchema": empty},
	}}
}

func call(message map[string]any, board *adminboard.Board, stateDir string) (any, *rpcError) {
	params, _ := message["params"].(map[string]any)
	name, _ := params["name"].(string)
	if params == nil || name == "" {
		return nil, &rpcError{-32602, "Invalid params"}
	}
	arguments, _ := params["arguments"].(map[string]any)
	if params["arguments"] != nil && arguments == nil {
		return nil, &rpcError{-32602, "Invalid params"}
	}
	if arguments == nil {
		arguments = map[string]any{}
	}
	switch name {
	case "status":
		board.Refresh(stateDir)
		return toolText(board.StatusJSON(), false), nil
	case "set_hostname":
		value, _ := arguments["hostname"].(string)
		previous := store.ReadHostname(stateDir)
		saved, err := store.WriteHostname(stateDir, value)
		if err != nil {
			return toolText("节点名称要用小写字母、数字和连字符，长度 1 到 63，并且不能用连字符开头或结尾。", true), nil
		}
		if saved == previous {
			board.SetNotice("节点名称没有变化。")
		} else {
			board.SetNotice("节点名称已保存，正在连接。")
			board.Poke("name")
		}
		board.Refresh(stateDir)
		return toolText(board.StatusJSON(), false), nil
	case "set_auth_key":
		value, _ := arguments["auth_key"].(string)
		if err := store.WriteAuthKey(stateDir, value); err != nil {
			return toolText("auth key 必须以 tskey-auth- 开头。", true), nil
		}
		board.ClearLogin()
		board.SetNotice("auth key 已保存，正在连接。")
		board.Poke("key")
		board.Refresh(stateDir)
		return toolText(board.StatusJSON(), false), nil
	case "start_login":
		board.SetLoginError("")
		board.SetNotice("正在向 Tailscale 申请登录链接。")
		board.Poke("login")
		board.Refresh(stateDir)
		return toolText(board.StatusJSON(), false), nil
	default:
		return toolText("Unknown tool", true), nil
	}
}

func toolText(text string, isError bool) map[string]any {
	return map[string]any{
		"content": []any{map[string]any{"type": "text", "text": text}},
		"isError": isError,
	}
}

func agree(message map[string]any) string {
	params, _ := message["params"].(map[string]any)
	version, _ := params["protocolVersion"].(string)
	if protocols[version] {
		return version
	}
	return preferred
}

func versionOf(message map[string]any, requested string) string {
	if message["method"] == "initialize" {
		return agree(message)
	}
	if protocols[requested] {
		return requested
	}
	return preferred
}

func writeResult(w http.ResponseWriter, id any, result any, version string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("MCP-Protocol-Version", version)
	_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": id, "result": result})
}

func writeError(w http.ResponseWriter, status int, id any, code int, message, version string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("MCP-Protocol-Version", version)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"error":   map[string]any{"code": code, "message": message},
	})
}

type rpcError struct {
	code    int
	message string
}
