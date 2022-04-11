package main

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
)

const (
	TS_PACKET_SIZE              = 188
	SYNC_BYTE                   = 0x47
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

var (
	VIDEO_PID uint32 = 200
	// errors
	ErrPesStartCode = errors.New("PES wrong start code prefix (must be 0x000001)")
	ErrSyncByte     = errors.New("No Sync byte (0x47)")
)

type Demuxer struct {
	r io.Reader
	b *RingBuffer
}

func NewDemuxer(rd io.Reader, buf *RingBuffer) *Demuxer {
	return &Demuxer{
		r: rd,
		b: buf,
	}
}

type TSPacket struct {
	tsHeader []byte
	PUSI     bool
	PID      uint32
	CC       uint8
	payload  []byte
}

func (tsp *TSPacket) ParsePes() error {
	p := tsp.payload
	if !(p[0] == 0x00 && p[1] == 0x00 && p[2] == 0x01) {
		log.Printf("error; payload: %x\n", p)
		return ErrPesStartCode
	}

	streamId := p[3]
	pesPacketLength := binary.BigEndian.Uint16(p[4:6])
	pesHeaderFlags := binary.BigEndian.Uint16(p[6:8])
	pesHeaderLength := int(p[8])

	log.Printf("streamId: %x\tpesHeaderFlags: %x\t pesHeaderLength: %d\tpesPacketLength: %d\n",
		streamId, pesHeaderFlags, pesHeaderLength, pesPacketLength)

	return nil
}

func NewTSPacket(d *[]byte) {
	payloadOffset := 4
	// mask is 0x30 actually but we dont care if payload empty
	adaptationFieldControl := (binary.BigEndian.Uint32(*d) & 0x20) != 0
	if adaptationFieldControl {
		AFLen := int((*d)[4])
		// we need to increment because of pusi pointer 8bit field
		payloadOffset += AFLen + 1
	}
	var p TSPacket
	p.payload = (*d)[payloadOffset:]
	p.tsHeader = (*d)[0:4]
	p.PUSI = (binary.BigEndian.Uint32(p.tsHeader) & 0x400000) != 0
	p.PID = (binary.BigEndian.Uint32(p.tsHeader) & 0x1fff00) >> 8
	p.CC = uint8(binary.BigEndian.Uint32(p.tsHeader) & 0xf)

	if p.tsHeader[0] != SYNC_BYTE {
		log.Println("SyncByte Error")
		//return -1, ErrSyncByte
	}
	/* cc error counter
		if PID == VIDEO_PID {
			log.Printf("CC: %d\n", CC)
	        }
	*/
	if p.PUSI && p.PID == 200 {
		//log.Printf("PID: %4d Header: %x (HLen: %d) PUSI: %v\n", p.PID, p.tsHeader, len(p.tsHeader), p.PUSI)
		//fmt.Println(payload)
		p.ParsePes()
	}
}

func (d *Demuxer) Parse() (i int, err error) {
	/*
	 https://en.wikipedia.org/wiki/Packetized_elementary_stream
	 https://en.wikipedia.org/wiki/MPEG_transport_stream
	 https://download.tek.com/document/2AW_14974_4-Poster.pdf
	*/
	log.Println("Demuxing")
	buf := make([]byte, TS_PACKET_SIZE)
	analyze_buf := make([]byte, TS_PACKET_SIZE)
	// total mpegts packets
	var m runtime.MemStats
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
		d.b.Write(buf)
		d.b.Read(analyze_buf)
		NewTSPacket(&analyze_buf)
		i++
		if i%100000 == 0 {
			runtime.ReadMemStats(&m)
			fmt.Printf("Alloc = %v MiB\tTotalAlloc = %2v MiB\tSys = %v MiB\tNumGC = %v\n", m.Alloc/1024/1024, m.TotalAlloc/1024/1024, m.Sys/1024/1024, m.NumGC)
		}
	}
	//return
}

func main() {
	n := os.Args[1]
	f, err := os.Open(n)

	if err != nil {
		log.Println(err.Error())
		return
	}
	defer f.Close()

	b, err := NewRingBuffer(2 * 1024 * 1024) // 2MB approx
	if b != nil {
		log.Printf("Buffer %vK size created\n", b.size/1024)
	}

	demux := NewDemuxer(bufio.NewReaderSize(f, TS_PACKET_SIZE*1024), b)

	if cs, err := demux.Parse(); err != nil {
		log.Printf("Total MPEGTS packets: %d\n", cs)
		log.Println(err.Error())
	}
	log.Printf("[buffer] Total bytes written: %d\tWrite positsion: %d\tRead position: %d\n", b.t, b.w, b.r)

}
