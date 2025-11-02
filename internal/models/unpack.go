package models

type Unpack struct {
	Packages []Packages `json:"packages"`
}

type Packages struct {
	Name string `json:"name"`
	Ver  string `json:"ver,omitempty"`
}
