package fileformat

import (
	"encoding/binary"
	"fmt"
	"os"
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



