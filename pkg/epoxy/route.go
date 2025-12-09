package epoxy

type Route struct {
	Prefix      string `json:"prefix"`
	Target      string `json:"target"`
	Strip       bool   `json:"strip,omitempty"`
	RewriteHost bool   `json:"rewrite_host,omitempty"`
}
