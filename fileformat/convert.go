package fileformat

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"shazoom/utils"
	"strings"
)

// channels => Mono(1) or Stereo(2)
// function takes in a input audio file and returns loc of converted wav file.
func ConvertToWAV(inputFilePath string, channels int) (wavFilePath string, err error) {

	//verifying file path
	_, err = os.Stat(inputFilePath)
	if err != nil {
		return "", fmt.Errorf("input file does not exists!: %v", err)
	}	

	//checking channel, safe proofing it to 1 if it's not 1 or 2.
	if channels < 1 || channels > 2 {
		channels = 1
	}

	//renaming the extenstion of the file to .wav
	fileExt := filepath.Ext(inputFilePath)
	outputFile := strings.TrimSuffix(inputFilePath, fileExt) + ".wav"

	tempFile := filepath.Join(filepath.Dir(outputFile), "temp_"+filepath.Base(outputFile))
	defer os.Remove(tempFile) //remove temp file after use to free memory

	//using ffmpeg to take data from input file and write it to temp file (WAV)
	cmd := exec.Command(
		"ffmpeg",
		"-y",                //overwrites output file without asking
		"-i", inputFilePath, //input file path
		"-c", "pcm_s16le", //audio codec => encoding the output file using Pulse Code Modulation signed 16 bit little endian format => uncompressed raw audio codec commonly used for high-quality audio processing.
		"-ar", "44100", //sample rate
		"-ac", fmt.Sprint(channels),
		outputFile,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to convert into wav, err : %v, output: %v", err, string(output))
	}

	//copy temp file contents into output
	err = utils.RenameFile(tempFile, outputFile)
	if err != nil {
		return "", err
	}

	return outputFile, nil
}

//converts a file from any given format to wav
func ReformatWav(filePath string, channels int) (reformatedFilePath string, err error) {
	if channels < 1 || channels > 2 {
		channels = 1
	}

	//get the file from the filepath
	fileExt := filepath.Ext(filePath)
	outputFile := strings.TrimSuffix(filePath, fileExt) + "rfm.wav"

	cmd := exec.Command(
		"ffmpeg",
		"-y",
		"-i", filePath,
		"-c", "pcm_s16le",
		"-ar", "44100",
		"-ac", fmt.Sprint(channels),
		outputFile,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to convert to WAV: %v, output %v", err, string(output))
	}

	return outputFile, nil
}

type ConversionOptions struct {
	Channels int
	SampleRate int
	OutputFilePath string
	useTempFile bool
}

func convertToWAV(filePath string, opts ConversionOptions) (string, error) {
	if(opts.Channels < 1 || opts.Channels > 2){
		opts.Channels = 1;
	}
	
	//TODO: Add logic here to make sure if tempFile is required you can pass in true for that
	//TODO: Explanation, found two redundant functions, doing the same thing just one using a temp file
	// Trying to remove or merge it into one logic so that both cases are handled via one function. 

	//generate a direct output file
	outputFile := strings.TrimSuffix(filePath, filepath.Ext(filePath)) + ".rfm.wav"

	cmd := exec.Command(
		"ffmpeg",
		"-y",
		"-i", filePath,
		"-c", "pcm_s16le",
		"-ar", "44100",
		"-ac", fmt.Sprint(opts.Channels),
		outputFile,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("error occurred during converion to WAV: %v, output: %v", err, string(output))
	}

	return outputFile, nil
}
