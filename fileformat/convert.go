package fileformat

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	//"shazoom/utils"
	"strings"
)

// channels => Mono(1) or Stereo(2)
// function takes in a input audio file and returns loc of converted wav file.
func ConvertToWAV(inputFilePath string, channels int) (wavFilePath string, err error) {

	//verifying file path
	_, err = os.Stat(inputFilePath)
	if err != nil {
		return "", fmt.Errorf("input file does not exists!: %w", err)
	}	

	opts := ConversionOptions{
		Channels: channels,
		useTempFile: true,
	}

	return convertToWAV(inputFilePath, opts)
}

//converts a file from any given format to wav
func ReformatWav(filePath string, channels int) (string, error) {
	opts := ConversionOptions{
		Channels: channels,
		useTempFile: false,
	}
	return convertToWAV(filePath, opts)
}

type ConversionOptions struct {
	Channels int
	useTempFile bool
}

//This function works on opts, in which if useTempFile is true then we create a temp file, else we don't.
func convertToWAV(filePath string, opts ConversionOptions) (string, error) {
	if(opts.Channels < 1 || opts.Channels > 2){
		opts.Channels = 1
	}
	
	outputFile := strings.TrimSuffix(filePath, filepath.Ext(filePath)) + ".rfm.wav"
	var targetFile string

	//creating output directory before creating temp -> Found out via testing!
	outputDir := filepath.Dir(outputFile)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	if(opts.useTempFile){
		tempFile :=  filepath.Join(filepath.Dir(outputFile), "temp_"+filepath.Base(outputFile))
		targetFile = tempFile
		defer os.Remove(tempFile)		
	} else {
		targetFile = outputFile
	}

	cmd := exec.Command(
			"ffmpeg",
			"-y",
			"-i", filePath,
			"-c", "pcm_s16le",
			"-ar", "44100",
			"-ac", fmt.Sprint(opts.Channels),
			targetFile,
		)

	output, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("failed to convert into wav, err : %w, output: %v", err, string(output))
	}

	if opts.useTempFile {
		//TODO: User utils.RenameFile when front-end exists!
		err = os.Rename(targetFile, outputFile)
				if err != nil {
					return "", err
				}
	}

	return outputFile, nil
}
