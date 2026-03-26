package main

import "encoding/binary"

// File format magic headers — minimal valid headers for each type.
// These make downloaded files recognizable by file type detection tools.

// pngHeader returns the minimal PNG file signature + IHDR chunk for a 1x1 pixel.
func pngHeader() []byte {
	return []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
		0x00, 0x00, 0x00, 0x0D, // IHDR length
		0x49, 0x48, 0x44, 0x52, // "IHDR"
		0x00, 0x00, 0x00, 0x01, // width: 1
		0x00, 0x00, 0x00, 0x01, // height: 1
		0x08, 0x02, // 8-bit RGB
		0x00, 0x00, 0x00, // compression, filter, interlace
	}
}

// jpegHeader returns the minimal JPEG SOI + APP0 marker.
func jpegHeader() []byte {
	return []byte{
		0xFF, 0xD8, 0xFF, 0xE0, // SOI + APP0 marker
		0x00, 0x10, // Length
		0x4A, 0x46, 0x49, 0x46, 0x00, // "JFIF\0"
		0x01, 0x01, // Version
		0x00,       // Aspect ratio units
		0x00, 0x01, // X density
		0x00, 0x01, // Y density
		0x00, 0x00, // Thumbnail
	}
}

// gifHeader returns the GIF89a signature.
func gifHeader() []byte {
	return []byte{
		0x47, 0x49, 0x46, 0x38, 0x39, 0x61, // "GIF89a"
		0x01, 0x00, 0x01, 0x00, // 1x1
		0x80, 0x00, 0x00, // GCT flag
	}
}

// webpHeader returns the RIFF/WEBP signature.
func webpHeader() []byte {
	return []byte{
		0x52, 0x49, 0x46, 0x46, // "RIFF"
		0x00, 0x00, 0x00, 0x00, // File size (placeholder)
		0x57, 0x45, 0x42, 0x50, // "WEBP"
	}
}

// bmpHeader returns a minimal BMP file header.
func bmpHeader() []byte {
	return []byte{
		0x42, 0x4D, // "BM"
		0x00, 0x00, 0x00, 0x00, // File size (placeholder)
		0x00, 0x00, 0x00, 0x00, // Reserved
		0x36, 0x00, 0x00, 0x00, // Pixel data offset
	}
}

// mp4Header returns the ftyp box for an MP4 file.
func mp4Header() []byte {
	return []byte{
		0x00, 0x00, 0x00, 0x18, // Box size: 24
		0x66, 0x74, 0x79, 0x70, // "ftyp"
		0x69, 0x73, 0x6F, 0x6D, // "isom"
		0x00, 0x00, 0x02, 0x00, // Minor version
		0x69, 0x73, 0x6F, 0x6D, // "isom"
		0x69, 0x73, 0x6F, 0x32, // "iso2"
	}
}

// webmHeader returns a minimal EBML/WebM header.
func webmHeader() []byte {
	return []byte{
		0x1A, 0x45, 0xDF, 0xA3, // EBML header
		0x93,                   // Size
		0x42, 0x86, 0x81, 0x01, // EBML version
		0x42, 0xF7, 0x81, 0x01, // EBML read version
		0x42, 0x82, 0x84, // DocType
		0x77, 0x65, 0x62, 0x6D, // "webm"
	}
}

// mkvHeader returns a minimal EBML/Matroska header.
func mkvHeader() []byte {
	return []byte{
		0x1A, 0x45, 0xDF, 0xA3, // EBML header
		0x01, 0x00, 0x00, 0x00, // Size (placeholder)
		0x00, 0x00, 0x00, 0x1F,
		0x42, 0x86, 0x81, 0x01, // EBML version
	}
}

// mp3Header returns an ID3v2 tag header + MPEG frame sync.
func mp3Header() []byte {
	return []byte{
		0x49, 0x44, 0x33, // "ID3"
		0x03, 0x00, // Version 2.3
		0x00,                   // Flags
		0x00, 0x00, 0x00, 0x00, // Size
		0xFF, 0xFB, 0x90, 0x00, // MPEG1 Layer3 frame sync
	}
}

// oggHeader returns the OGG page header magic.
func oggHeader() []byte {
	return []byte{
		0x4F, 0x67, 0x67, 0x53, // "OggS"
		0x00,                                           // Version
		0x02,                                           // Header type (BOS)
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Granule position
	}
}

// wavHeader returns a valid RIFF/WAVE header for the given file size.
func wavHeader(totalSize int64) []byte {
	h := make([]byte, 44)
	copy(h[0:4], []byte("RIFF"))
	binary.LittleEndian.PutUint32(h[4:8], uint32(totalSize-8))
	copy(h[8:12], []byte("WAVE"))
	copy(h[12:16], []byte("fmt "))
	binary.LittleEndian.PutUint32(h[16:20], 16)     // Chunk size
	binary.LittleEndian.PutUint16(h[20:22], 1)      // PCM
	binary.LittleEndian.PutUint16(h[22:24], 2)      // Stereo
	binary.LittleEndian.PutUint32(h[24:28], 44100)  // Sample rate
	binary.LittleEndian.PutUint32(h[28:32], 176400) // Byte rate
	binary.LittleEndian.PutUint16(h[32:34], 4)      // Block align
	binary.LittleEndian.PutUint16(h[34:36], 16)     // Bits per sample
	copy(h[36:40], []byte("data"))
	binary.LittleEndian.PutUint32(h[40:44], uint32(totalSize-44))
	return h
}

// flacHeader returns the FLAC stream marker.
func flacHeader() []byte {
	return []byte{
		0x66, 0x4C, 0x61, 0x43, // "fLaC"
		0x80,             // Last metadata block flag + STREAMINFO type
		0x00, 0x00, 0x22, // Length: 34
	}
}

// pdfHeader returns a minimal PDF file header.
func pdfHeader() []byte {
	return []byte("%PDF-1.4\n%\xe2\xe3\xcf\xd3\n")
}

// zipHeader returns the local file header for a ZIP archive.
func zipHeader() []byte {
	return []byte{
		0x50, 0x4B, 0x03, 0x04, // Local file header signature
		0x14, 0x00, // Version needed
		0x00, 0x00, // Flags
		0x00, 0x00, // Compression (stored)
		0x00, 0x00, 0x00, 0x00, // Mod time/date
	}
}

// tarHeader returns a minimal TAR file header (512 bytes, mostly zeros).
func tarHeader() []byte {
	h := make([]byte, 512)
	copy(h[0:], "test.dat")  // Filename
	copy(h[100:], "0000644") // Mode
	copy(h[257:], "ustar")   // Magic
	return h[:64]            // Truncated — just enough for magic detection
}

// gzipHeader returns the GZIP magic number header.
func gzipHeader() []byte {
	return []byte{
		0x1F, 0x8B, // Magic
		0x08,                   // Compression: deflate
		0x00,                   // Flags
		0x00, 0x00, 0x00, 0x00, // Modification time
		0x00, // Extra flags
		0xFF, // OS: unknown
	}
}

// sevenZipHeader returns the 7-Zip signature.
func sevenZipHeader() []byte {
	return []byte{
		0x37, 0x7A, 0xBC, 0xAF, 0x27, 0x1C, // 7z signature
	}
}
