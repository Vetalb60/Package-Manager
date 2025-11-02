package models

type Pack struct {
	Packets []Packets `json:"packets"`
}

type Packets struct {
	Name    string    `json:"name"`
	Ver     string    `json:"ver"`
	Targets []Targets `json:"targets"`
}

type Targets struct {
	Path    string `json:"path"`
	Exclude string `json:"exclude"`
}
