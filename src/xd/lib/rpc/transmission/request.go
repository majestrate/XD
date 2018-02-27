package transmission

type Request struct {
	Method string                 `json:"method"`
	Args   map[string]interface{} `json:"arguments"`
	Tag    interface{}            `json:"tag"`
}
