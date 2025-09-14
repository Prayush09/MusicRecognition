package models

import (
	"time"
)

type Song struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Title     string    `gorm:"size:255;not null;index" json:"title"`
	Artist    string    `gorm:"size:255;not null;index" json:"artist"`
	YtID      string    `gorm:"size:255;uniqueIndex" json:"yt_id"`
	Key       string    `gorm:"size:255;uniqueIndex;not null" json:"key"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

type Fingerprint struct {
	Address      uint32 `gorm:"primaryKey;autoIncrement:false" json:"address"`
	AnchorTimeMs uint32 `gorm:"primaryKey;autoIncrement:false" json:"anchor_time_ms"`
	SongID       uint32 `gorm:"primaryKey;autoIncrement:false;index" json:"song_id"`
}

type Couple struct {
	AnchorTimeMs uint32 `json:"anchor_time_ms"`
	SongID       uint32 `json:"song_id"`
}
