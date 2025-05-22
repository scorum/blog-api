package rpc

import (
	"encoding/json"
	"fmt"
	"net/http"

	protocol "github.com/scorum/scorum-go/transport"
	log "github.com/sirupsen/logrus"
)

type Context struct {
	ID      uint64
	Request *http.Request
	Writer  http.ResponseWriter
	params  *Params
	log     *log.Entry

	// indicates that the context, has been flushed: response is already written
	flushed bool
}

func NewContext(req *http.Request, writer http.ResponseWriter) *Context {
	return &Context{
		Request: req,
		Writer:  writer,
		log:     log.NewEntry(log.StandardLogger()),
	}
}

// API returns API name
func (c Context) API() string {
	return c.params.API
}

// Method returns method name
func (c Context) Method() string {
	return c.params.Method
}

func (c Context) Param(at int, p interface{}) error {
	if at >= len(c.params.Args) {
		return fmt.Errorf("no params at index %d", at)
	}

	arg := c.params.Args[at]
	if arg == nil {
		p = nil
		return nil
	}

	return json.Unmarshal(*arg, p)
}

// Parse RPC request
func (c *Context) Parse() (ok bool) {
	var rpcRequest Request
	if err := json.NewDecoder(c.Request.Body).Decode(&rpcRequest); err != nil {
		c.WriteError(InvalidRequestCode, fmt.Sprintf("message body is not an rpc request: %s", err.Error()))
		return false
	}

	c.ID = rpcRequest.ID
	c.params = &rpcRequest.Params

	if rpcRequest.Method != "call" {
		c.WriteError(InvalidRequestCode, "rpc request method should be call")
		return false
	}

	// set log
	var requestJson string
	if json, err := json.Marshal(rpcRequest); err != nil {
		requestJson = fmt.Sprintf("%+v", rpcRequest)
	} else {
		requestJson = string(json)
	}

	c.log = log.WithField("request", requestJson)
	return true
}

// WriteResult
func (c *Context) WriteResult(out interface{}) {
	if c.flushed {
		panic("context is flushed")
	}
	defer func() {
		c.flushed = true
	}()

	if out == nil {
		c.Writer.WriteHeader(http.StatusNoContent)
		return
	}

	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(http.StatusOK)

	bytes, err := json.Marshal(out)
	if err != nil {
		c.log.Fatal(err)
	}

	raw := json.RawMessage(bytes)
	json.NewEncoder(c.Writer).Encode(&protocol.RPCResponse{
		ID:     c.ID,
		Result: &raw,
	})
}

// WriteError
func (c *Context) WriteError(code int, message string) {
	if c.flushed {
		panic("context is flushed")
	}

	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.WriteHeader(http.StatusOK)

	if code == InternalErrorCode {
		c.log.Errorf("error: %s", message)
	} else {
		c.log.Debugf("error: %s, code: %d", message, code)
	}

	out := protocol.RPCResponse{
		ID: c.ID,
		Error: &protocol.RPCError{
			Code:    code,
			Message: message,
		},
	}

	if err := json.NewEncoder(c.Writer).Encode(out); err != nil {
		c.log.Fatal(err)
	}

	c.flushed = true
}
