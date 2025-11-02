package main

import (
	fileformat "shazoom/fileformat"
	"fmt"
)

func main() {
	filepath := "/Users/prayushgiri/Downloads/Tum Se Teri Baaton Mein Aisa Uljha Jiya 320 Kbps.mp3"

	file, err := fileformat.ReformatWav(filepath, 2)
	if err != nil {
		fmt.Errorf("could not convert to WAV")
		return
	}

	

}
