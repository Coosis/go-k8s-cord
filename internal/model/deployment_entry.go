package model

type DeploymentEntry struct {
	Name     string `json:"name"`
	Checksum string `json:"checksum"`
}
