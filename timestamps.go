package main

func ExtractPCR(bytes []byte) uint64 {
	var a, b, c, d, e, f uint64
	a = uint64(bytes[0])
	b = uint64(bytes[1])
	c = uint64(bytes[2])
	d = uint64(bytes[3])
	e = uint64(bytes[4])
	f = uint64(bytes[5])
	pcrBase := (a << 25) | (b << 17) | (c << 9) | (d << 1) | (e >> 7)
	pcrExt := ((e & 0x1) << 8) | f
	return pcrBase*300 + pcrExt
}

func ExtractTime(bytes []byte) uint64 {
	var a, b, c, d, e uint64
	a = uint64((bytes[0] >> 1) & 0x07)
	b = uint64(bytes[1])
	c = uint64((bytes[2] >> 1) & 0x7f)
	d = uint64(bytes[3])
	e = uint64((bytes[4] >> 1) & 0x7f)
	return (a << 30) | (b << 22) | (c << 15) | (d << 7) | e
}
