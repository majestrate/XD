package transmission

type Response struct {
	Result string `json:"result"`
	Args   Args   `json:"arguments"`
	Tag    Tag    `json:"tag"`
}
