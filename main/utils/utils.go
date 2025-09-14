package utils

import (
	"crypto/rand"
	"encoding/binary"
	"strings"
	"time"
)

// GenerateUniqueID generates a unique ID for songs
func GenerateUniqueID() uint32 {
	var b [4]byte
	rand.Read(b[:])
	timestamp := uint32(time.Now().Unix())
	random := binary.LittleEndian.Uint32(b[:])
	return timestamp ^ random
}

// GenerateSongKey generates a unique key for a song based on title and artist
func GenerateSongKey(title, artist string) string {
	key := strings.ToLower(strings.TrimSpace(title + "-" + artist))
	// Remove special characters and normalize spaces
	key = strings.ReplaceAll(key, " ", "_")
	key = strings.ReplaceAll(key, "'", "")
	key = strings.ReplaceAll(key, "\"", "")
	key = strings.ReplaceAll(key, "&", "and")
	return key
}
