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

// cached downloading piece
type cachedPiece struct {
	pending    *bittorrent.Bitfield
	obtained   *bittorrent.Bitfield
	lastActive time.Time
	index      uint32
	length     uint32
	mtx        sync.Mutex
}

// is this piece done downloading ?
func (p *cachedPiece) done() bool {
	return p.obtained.Completed()
}

// mark slice of data at offset as obtained
func (p *cachedPiece) put(offset uint32, l uint32) {
	// set obtained
	idx := offset / BlockSize
	if l != BlockSize {
		// last block of last piece
		idx++
	}
	p.obtained.Set(idx)
	p.pending.Unset(idx)

	p.lastActive = time.Now()
}

// cancel a slice
func (p *cachedPiece) cancel(offset, length uint32) {
	idx := offset / BlockSize
	p.pending.Unset(idx)
	p.lastActive = time.Now()
}

func (p *cachedPiece) nextRequest() (r *common.PieceRequest) {
	p.mtx.Lock()
	defer p.mtx.Unlock()
	l := p.length
	r = new(common.PieceRequest)
	r.Index = p.index
	r.Length = BlockSize
	idx := uint32(0)
	for r.Begin < l {
		if p.pending.Has(idx) || p.obtained.Has(idx) {
			r.Begin += BlockSize
			idx++
		} else {
			break
		}
	}

	if r.Begin+r.Length > l {
		// is this probably the last piece ?
		if (r.Begin+r.Length)-l >= BlockSize {
			// no, let's just say there are no more blocks left
			log.Debugf("no next piece request for idx=%d", r.Index)
			r = nil
			return
		} else {
			// yes so let's correct the size
			r.Length = l - r.Begin
		}
	}
	log.Debugf("next piece request made: idx=%d offset=%d len=%d total=%d", r.Index, r.Begin, r.Length, l)
	p.pending.Set(idx)
	return
}

// picks the next good piece to download
type PiecePicker func(*bittorrent.Bitfield, []uint32) (uint32, bool)

type pieceTracker struct {
	mtx       sync.Mutex
	requests  map[uint32]*cachedPiece
	pending   int
	st        storage.Torrent
	have      func(uint32)
	nextPiece PiecePicker
}

func (pt *pieceTracker) visitCached(idx uint32, v func(*cachedPiece)) {
	pt.mtx.Lock()
	_, has := pt.requests[idx]
	if !has {
		if !pt.newPiece(idx) {
			pt.mtx.Unlock()
			return
		}
	}
	pc := pt.requests[idx]
	pt.mtx.Unlock()
	v(pc)
}

func createPieceTracker(st storage.Torrent, picker PiecePicker) (pt *pieceTracker) {
	pt = &pieceTracker{
		requests:  make(map[uint32]*cachedPiece),
		st:        st,
		nextPiece: picker,
	}
	return
}

func (pt *pieceTracker) newPiece(piece uint32) bool {

	info := pt.st.MetaInfo()

	sz := info.LengthOfPiece(piece)
	bits := sz / BlockSize
	log.Debugf("new piece idx=%d len=%d bits=%d", piece, sz, bits)
	pt.requests[piece] = &cachedPiece{
		pending:    bittorrent.NewBitfield(bits, nil),
		obtained:   bittorrent.NewBitfield(bits, nil),
		length:     sz,
		index:      piece,
		lastActive: time.Now(),
	}
	return true
}

func (pt *pieceTracker) removePiece(piece uint32) {
	pt.mtx.Lock()
	delete(pt.requests, piece)
	pt.mtx.Unlock()
}

func (pt *pieceTracker) pendingPiece(remote *bittorrent.Bitfield) (idx uint32, old bool) {
	pt.mtx.Lock()
	for k := range pt.requests {
		if remote.Has(k) {
			idx = k
			old = true
			break
		}
	}
	pt.mtx.Unlock()
	return
}

func (pt *pieceTracker) iterCached(v func(*cachedPiece)) {
	pieces := []uint32{}
	pt.mtx.Lock()
	for idx := range pt.requests {
		pieces = append(pieces, idx)
	}
	pt.mtx.Unlock()
	for _, idx := range pieces {
		pt.visitCached(idx, v)
	}
}

func (cp *cachedPiece) isExpired() (expired bool) {
	now := time.Now()
	expired = now.Sub(cp.lastActive) > time.Second*30
	return
}

func (pt *pieceTracker) nextRequestForDownload(remote *bittorrent.Bitfield, req *common.PieceRequest) bool {
	var r *common.PieceRequest
	idx, old := pt.pendingPiece(remote)
	if old {
		pt.visitCached(idx, func(cp *cachedPiece) {
			r = cp.nextRequest()
		})
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
			pt.visitCached(idx, func(cp *cachedPiece) {
				r = cp.nextRequest()
			})
		}
	}
	if r != nil && r.Length > 0 {
		req.Copy(r)
	} else {
		return false
	}
	return true
}

// cancel previously requested piece request
func (pt *pieceTracker) canceledRequest(r common.PieceRequest) {
	if r.Length == 0 {
		return
	}
	pt.visitCached(r.Index, func(pc *cachedPiece) {
		pc.cancel(r.Begin, r.Length)
	})
}

func (pt *pieceTracker) handlePieceData(d common.PieceData) {
	idx := d.Index
	pt.visitCached(idx, func(pc *cachedPiece) {
		begin := d.Begin
		l := uint32(len(d.Data))
		err := pt.st.PutChunk(idx, begin, d.Data)
		if err == nil {
			pc.put(begin, l)
		} else {
			log.Errorf("failed to put chunk %d: %s", idx, err.Error())
		}
		if pc.done() {
			err = pt.st.VerifyPiece(idx)
			if err == nil {
				pt.st.Flush()
				if pt.have != nil {
					pt.have(d.Index)
				}
			} else {
				log.Warnf("put piece %d failed: %s", idx, err.Error())
			}
			pt.removePiece(idx)
		}
	})
}
