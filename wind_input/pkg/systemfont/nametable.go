package systemfont

import (
	"encoding/binary"
	"io"
	"os"
	"unicode/utf16"
)

// readLocalizedFamilyName reads the font file's OpenType/TrueType name table
// and returns the Chinese localized font family name (nameID=1).
// Returns empty string if no Chinese name is found or on any error.
func readLocalizedFamilyName(fontPath string) string {
	f, err := os.Open(fontPath)
	if err != nil {
		return ""
	}
	defer f.Close()

	// Peek at the first 4 bytes to detect TTC vs single font
	var tag [4]byte
	if _, err := io.ReadFull(f, tag[:]); err != nil {
		return ""
	}

	if string(tag[:]) == "ttcf" {
		return readNameFromTTC(f)
	}

	// Single TTF/OTF — rewind and parse directly
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return ""
	}
	return readNameFromSFNT(f)
}

// readNameFromTTC handles TrueType Collection files.
// It reads the first font's name table only.
func readNameFromTTC(r io.ReadSeeker) string {
	// Already consumed 'ttcf' tag (4 bytes).
	// Remaining TTC header: majorVersion(2) + minorVersion(2) + numFonts(4)
	var buf [8]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return ""
	}
	numFonts := binary.BigEndian.Uint32(buf[4:8])
	if numFonts == 0 {
		return ""
	}

	// Read offset of the first font in the collection
	var offsetBuf [4]byte
	if _, err := io.ReadFull(r, offsetBuf[:]); err != nil {
		return ""
	}
	offset := binary.BigEndian.Uint32(offsetBuf[:])

	if _, err := r.Seek(int64(offset), io.SeekStart); err != nil {
		return ""
	}
	return readNameFromSFNT(r)
}

// readNameFromSFNT parses a single SFNT font (TTF or OTF) to find the 'name' table.
func readNameFromSFNT(r io.ReadSeeker) string {
	// Offset table: sfVersion(4) + numTables(2) + searchRange(2) + entrySelector(2) + rangeShift(2)
	var header [12]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return ""
	}
	numTables := binary.BigEndian.Uint16(header[4:6])

	// Scan table directory for the 'name' table
	var nameOffset, nameLength uint32
	for i := 0; i < int(numTables); i++ {
		var rec [16]byte // tag(4) + checksum(4) + offset(4) + length(4)
		if _, err := io.ReadFull(r, rec[:]); err != nil {
			return ""
		}
		if string(rec[:4]) == "name" {
			nameOffset = binary.BigEndian.Uint32(rec[8:12])
			nameLength = binary.BigEndian.Uint32(rec[12:16])
			break
		}
	}
	if nameOffset == 0 || nameLength == 0 {
		return ""
	}

	// Read the name table data
	if _, err := r.Seek(int64(nameOffset), io.SeekStart); err != nil {
		return ""
	}
	// Cap to 64 KB to guard against corrupt files
	if nameLength > 64*1024 {
		nameLength = 64 * 1024
	}
	data := make([]byte, nameLength)
	if _, err := io.ReadFull(r, data); err != nil {
		return ""
	}

	return parseChineseFamilyName(data)
}

// Windows-platform language IDs used to locate Chinese name records.
const (
	langZhCN = 0x0804 // Simplified Chinese
	langZhTW = 0x0404 // Traditional Chinese
)

// parseChineseFamilyName extracts the Chinese Font Family (nameID=1) from
// a raw name table. Prefers Simplified Chinese; falls back to Traditional.
func parseChineseFamilyName(data []byte) string {
	if len(data) < 6 {
		return ""
	}
	count := binary.BigEndian.Uint16(data[2:4])
	stringOffset := binary.BigEndian.Uint16(data[4:6])

	var zhCN, zhTW string

	for i := 0; i < int(count); i++ {
		off := 6 + i*12
		if off+12 > len(data) {
			break
		}
		platformID := binary.BigEndian.Uint16(data[off:])
		encodingID := binary.BigEndian.Uint16(data[off+2:])
		languageID := binary.BigEndian.Uint16(data[off+4:])
		nameID := binary.BigEndian.Uint16(data[off+6:])
		length := binary.BigEndian.Uint16(data[off+8:])
		strOff := binary.BigEndian.Uint16(data[off+10:])

		// Only interested in Font Family (1) on Windows platform (3), Unicode BMP encoding (1)
		if nameID != 1 || platformID != 3 || encodingID != 1 {
			continue
		}

		start := int(stringOffset) + int(strOff)
		end := start + int(length)
		if end > len(data) || length == 0 {
			continue
		}

		s := decodeUTF16BE(data[start:end])
		switch languageID {
		case langZhCN:
			return s // Best match — return immediately
		case langZhTW:
			if zhTW == "" {
				zhTW = s
			}
		}
	}

	if zhCN != "" {
		return zhCN
	}
	return zhTW
}

// decodeUTF16BE decodes a big-endian UTF-16 byte slice into a Go string.
func decodeUTF16BE(b []byte) string {
	if len(b) < 2 || len(b)%2 != 0 {
		return ""
	}
	u16s := make([]uint16, len(b)/2)
	for i := range u16s {
		u16s[i] = binary.BigEndian.Uint16(b[i*2:])
	}
	return string(utf16.Decode(u16s))
}
