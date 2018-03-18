package extensions

// PeerExchange is a BitTorrent Extension indicating we support PEX
const PeerExchange = Extension("i2p_pex")

type PEX struct {
	Added   string `bencode:"added"`
	Dropped string `bencode:"dropped"`
}
