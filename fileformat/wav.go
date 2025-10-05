package fileformat

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"shazoom/models"
	"shazoom/utils"
	"strings"
	"time"

	"github.com/mdobak/go-xerrors"
)

type WavHeader struct {
	ChunkID       [4]byte
	ChunkSize     uint32
	Format        [4]byte
	Subchunk1ID   [4]byte
	Subchunk1Size uint32
	AudioFormat   uint16
	NumChannels   uint16
	SampleRate    uint32
	BytesPerSec   uint32
	BlockAlign    uint16
	BitsPerSample uint16
	Subchunk2ID   [4]byte
	Subchunk2Size uint32
}

func writeWavHeader(file *os.File, data []byte, sampleRate, channels, bitsPerSample int) error {
	if len(data)%channels != 0 {
		return fmt.Errorf("invalid data or invalid no of channels")
	}

	subHeaderChunkSize := uint16(16)
	bytesPerSample := bitsPerSample / 8
	blockAlign := uint16(bytesPerSample * channels)
	subDataChunk := uint16(len(data))

	header := WavHeader{
		ChunkID:       [4]byte{'R', 'I', 'F', 'F'}, //flag to say this is a RIFF file — read the next chunk sizes and types accordingly.
		ChunkSize:     uint32(36 + len(data)),      //size of header + data
		Format:        [4]byte{'W', 'A', 'V', 'E'}, //flag for format of file type
		Subchunk1ID:   [4]byte{'F', 'M', 'T', ' '}, //flag for meta data format
		Subchunk1Size: uint32(subHeaderChunkSize),
		AudioFormat:   uint16(1), //PCM Format
		NumChannels:   uint16(channels),
		SampleRate:    uint32(sampleRate),
		BytesPerSec:   uint32(channels * sampleRate * bytesPerSample), //streaming speed
		BlockAlign:    blockAlign,
		BitsPerSample: uint16(bitsPerSample),
		Subchunk2ID:   [4]byte{'D', 'A', 'T', 'A'}, //flag for data
		Subchunk2Size: uint32(subDataChunk),
	}

	//write header into the file
	err := binary.Write(file, binary.LittleEndian, header)
	if err != nil {
		return fmt.Errorf("cannot write header to file: %v", err)
	}

	return nil
}

func WriteWavFile(filename string, data []byte, sampleRate, channels, bitsPerSample int) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}

	defer f.Close()

	if sampleRate <= 0 || channels <= 0 || bitsPerSample <= 0 {
		return fmt.Errorf(
			"values must be greater than zero (sampleRate: %d, channels: %d, bitsPerSample: %d)",
			sampleRate, channels, bitsPerSample,
		)
	}

	//write header
	err = writeWavHeader(f, data, sampleRate, channels, bitsPerSample)
	if err != nil {
		return err
	}

	_, err = f.Write(data)
	if err != nil {
		return err
	}

	return nil
}

type WavInfo struct {
	Channels   int
	SampleRate int
	Data       []byte
	Duration   float64
}

func ReadWavInfo(filename string) (*WavInfo, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("cannot read given file: %v", err)
	}

	if len(data) < 44 {
		return nil, fmt.Errorf("data provided in wav is insufficient")
	}

	var header WavHeader
	err = binary.Read(bytes.NewReader(data[:44]), binary.LittleEndian, &header)
	if err != nil {
		return nil, err
	}

	if string(header.ChunkID[:]) != "RIFF" || string(header.Format[:]) != "WAVE" || header.AudioFormat != 1 {
		return nil, errors.New("invalid header format")
	}

	//if all checks pass => extract info
	info := &WavInfo{
		Channels:   int(header.NumChannels),
		SampleRate: int(header.SampleRate),
		Data:       data[44:],
	}

	//caluclate duration
	if header.BitsPerSample == 16 {
		info.Duration = float64(len(info.Data)) / float64(int(header.NumChannels)*2*int(header.SampleRate))
	} else {
		return nil, errors.New("unsupported bits per sample format")
	}

	return info, nil
}

// converts 16-bit PCM WAV byte data into normalized floating-point audio samples in the range [-1.0, 1.0].
func WavBytesToSample(data []byte) ([]float64, error) {
	//check for incomplete data
	if len(data)%2 != 0 {
		return nil, errors.New("incomplete data")
	}

	numSamples := len(data) / 2
	output := make([]float64, numSamples)

	for i := 0; i < len(data); i += 2 {
		// Interpret bytes as a 16-bit signed integer (little-endian)
		sample := int16(binary.LittleEndian.Uint16(data[i : i+2]))

		// Scale the sample to the range [-1, 1]
		output[i/2] = float64(sample) / 32768.0
	}

	return output, nil
}

//Extracting audio data from the incoming audio file, json structure returned by ffprobe

type FFMPEGMetaData struct {
	Streams []struct {
		Index         int               `json:"index"`
		CodecName     string            `json:"codec_name"`
		CodecLongName string            `json:"codec_long_name"`
		CodecType     string            `json:"codec_type"`
		SampleFMT     string            `json:"sample_fmt"`
		SampleRate    string            `json:"sample_rate"`
		Channels      int               `json:"channels"`
		ChannelLayout string            `json:"channel_layout"`
		BitsPerSample int               `json:"bits_per_sample"`
		Duration      string            `json:"duration"`
		BitRate       string            `json:"bit_rate"`
		Disposition   map[string]int    `json:"disposition"`
		Tags          map[string]string `json:"tags"`
	} `json:"streams"`
	Format struct {
		Streams        int               `json:"nb_streams"`
		FormFilename   string            `json:"filename"`
		NbatName       string            `json:"format_name"`
		FormatLongName string            `json:"format_long_name"`
		StartTime      string            `json:"start_time"`
		Duration       string            `json:"duration"`
		Size           string            `json:"size"`
		BitRate        string            `json:"bit_rate"`
		Tags           map[string]string `json:"tags"`
	} `json:"format"`
}

func GetMetadata(filepath string) (FFMPEGMetaData, error) {
	var metadata FFMPEGMetaData

	//running ffprobe, no warning or errors printing, in json format and to show only the format and streams of the file
	cmd := exec.Command("ffprobe", "-v", "quiet", "-print_format", "json", "-show_format", "-show_streams", filepath)

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		fmt.Println("ffprobe error:", stderr.String())
		return metadata, err
	}

	err = json.Unmarshal(out.Bytes(), &metadata)
	if err != nil {
		return metadata, err
	}

	for k, v := range metadata.Streams[0].Tags {
		metadata.Streams[0].Tags[strings.ToLower(k)] = v
	}

	for k, v := range metadata.Format.Tags {
		metadata.Format.Tags[strings.ToLower(k)] = v
	}

	return metadata, nil
}

func ProcessRecording(recData *models.RecordData, saveRecording bool) ([]float64, error) {
	audioData, err := base64.StdEncoding.DecodeString(recData.Audio)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	filename := fmt.Sprintf("%04d_%02d_%02d_%02d_%02d_%02d.wav",
		now.Second(), now.Minute(), now.Hour(),
		now.Day(), now.Month(), now.Year(),
	)
	filePath := "tmp/" + filename

	err = WriteWavFile(filePath, audioData, recData.SampleRate, recData.Channels, recData.SampleSize)
	if err != nil {
		return nil, err
	}

	reformatedWavFile, err := ReformatWav(filePath, 1)
	if err != nil {
		return nil, err
	}

	wavInfo, _ := ReadWavInfo(reformatedWavFile)
	samples, _ := WavBytesToSample(wavInfo.Data)

	if saveRecording {
		logger := utils.GetLogger()
		ctx := context.Background()

		err := utils.CreateFolder("recordings")
		if err != nil {
			err := xerrors.New(err)
			logger.ErrorContext(ctx, "Failed create folder.", slog.Any("error", err))
		}

		newFilePath := strings.Replace(reformatedWavFile, "tmp/", "recordings/", 1)
		err = os.Rename(reformatedWavFile, newFilePath)
		if err != nil {
			logger.ErrorContext(ctx, "Failed to move file.", slog.Any("error", err))
		}
	}

	utils.DeleteFile(filename)
	utils.DeleteFile(reformatedWavFile)

	return samples, nil
}
