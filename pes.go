package main

import (
	"log"
)

const (
	STREAM_TYPE_VIDEO_MPEG1     = 0x01
	STREAM_TYPE_VIDEO_MPEG2     = 0x02
	STREAM_TYPE_AUDIO_MPEG1     = 0x03
	STREAM_TYPE_AUDIO_MPEG2     = 0x04
	STREAM_TYPE_PRIVATE_SECTION = 0x05
	STREAM_TYPE_PRIVATE_DATA    = 0x06
	STREAM_TYPE_AUDIO_AAC       = 0x0f
	STREAM_TYPE_VIDEO_MPEG4     = 0x10
	STREAM_TYPE_VIDEO_H264      = 0x1b
	STREAM_TYPE_VIDEO_HEVC      = 0x24
	STREAM_TYPE_AUDIO_AC3       = 0x81
)

type PesHeader struct {
	pts uint64
	dts uint64
}

func (tsp *TSPacket) ParsePes() error {
	p := tsp.Data
	phdr := new(PesHeader)
	if !(p[0] == 0x00 && p[1] == 0x00 && p[2] == 0x01) {
		log.Printf("error; payload: %x\n", p)
		return ErrPesStartCode
	}

	//streamId := p[3]
	//pesPacketLength := uint16(p[4])<<8 | uint16(p[5])
	//pesHeaderFlags := binary.BigEndian.Uint16(p[6:8])
	ptsDtsIndicator := (uint8(p[7]) & 0xc0 >> 6)

	if ptsDtsIndicator == 2 {
		phdr.pts = ExtractTime(p[9:14])
	} else if ptsDtsIndicator == 3 {
		phdr.pts = ExtractTime(p[9:14])
		phdr.dts = ExtractTime(p[14:19])
	}
	tsp.PES = *phdr
	return nil
}
