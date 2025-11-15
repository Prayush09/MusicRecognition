package utils

import (
	"io"
	"os"
	"fmt"
)


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