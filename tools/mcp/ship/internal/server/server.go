package server

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const protocolVersion = "2024-11-05"

// Run serves a minimal MCP server over stdio.
func Run(ctx context.Context, in io.Reader, out io.Writer, log io.Writer, docsRoot string) error {
	s := &mcpServer{docsRoot: docsRoot, out: out, log: log}
	reader := bufio.NewReader(in)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		body, err := readMessage(reader)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}

		var req rpcRequest
		if err := json.Unmarshal(body, &req); err != nil {
			_ = s.writeError(nil, -32700, "parse error")
			continue
		}

		if len(req.ID) == 0 {
			// Notification: accept and ignore unknown notifications.
			continue
		}

		if err := s.handleRequest(req); err != nil {
			return err
		}
	}
}

type mcpServer struct {
	docsRoot string
	out      io.Writer
	log      io.Writer
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcErrorObject `json:"error,omitempty"`
}

type rpcErrorObject struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (s *mcpServer) handleRequest(req rpcRequest) error {
	switch req.Method {
	case "initialize":
		result := map[string]any{
			"protocolVersion": protocolVersion,
			"capabilities": map[string]any{
				"tools": map[string]any{},
			},
			"serverInfo": map[string]any{
				"name":    "ship-mcp",
				"version": "0.1.0",
			},
		}
		return s.writeResult(req.ID, result)
	case "ping":
		return s.writeResult(req.ID, map[string]any{})
	case "tools/list":
		return s.writeResult(req.ID, map[string]any{"tools": toolDefinitions()})
	case "tools/call":
		result, err := s.handleToolsCall(req.Params)
		if err != nil {
			return s.writeError(req.ID, -32602, err.Error())
		}
		return s.writeResult(req.ID, result)
	case "shutdown":
		return s.writeResult(req.ID, map[string]any{})
	default:
		return s.writeError(req.ID, -32601, "method not found")
	}
}

func (s *mcpServer) writeResult(id json.RawMessage, result any) error {
	return s.writeResponse(rpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	})
}

func (s *mcpServer) writeError(id json.RawMessage, code int, message string) error {
	return s.writeResponse(rpcResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &rpcErrorObject{
			Code:    code,
			Message: message,
		},
	})
}

func (s *mcpServer) writeResponse(resp rpcResponse) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}

	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))
	if _, err := io.WriteString(s.out, header); err != nil {
		return err
	}
	_, err = s.out.Write(data)
	return err
}

func readMessage(r *bufio.Reader) ([]byte, error) {
	contentLength := -1
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		name := strings.TrimSpace(strings.ToLower(parts[0]))
		value := strings.TrimSpace(parts[1])
		if name == "content-length" {
			n, err := strconv.Atoi(value)
			if err != nil {
				return nil, fmt.Errorf("invalid content-length: %w", err)
			}
			contentLength = n
		}
	}

	if contentLength < 0 {
		return nil, errors.New("missing content-length header")
	}

	body := make([]byte, contentLength)
	if _, err := io.ReadFull(r, body); err != nil {
		return nil, err
	}
	return body, nil
}
