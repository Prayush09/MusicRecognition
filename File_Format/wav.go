package fileformat

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"errors"
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

	header := WavHeader {
		ChunkID: [4]byte{'R', 'I', 'F', 'F'}, //flag to say this is a RIFF file — read the next chunk sizes and types accordingly.
		ChunkSize: uint32(36 + len(data)), //size of header + data
		Format: [4]byte{'W', 'A', 'V', 'E'}, //flag for format of file type
		Subchunk1ID: [4]byte{'F', 'M', 'T', ' '}, //flag for meta data format 
		Subchunk1Size: uint32(subHeaderChunkSize),
		AudioFormat: uint16(1), //PCM Format
		NumChannels: uint16(channels),
		SampleRate: uint32(sampleRate),
		BytesPerSec: uint32(channels * sampleRate * bytesPerSample), //streaming speed
		BlockAlign: blockAlign,
		BitsPerSample: uint16(bitsPerSample),
		Subchunk2ID: [4]byte{'D', 'A', 'T', 'A'}, //flag for data
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

type WavInfo struct{
	Channels int
	SampleRate int
	Data []byte
	Duration float64
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
		Channels: int(header.NumChannels),
		SampleRate: int(header.SampleRate),
		Data: data[44:],
	}

	//caluclate duration
	if header.BitsPerSample == 16 {
		info.Duration = float64(len(info.Data)) / float64(int(header.NumChannels) * 2 * int(header.SampleRate))
	} else {
		return nil, errors.New("unsupported bits per sample format")
	}

	return info, nil
}

// converts 16-bit PCM WAV byte data into normalized floating-point audio samples in the range [-1.0, 1.0].
func WavBytesToSample(data []byte) ([]float64, error){
	//check for incomplete data
	if len(data)%2 != 0{
		return nil, errors.New("incomplete data")
	}	

	numSamples := len(data)/2
	output := make([]float64, numSamples)

	for i := 0; i < len(data); i += 2 {
		// Interpret bytes as a 16-bit signed integer (little-endian)
		sample := int16(binary.LittleEndian.Uint16(data[i : i + 2]))

		// Scale the sample to the range [-1, 1]
		output[i/2] = float64(sample) / 32768.0
	}

	return output, nil
}