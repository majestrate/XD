package transmission

type Request struct {
	Method string `json:"method"`
	Args   Args   `json:"arguments"`
	Tag    Tag    `json:"tag"`
}
