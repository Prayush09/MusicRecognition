package core

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"math/cmplx"
	"os"
)

/*
Image showcases:

	Horizontal axis = frequency (low to high)
	Vertical axis = time (top to bottom)
	Brightness = amplitude/loudness at that frequency and time
*/
func SpectrogramToImage(spectrogram [][]complex128, outputPath string) error {
	numOfWindows := len(spectrogram)
	numFreqBins := len(spectrogram[0])

	//create image
	img := image.NewGray(image.Rect(0, 0, numFreqBins, numOfWindows))

	//finding the max mag to create different intesity points in the heat map
	maxMagnitude := 0.0
	for i := range numOfWindows {
		for j := range numFreqBins {
			magnitude := cmplx.Abs(spectrogram[i][j])
			if magnitude > maxMagnitude {
				maxMagnitude = magnitude
			}

		}
	}

	//making sure intensity lies in between 0 and 255 | (0 = black/quiet, 255 = white/loud)
	for i := range numOfWindows {
		for j := range numFreqBins {
			magnitude := cmplx.Abs(spectrogram[i][j])
			intensity := uint8(math.Floor(255 * (magnitude / maxMagnitude)))
			img.SetGray(j, i, color.Gray{Y: intensity}) //adding intensity into each specified location.
		}
	}

	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	err = png.Encode(file, img)
	if err != nil {
		return err
	}

	return nil
}
