package main

import (
	"encoding/binary"
	"bufio"
	"bytes"
	"io"
	"os"
	"log"
	"errors"
)

const (
	TS_PACKET_SIZE = 188
	SYNC_BYTE = 0x47
	STREAM_TYPE_VIDEO_MPEG1 = 0x01
	STREAM_TYPE_VIDEO_MPEG2 = 0x02
	STREAM_TYPE_AUDIO_MPEG1 = 0x03
	STREAM_TYPE_AUDIO_MPEG2 = 0x04
	STREAM_TYPE_PRIVATE_SECTION = 0x05
	STREAM_TYPE_PRIVATE_DATA = 0x06
	STREAM_TYPE_AUDIO_AAC = 0x0f
	STREAM_TYPE_VIDEO_MPEG4 = 0x10
	STREAM_TYPE_VIDEO_H264 = 0x1b
	STREAM_TYPE_VIDEO_HEVC = 0x24
	STREAM_TYPE_AUDIO_AC3 = 0x81
)

var (
	VIDEO_PID uint32 = 200
	// errors
	ErrPesStartCode = errors.New("PES wrong start code prefix (must be 0x000001)")
	ErrSyncByte = errors.New("No Sync byte (0x47)")
)

type Demuxer struct {
	r io.Reader
	b *bytes.Buffer
}

func NewDemuxer(r io.Reader) *Demuxer {
	return &Demuxer{
		r: r,
		b : &bytes.Buffer{},
	}
}

type TSPacket struct {
	PUSI			bool
	tsHeader		[]byte
	PID				uint32
    payload			*[]byte
	CC				uint8
}

func (tsp *TSPacket) ParsePes() error {
	p := *tsp.payload
	if !(p[0] == 0x00 && p[1] == 0x00 && p[2] == 0x01) {
		log.Printf("error; payload: %x\n", p)
		return ErrPesStartCode
	}

	//streamId := p[3]
	//pesPacketLength := binary.BigEndian.Uint16(p[4:6])
	//pesHeaderFlags := binary.BigEndian.Uint16(p[6:8])
	//pesHeaderLength := int(p[8])

	//log.Printf("streamId:%d", streamId)
	//log.Printf("pesHeaderFlags:%x", pesHeaderFlags)
	//log.Printf("pesHeaderLength:%d", pesHeaderLength)
	//log.Printf("pesPacketLength:%d", pesPacketLength)

	return nil
}


func (p *TSPacket) Parse(d *[]byte) {
	payloadOffset := 4
	// mask is 0x30 actually but we dont care if payload empty
	adaptationFieldControl := (binary.BigEndian.Uint32(*d) & 0x20) != 0
	if adaptationFieldControl {
		AFLen := int((*d)[4])
		// we need to increment because of pusi pointer
		payloadOffset += AFLen + 1
		//log.Printf("len: %d offset: %d\n", AFLen, payloadOffset)
	}
	payload := (*d)[payloadOffset:]
	p.tsHeader = (*d)[0:4]
	p.PUSI = (binary.BigEndian.Uint32(*d) & 0x400000) != 0
	p.PID = (binary.BigEndian.Uint32(*d) & 0x1fff00) >> 8
	p.payload = &payload
	p.CC = uint8(binary.BigEndian.Uint32(*d) & 0xf)
}

func (d *Demuxer) Parse() (i int,err error) {
	/*
	 https://en.wikipedia.org/wiki/Packetized_elementary_stream 
	 https://en.wikipedia.org/wiki/MPEG_transport_stream 
	 https://download.tek.com/document/2AW_14974_4-Poster.pdf
	*/
	log.Println("Demuxing")
	buf := make([]byte, TS_PACKET_SIZE)
	// total mpegts packets
	for {
		_, err := io.ReadFull(d.r, buf)
		if err != nil {
			// need to flush buff
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				return i, err
			} else {
				return -1, err
			}
		}
		var p TSPacket
		p.Parse(&buf)
		if p.tsHeader[0] != SYNC_BYTE {
			log.Println("SyncByte Error")
			return -1, ErrSyncByte
		}
		i++
		// cc error counter
		//if p.PID == VIDEO_PID {
			//log.Printf(p.CC)
		//}
		if p.PUSI && p.PID == VIDEO_PID {
			log.Printf("PID: %4d Deader: %x (HLen: %d) PUSI: %v\n", p.PID, p.tsHeader, len(p.tsHeader), p.PUSI)
			p.ParsePes()
		}
	}
	return
}

func main() {
	n := os.Args[1]
	f, err := os.Open(n)

	if err != nil {
		log.Println(err.Error())
		return
	}
	defer f.Close()

	demux := NewDemuxer(bufio.NewReaderSize(f, TS_PACKET_SIZE*1024))

	if cs, err := demux.Parse(); err != nil {
		log.Printf("Total MPEGTS packets: %d\n", cs)
		log.Println(err.Error())
	}
}
