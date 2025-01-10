package main

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"log"
	"os"
)

const (
	TS_PACKET_SIZE = 188
	SYNC_BYTE      = 0x47
	BUFF_SIZE      = 2 * 1024 * 1024
)

var (
	// errors
	ErrPesStartCode = errors.New("PES wrong start code prefix (must be 0x000001)")
	ErrSyncByte     = errors.New("No Sync byte (0x47)")
	ErrNotEnough    = errors.New("Not enough read")
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
	Payload                bool
	PUSI                   bool
	PID                    uint32
	CC                     uint8
	Data                   []byte
	HasPCR                 bool
	PCR                    uint64
	adaptationFieldControl bool
	PES                    PesHeader
}

type Analyzer struct {
	videoPID     uint32
	ccLast       uint8
	ccCounter    uint64
	totalPackets uint64
	tv           uint64
	last_pts     uint64
	first_pts    uint64
	duration     uint64
}

func (d *Demuxer) NextTSPacket() (*TSPacket, error) {

	payloadOffset := 4
	// p is analyze buffer
	p := make([]byte, TS_PACKET_SIZE)
	if d.b.Length() < 188 {
		//log.Println("slow read from file")
		return nil, ErrNotEnough
	}
	if _, err := d.b.Read(p); err != nil {
		return nil, err
	}
	pkt := new(TSPacket)
	// mask is 0x30 actually but we dont care if payload empty
	pkt.adaptationFieldControl = (p[3])&0x20 != 0
	if pkt.adaptationFieldControl {
		AFLen := int((p)[4])
		// we need to increment because of pusi pointer 8bit field
		payloadOffset += AFLen + 1
	}

	//log.Println((payloadOffset))
	pkt.Data = (p)[payloadOffset:]

	if p[0] != SYNC_BYTE {
		log.Printf("SyncByte Error\n")
	}

	pkt.PUSI = p[1]&0x040 != 0
	pkt.PID = uint32(p[1]&0x1f)<<8 | uint32(p[2])
	pkt.CC = uint8(binary.BigEndian.Uint32(p[0:4]) & 0xf)
	pkt.HasPCR = p[5]&0x10 != 0 && pkt.adaptationFieldControl
	if pkt.HasPCR {
		pkt.PCR = ExtractPCR(p[6:12])
	}
	// check payload
	if p[3]&0x10 != 0 {
		pkt.Payload = true
	} else {
		pkt.Payload = false
	}

	return pkt, nil
}

func (a *Analyzer) AnalyzeStream(p *TSPacket) {

	if p.PID == a.videoPID {
		a.tv++
		if p.Payload && p.CC != (a.ccLast+1)&0xf {
			a.ccCounter++
		}
		if p.HasPCR {
			//log.Println(p.PCR)
		}
		a.ccLast = p.CC
		if p.PUSI {
			p.ParsePes()
			if a.first_pts == 0 {
				a.first_pts = p.PES.pts
			}
			a.last_pts = p.PES.pts
		}
	}
	a.totalPackets++
}

func (d *Demuxer) Demux() error {
	/*
	 https://en.wikipedia.org/wiki/Packetized_elementary_stream
	 https://en.wikipedia.org/wiki/MPEG_transport_stream
	 https://download.tek.com/document/2AW_14974_4-Poster.pdf
	*/

	log.Println("Demuxing")
	a := &Analyzer{videoPID: 200, ccLast: 16}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func(ctx context.Context) uint64 {
		for {
			select {
			case <-ctx.Done():
				break
			default:
				pkt, err := d.NextTSPacket()
				if err != nil {
					continue
				}
				a.AnalyzeStream(pkt)

			}
		}
	}(ctx)
	_, err := io.Copy(d.b, d.r)
	if err != nil {
		return err
	}
	d.b.Flush()
	a.duration = ((a.last_pts - a.first_pts) + 3003) / 90000
	log.Println("Total video packets", a.tv)
	log.Printf("Total MPEGTS packets: %d\tBytes read: %v\tCC errors: %d, duration: %d sec\n", a.totalPackets, d.b.t, a.ccCounter, a.duration)
	return nil
}

func main() {
	n := os.Args[1]
	f, err := os.Open(n)

	if err != nil {
		log.Println(err.Error())
		return
	}
	defer f.Close()

	b := NewRingBuffer(5 * BUFF_SIZE) // 2MB approx
	b.SetBlocking(true)

	demux := NewDemuxer(bufio.NewReaderSize(f, TS_PACKET_SIZE*1024), b)

	if err := demux.Demux(); err != nil {
		log.Println(err.Error())
	}
}
