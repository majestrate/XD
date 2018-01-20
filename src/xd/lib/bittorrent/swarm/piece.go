package swarm

import (
	"sync"
	"time"
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
	piece      *common.PieceData
	progress   []byte
	lastActive time.Time
	mtx        sync.Mutex
}

// is this piece done downloading ?
func (p *cachedPiece) done() bool {
	p.mtx.Lock()
	defer p.mtx.Unlock()
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
		copy(p.piece.Data[offset:offset+l], data)
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
	log.Debugf("cancel piece idx=%d offset=%d length=%d", p.piece.Index, offset, length)
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
	p.lastActive = time.Now()
}

func (p *cachedPiece) hasNextRequest() (has bool) {
	p.mtx.Lock()
	idx := 0
	for idx < len(p.progress) {
		if p.progress[idx] == Obtained {
			idx += BlockSize
		} else {
			has = true
			break
		}
	}
	p.mtx.Unlock()
	return
}

func (p *cachedPiece) nextRequest() (r *common.PieceRequest) {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	l := uint32(len(p.progress))
	r = new(common.PieceRequest)
	r.Index = p.piece.Index
	r.Length = BlockSize

	for r.Begin < l {
		if p.progress[r.Begin] == Missing {
			break
		}
		r.Begin += BlockSize
	}

	if r.Begin+r.Length > l {
		if (r.Begin+r.Length)-l >= BlockSize {
			log.Debugf("no next piece request for idx=%d", r.Index)
			r = nil
			return
		} else {
			r.Length = l - r.Begin
		}
	}
	log.Debugf("next piece request made: idx=%d offset=%d len=%d total=%d", r.Index, r.Begin, r.Length, l)
	p.set(r.Begin, r.Length, Pending)
	return
}

// picks the next good piece to download
type PiecePicker func(*bittorrent.Bitfield, []uint32) (uint32, bool)

type pieceTracker struct {
	mtx        sync.Mutex
	requests   map[uint32]*cachedPiece
	st         storage.Torrent
	have       func(uint32)
	nextPiece  PiecePicker
	maxPending int
}

func createPieceTracker(st storage.Torrent, picker PiecePicker) (pt *pieceTracker) {
	pt = &pieceTracker{
		requests:   make(map[uint32]*cachedPiece),
		st:         st,
		nextPiece:  picker,
		maxPending: 32,
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

	if len(pt.requests) >= pt.maxPending {
		return
	}

	info := pt.st.MetaInfo()

	sz := info.LengthOfPiece(piece)

	log.Debugf("new piece idx=%d len=%d", piece, sz)
	cp = &cachedPiece{
		progress: make([]byte, sz),
		piece: &common.PieceData{
			Data:  make([]byte, sz),
			Index: piece,
		},
		lastActive: time.Now(),
	}
	pt.requests[piece] = cp
	return
}

func (pt *pieceTracker) removePiece(piece uint32) {
	pt.mtx.Lock()
	delete(pt.requests, piece)
	pt.mtx.Unlock()
}

func (pt *pieceTracker) pendingPiece(remote *bittorrent.Bitfield) (idx uint32, old bool) {
	for k := range pt.requests {
		if remote.Has(k) {
			idx = k
			old = true
			return
		}
	}
	return
}

// cancel entire pieces that have not been fetched within a duration
func (pt *pieceTracker) cancelTimedOut(dlt time.Duration) {
	now := time.Now()
	for idx := range pt.requests {
		if now.Sub(pt.requests[idx].lastActive) > dlt {
			delete(pt.requests, idx)
		}
	}
}

func (pt *pieceTracker) nextRequestForDownload(remote *bittorrent.Bitfield) (r *common.PieceRequest) {

	pt.mtx.Lock()
	defer pt.mtx.Unlock()
	pt.cancelTimedOut(time.Second * 30)

	idx, old := pt.pendingPiece(remote)
	var cp *cachedPiece
	if old {
		cp = pt.requests[idx]
		r = cp.nextRequest()
	}
	if r == nil {
		var exclude []uint32
		for k := range pt.requests {
			exclude = append(exclude, k)
		}
		log.Debugf("get next piece excluding %d", exclude)
		var has bool
		idx, has = pt.nextPiece(remote, exclude)
		if has {
			_, has = pt.requests[idx]
			if !has {
				cp = pt.newPiece(idx)
				if cp != nil {
					r = cp.nextRequest()
				}
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
	if pc == nil {

	} else {
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
