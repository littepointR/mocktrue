package visual

// ChecksumSpec describes the checksum algorithm.
type ChecksumSpec struct {
	Type   string // "none"|"sum8"|"crc8"|"crc16"|"crc32"
	Offset int
	Width  int
	Poly   uint32
	Init   uint32
	XorOut uint32
}

// Checksum calculates checksums for various algorithms.
func Checksum(data []byte, spec ChecksumSpec) uint32 {
	switch spec.Type {
	case "sum8":
		var sum uint32
		for _, b := range data {
			sum += uint32(b)
		}
		return sum & 0xFF
	case "crc8":
		return uint32(crc8(data, uint8(spec.Poly), uint8(spec.Init), uint8(spec.XorOut)))
	case "crc16":
		return uint32(crc16(data, uint16(spec.Poly), uint16(spec.Init), uint16(spec.XorOut)))
	case "crc32":
		return crc32(data, spec.Poly, spec.Init, spec.XorOut)
	default:
		return 0
	}
}

func crc8(data []byte, poly, init, xorOut uint8) uint8 {
	crc := init
	for _, b := range data {
		crc ^= b
		for i := 0; i < 8; i++ {
			if crc&0x80 != 0 {
				crc = (crc << 1) ^ poly
			} else {
				crc <<= 1
			}
		}
	}
	return crc ^ xorOut
}

func crc16(data []byte, poly, init, xorOut uint16) uint16 {
	crc := init
	for _, b := range data {
		crc ^= uint16(b)
		for i := 0; i < 8; i++ {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ poly
			} else {
				crc <<= 1
			}
		}
	}
	return crc ^ xorOut
}

func crc32(data []byte, poly, init, xorOut uint32) uint32 {
	crc := init
	for _, b := range data {
		crc ^= uint32(b)
		for i := 0; i < 8; i++ {
			if crc&0x80000000 != 0 {
				crc = (crc << 1) ^ poly
			} else {
				crc <<= 1
			}
		}
	}
	return crc ^ xorOut
}
