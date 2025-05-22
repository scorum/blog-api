package rpc

import (
	"encoding/json"
	"fmt"
)

type Request struct {
	Method string `json:"method"`
	Params Params `json:"params"`
	ID     uint64 `json:"id"`
}

type Params struct {
	API    string             `json:"api"`
	Method string             `json:"method"`
	Args   []*json.RawMessage `json:"args"`
}

func (p *Params) UnmarshalJSON(buf []byte) (err error) {
	tmp := []interface{}{&p.API, &p.Method, &p.Args}
	wantLen := len(tmp)
	if err := json.Unmarshal(buf, &tmp); err != nil {
		println(err.Error())
		return err
	}
	if g, e := len(tmp), wantLen; g != e {
		return fmt.Errorf("wrong number args: %d != %d", g, e)
	}
	return nil
}
