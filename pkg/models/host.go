package models

type Host struct {
	Host   string            `json:"host"`
	Values map[string]string `json:"values"`
}
