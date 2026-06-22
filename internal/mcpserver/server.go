package mcpserver

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
)

type request struct {
	JSONRPC string          `json:"jsonrpc,omitempty"`
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type response struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      any            `json:"id,omitempty"`
	Result  any            `json:"result,omitempty"`
	Error   *responseError `json:"error,omitempty"`
}

type responseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func Serve(ctx context.Context, snapshot Snapshot, in io.Reader, out io.Writer) error {
	scanner := bufio.NewScanner(in)
	encoder := json.NewEncoder(out)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		var req request
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			if err := encoder.Encode(errorResponse(nil, -32700, "parse error: "+err.Error())); err != nil {
				return err
			}
			continue
		}
		resp := handleRequest(snapshot, req)
		if err := encoder.Encode(resp); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func handleRequest(snapshot Snapshot, req request) response {
	switch req.Method {
	case "initialize":
		return okResponse(req.ID, map[string]any{
			"protocolVersion": "2024-11-05",
			"serverInfo": map[string]string{
				"name":    "tailchase",
				"version": "local",
			},
			"capabilities": map[string]any{
				"resources": map[string]bool{"listChanged": false},
				"tools":     map[string]bool{"listChanged": false},
			},
		})
	case "resources/list":
		return okResponse(req.ID, map[string]any{"resources": snapshot.ResourceList()})
	case "resources/read":
		var params struct {
			URI string `json:"uri"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return errorResponse(req.ID, -32602, "invalid params: "+err.Error())
		}
		resource, err := snapshot.ReadResource(params.URI)
		if err != nil {
			return errorResponse(req.ID, -32602, err.Error())
		}
		return okResponse(req.ID, map[string]any{
			"contents": []map[string]string{
				{"uri": resource.URI, "mimeType": resource.MimeType, "text": resource.Text},
			},
		})
	case "tools/list":
		return okResponse(req.ID, map[string]any{"tools": snapshot.Tools})
	case "tools/call":
		var params struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return errorResponse(req.ID, -32602, "invalid params: "+err.Error())
		}
		text, err := snapshot.CallTool(params.Name)
		if err != nil {
			return errorResponse(req.ID, -32602, err.Error())
		}
		return okResponse(req.ID, map[string]any{
			"content": []map[string]string{
				{"type": "text", "text": text},
			},
		})
	default:
		return errorResponse(req.ID, -32601, fmt.Sprintf("method %q not found", req.Method))
	}
}

func okResponse(id any, result any) response {
	return response{JSONRPC: "2.0", ID: id, Result: result}
}

func errorResponse(id any, code int, message string) response {
	return response{JSONRPC: "2.0", ID: id, Error: &responseError{Code: code, Message: message}}
}
