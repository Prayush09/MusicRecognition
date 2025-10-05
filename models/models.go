package models

type Couple struct {
	AnchorTime uint32
	SongId     uint32
}

type RecordData struct {
	Audio      string  `json:"audio"`
	Duration   float64 `json:"duration"`
	Channels   int     `json:"channels"`
	SampleRate int  	`json:"sample_rate"`
	SampleSize int   	`json:"sample_size"`
}
