package db

import (
	"fmt"
	"shazoom/models"
	"shazoom/utils"
)

type DBClient interface {
	Close() error
	StoreFingerprints(fingerprints map[uint32]models.Couple) error
	GetCouples(addresses []uint32) (map[uint32][]models.Couple, error)
	TotalSongs() (int, error)
	RegisterSong(songTitle, songArtist, ytID string) (uint32, error)
	GetSong(filterKey string, value interface{}) (Song, bool, error)
	GetSongByID(songID uint32) (Song, bool, error)
	GetSongByYTID(ytID string) (Song, bool, error)
	GetSongByKey(key string) (Song, bool, error)
	DeleteSongByID(songID uint32) error
	DeleteCollection(collectionName string) error
}

type Song struct {
	Title     string
	Artist    string
	YouTubeID string
}

func NewDBClient() (DBClient, error) {
	var (
			dbUser = utils.GetEnv("DB_USER", "postgres")
			dbPass = utils.GetEnv("DB_PASS", "")
			dbHost = utils.GetEnv("DB_HOST", "34.100.143.210") 
			dbPort = utils.GetEnv("DB_PORT", "5432")
			dbName = utils.GetEnv("DB_NAME", "postgres")
		)
        
        dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=require", 
            dbUser, dbPass, dbHost, dbPort, dbName)

		return NewPostgresClient(dsn)
}