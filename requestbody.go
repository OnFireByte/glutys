package glutys

import "encoding/json"

type RequestBody struct {
	Method string            `json:"method"`
	Args   []json.RawMessage `json:"args"`
}
