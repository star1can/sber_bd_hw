package model

import "encoding/json"

type Player struct {
	Id    string  `json:"id"`
	Score float64 `json:"score"`
}

type RankedPlayer struct {
	Id    string  `json:"player"`
	Score float64 `json:"score"`
	Rank  int64   `json:"rank"`
}

type LeaderboardResponse struct {
	Players []Player `json:"players"`
	Count   int64    `json:"count"`
}

type UpdateScoreRequest struct {
	Id    string  `json:"id"`
	Score float64 `json:"score"`
}

func (p Player) MarshalBinary() ([]byte, error) {
	return json.Marshal(p)
}
