package swarm

import (
	"sync"
	"xd/lib/bittorrent"
	"xd/lib/common"
	"xd/lib/log"
	"xd/lib/storage"
)

// how big should we download pieces at a time (bytes)?
const BlockSize = 1024 * 16

const Missing = 0
const Pending = 1
const Obtained = 2

// cached downloading piece
type cachedPiece struct {
	piece    *common.PieceData
	progress []byte
	mtx      sync.Mutex
}

// is this piece done downloading ?
func (p *cachedPiece) done() bool {
	for _, b := range p.progress {
		if b != Obtained {
			return false
		}
	}
	return true
}

// put a slice of data at offset
func (p *cachedPiece) put(offset uint32, data []byte) {
	p.mtx.Lock()
	l := uint32(len(data))
	if offset+l <= uint32(len(p.progress)) {
		// put data
		c := copy(p.piece.Data[offset:], data)
		if c != len(data) {
			log.Errorf("copied invalid length of slice: len=%d", c)
		}
		// put progress
		p.set(offset, l, Obtained)
	} else {
		log.Warnf("block out of range %d, %d %d", offset, len(data), len(p.progress))
	}
	p.mtx.Unlock()
}

// cancel a slice
func (p *cachedPiece) cancel(offset, length uint32) {
	p.mtx.Lock()
	p.set(offset, length, Missing)
	p.mtx.Unlock()
}

func (p *cachedPiece) set(offset, length uint32, b byte) {
	l := uint32(len(p.progress))
	if offset+length <= l {
		for length > 0 {
			length--
			p.progress[offset+length] = b
		}
	} else {
		log.Warnf("invalid cached piece range: %d %d %d", offset, length, l)
	}
}

func (p *cachedPiece) nextRequest() (r *common.PieceRequest) {
	p.mtx.Lock()
	l := uint32(len(p.progress))
	r = &common.PieceRequest{
		Index:  p.piece.Index,
		Length: BlockSize,
	}

	for r.Begin+r.Length < l {

		if p.progress[r.Begin] == Missing {

			break
		}

		r.Begin += BlockSize
	}

	// probably a last piece, round to best fit
	if r.Begin+r.Length > l {
		r.Length = l - r.Begin
	}

	p.set(r.Begin, r.Length, Pending)
	p.mtx.Unlock()
	log.Debugf("next piece request made: idx=%d offset=%d len=%d total=%d", r.Index, r.Begin, r.Length, l)
	return
}

type pieceTracker struct {
	mtx      sync.RWMutex
	requests map[uint32]*cachedPiece
	st       storage.Torrent
	have     func(uint32)
}

func createPieceTracker(st storage.Torrent) (pt *pieceTracker) {
	pt = &pieceTracker{
		requests: make(map[uint32]*cachedPiece),
		st:       st,
	}
	return
}

func (pt *pieceTracker) getPiece(piece uint32) (cp *cachedPiece) {
	pt.mtx.Lock()
	cp, _ = pt.requests[piece]
	pt.mtx.Unlock()
	return
}

func (pt *pieceTracker) newPiece(piece uint32) (cp *cachedPiece) {
	info := pt.st.MetaInfo()
	np := info.Info.NumPieces()
	pl := info.Info.PieceLength
	sz := uint64(pl)
	ts := info.TotalSize()

	if piece+1 == np {
		sz = uint64(pl) - (uint64(np)*uint64(pl) - ts)
	}

	log.Debugf("new piece total=%d idx=%d len=%d", ts, piece, sz)
	cp = &cachedPiece{
		progress: make([]byte, sz),
		piece: &common.PieceData{
			Data:  make([]byte, sz),
			Index: piece,
		},
	}
	return
}

func (pt *pieceTracker) removePiece(piece uint32) {
	pt.mtx.Lock()
	delete(pt.requests, piece)
	pt.mtx.Unlock()
}

func (pt *pieceTracker) nextRequestForDownload(remote *bittorrent.Bitfield) (r *common.PieceRequest) {
	bf := pt.st.Bitfield()
	i := pt.st.MetaInfo()
	np := i.Info.NumPieces()
	idx := np
	for idx > 0 {
		idx--
		if remote.Has(idx) && !bf.Has(idx) {
			pt.mtx.Lock()
			cp, has := pt.requests[idx]
			if !has {
				cp = pt.newPiece(idx)
				pt.requests[idx] = cp
			}
			pt.mtx.Unlock()
			r = cp.nextRequest()
			if r != nil && r.Length > 0 {
				return
			}
		}
	}
	return
}

// cancel previously requested piece request
func (pt *pieceTracker) canceledRequest(r *common.PieceRequest) {
	if r == nil {
		return
	}
	pc := pt.getPiece(r.Index)
	if pc != nil {
		pc.cancel(r.Begin, r.Length)
	}
}

func (pt *pieceTracker) handlePieceData(d *common.PieceData) {
	pc := pt.getPiece(d.Index)
	if pc != nil {
		pc.put(d.Begin, d.Data)
		if pc.done() {
			err := pt.st.PutPiece(pc.piece)
			if err == nil {
				pt.st.Flush()
				if pt.have != nil {
					pt.have(d.Index)
				}
			} else {
				log.Warnf("put piece %d failed: %s", pc.piece.Index, err)
			}
			pt.removePiece(d.Index)
		}
	}
}