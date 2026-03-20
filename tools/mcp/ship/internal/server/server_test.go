package server

import (
	"bufio"
	"bytes"
	"fmt"
	"testing"
)

func TestReadMessage(t *testing.T) {
	t.Parallel()

	body := []byte(`{"jsonrpc":"2.0","id":1,"method":"ping"}`)
	data := []byte(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body))

	got, err := readMessage(bufio.NewReader(bytes.NewReader(data)))
	if err != nil {
		t.Fatalf("readMessage error: %v", err)
	}
	if string(got) != string(body) {
		t.Fatalf("got %q want %q", string(got), string(body))
	}
}

func TestReadMessageMissingLength(t *testing.T) {
	t.Parallel()

	_, err := readMessage(bufio.NewReader(bytes.NewReader([]byte("Header: x\r\n\r\n{}"))))
	if err == nil {
		t.Fatal("expected error")
	}
}
