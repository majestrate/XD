package swarm

import (
	"github.com/majestrate/XD/lib/bittorrent"
	"github.com/majestrate/XD/lib/common"
	"github.com/majestrate/XD/lib/log"
	"github.com/majestrate/XD/lib/storage"
	"github.com/majestrate/XD/lib/sync"
	"time"
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

// should we accept a piece data with offset and length ?
func (p *cachedPiece) accept(offset, length uint32) bool {
	if offset%BlockSize != 0 {
		log.Errorf("Rejecting chunk where piece offset=%d %% BlockSize=%d != 0", offset, BlockSize)
		return false
	}

	if offset+length > p.length {
		log.Errorf("Rejecting chunk where piece ending offset=%d > piece length=%d", offset+length, p.length)
		return false
	}

	if p.bitfieldIndex(offset) == p.finalChunkBitfieldIndex() {
		// last piece
		if length != p.finalChunkLen() {
			log.Errorf("Rejecting final chunk of piece where length=%d != finalChunkLen=%d", length, p.finalChunkLen())
			return false
		}
	} else {
		if length != BlockSize {
			log.Errorf("Rejecting non-final chunk of piece where length=%d != BlockSize=%d", length, BlockSize)
			return false
		}
	}

	return true
}

func (p *cachedPiece) finalChunkBitfieldIndex() uint32 {
	return p.bitfieldIndex(p.length - 1)
}

func (p *cachedPiece) finalChunkLen() uint32 {
	rem := p.length % BlockSize

	if rem == 0 {
		return BlockSize
	} else {
		return rem
	}
}

// is this piece done downloading ?
func (p *cachedPiece) done() bool {
	return p.obtained.Completed()
}

// calculate bitfield index for offset
func (p *cachedPiece) bitfieldIndex(offset uint32) uint32 {
	return offset / BlockSize
}

// mark slice of data at offset as obtained
func (p *cachedPiece) put(offset uint32) {
	// set obtained
	idx := p.bitfieldIndex(offset)
	p.obtained.Set(idx)
	p.pending.Unset(idx)
	p.lastActive = time.Now()
	log.Debugf("put idx=%d offset=%d bit=%d", p.index, offset, idx)
}

// cancel a slice
func (p *cachedPiece) cancel(offset uint32) {
	idx := p.bitfieldIndex(offset)
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
	for r.Begin < l {
		idx := p.bitfieldIndex(r.Begin)
		if p.pending.Has(idx) || p.obtained.Has(idx) {
			r.Begin += BlockSize
		} else {
			if idx == p.finalChunkBitfieldIndex() {
				r.Length = p.finalChunkLen()
			}
			log.Debugf("next piece request made: idx=%d offset=%d len=%d total=%d", r.Index, r.Begin, r.Length, l)
			p.pending.Set(idx)
			return
		}
	}

	log.Debugf("no next piece request for idx=%d", r.Index)
	r = nil
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
	if sz%BlockSize != 0 {
		bits++
	}
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
	expired = now.Sub(cp.lastActive) > time.Minute*5
	return
}

func (pt *pieceTracker) PendingPieces() (exclude []uint32) {
	pt.mtx.Lock()
	for k, cp := range pt.requests {
		if cp.pending.CountSet() > 0 {
			exclude = append(exclude, k)
		}
	}
	pt.mtx.Unlock()
	return
}

func (pt *pieceTracker) NextRequest(remote *bittorrent.Bitfield, lastReq *common.PieceRequest) (r *common.PieceRequest) {
	if lastReq != nil {
		pt.visitCached(lastReq.Index, func(cp *cachedPiece) {
			r = cp.nextRequest()
		})
	}
	if r != nil {
		return
	}
	// no last request or no more requests for last request
	// pick new piece
	exclude := pt.PendingPieces()
	idx, has := pt.nextPiece(remote, exclude)
	if !has {
		// no next piece
		return
	}
	// get next requset for this newly created piece
	pt.visitCached(idx, func(cp *cachedPiece) {
		r = cp.nextRequest()
	})
	return
}

// cancel previously requested piece request
func (pt *pieceTracker) canceledRequest(r *common.PieceRequest) {
	if r.Length == 0 {
		return
	}
	pt.visitCached(r.Index, func(pc *cachedPiece) {
		pc.cancel(r.Begin)
	})
}

func (pt *pieceTracker) handlePieceData(d *common.PieceData) {
	idx := d.Index
	pt.visitCached(idx, func(pc *cachedPiece) {
		if !pc.accept(d.Begin, uint32(len(d.Data))) {
			log.Errorf("invalid piece data: index=%d offset=%d length=%d", d.Index, d.Begin, len(d.Data))
			return
		}
		err := pt.st.PutChunk(d)
		if err == nil {
			pc.put(d.Begin)
		} else {
			pc.cancel(d.Begin)
			log.Errorf("failed to put chunk %d: %s", idx, err.Error())
		}
		if pc.done() {
			err = pt.st.VerifyPiece(idx)
			if err == nil {
				pt.st.Flush()
				if pt.have != nil {
					pt.have(idx)
				}
			} else {
				log.Warnf("put piece %d failed: %s", idx, err.Error())
			}
			pt.removePiece(idx)
		}
	})
}
