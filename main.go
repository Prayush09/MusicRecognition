package main

import (
	fileformat "shazoom/fileformat"
)

func main() {
	filepath := "/Users/prayushgiri/Downloads/Tum Se Teri Baaton Mein Aisa Uljha Jiya 320 Kbps.mp3"

	fileformat.GetMetadata(filepath)
}
