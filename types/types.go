package types

type OriginResponse struct {
	Identifier string `json:"identifier"`
	ServerTime string `json:"server_time"`
	Token      string `json:"token"`
}
