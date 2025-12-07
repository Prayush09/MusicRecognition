package utils

import (
	"io"
	"os"
	"fmt"
	"time"
	"math/rand"
)

func GenerateUniqueID() uint32 {
	//rand.seed(SEED) deprecated :(
	rand := rand.New(rand.NewSource(time.Now().UnixNano()))
	return rand.Uint32();
}

func GetEnv(key string, fallback ...string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	if len(fallback) > 0 {
		return fallback[0]
	}

	return ""
}

func GenerateSongKey(songTitle, songArtist string) string {
	return songTitle + "___" + songArtist
}

func RenameFile(sourcePath, destinationPath string) error {
	//get the source file
	srcFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("renamefile: failed to open source path!: %v", err)
	}

	//get the destination file
	destFile, err := os.Open(destinationPath)
	if err != nil {
		return fmt.Errorf("renamefile: failed to destination path!: %v", err)
	}
	defer destFile.Close()

	//copy contents from source file into destination file
	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return fmt.Errorf("renamefile: cannot copy src into dest: %v", err)
	}

	//close source file
	err = srcFile.Close()
	if err != nil {
		return err
	}

	//remove whole source path
	err = os.Remove(sourcePath)
	if err != nil {
		return err
	}

	return nil
}

func ExtendMap[K comparable, V any](dest, src map[K]V) {
	for k, v := range src {
		dest[k] = v
	}
}