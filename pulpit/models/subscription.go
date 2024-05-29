package models

type Subscription struct {
	Ns      string `json:"ns"`
	Owner   string `json:"owner"`
	Address string `json:"address"`
}
