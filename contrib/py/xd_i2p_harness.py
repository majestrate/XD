#!/usr/bin/env python3
"""Clean-room harness that talks to an XD peer over I2P.

The SAM client follows the published SAM v3 bridge protocol
(https://geti2p.net/en/docs/api/samv3). The BitTorrent exchange follows
BEP 3 (handshake and length-prefixed messages), BEP 10 (extension
handshake, reserved bit 20, message id 20), and BEP 9 (ut_metadata
blocks of 16384 bytes). Nothing here is derived from XD's source.

Usage:
  python3 xd_i2p_harness.py self-test
  python3 xd_i2p_harness.py sam
  python3 xd_i2p_harness.py metadata --dest DEST --infohash HEX
  python3 xd_i2p_harness.py metadata --dest DEST --torrent FILE

DEST may be a base64 destination, a .b32.i2p address, or another name
the SAM bridge can look up. The metadata command connects, exchanges
handshakes, requests every ut_metadata piece, and checks that each
reply is one BEP 9 block whose total_size matches the advertised
metadata_size. The assembled info-dict must hash to the infohash.
"""

from __future__ import annotations

import argparse
import base64
import hashlib
import os
import socket
import struct
import sys
import time

METADATA_BLOCK = 16 * 1024
BT_PSTR = b"BitTorrent protocol"
EXT_MESSAGE = 20
EXT_HANDSHAKE = 0
EXT_BIT_MASK = 0x10
MSG_NAMES = {
    0: "choke",
    1: "unchoke",
    2: "interested",
    3: "not interested",
    4: "have",
    5: "bitfield",
    6: "request",
    7: "piece",
    8: "cancel",
    9: "port",
    20: "extended",
}


class HarnessError(Exception):
    pass


class CheckFailure(Exception):
    pass


def bdecode_at(buf, index):
    if index >= len(buf):
        raise HarnessError("truncated bencode")
    kind = buf[index]
    if kind == ord("i"):
        end = buf.index(b"e", index)
        return int(buf[index + 1 : end]), end + 1
    if kind == ord("l"):
        index += 1
        items = []
        while buf[index] != ord("e"):
            item, index = bdecode_at(buf, index)
            items.append(item)
        return items, index + 1
    if kind == ord("d"):
        index += 1
        mapping = []
        while buf[index] != ord("e"):
            key, index = bdecode_at(buf, index)
            if not isinstance(key, bytes):
                raise HarnessError("bencode dict key is not a string")
            value_at = index
            value, index = bdecode_at(buf, index)
            mapping.append((key, value, buf[value_at:index]))
        return mapping, index + 1
    if ord("0") <= kind <= ord("9"):
        colon = buf.index(b":", index)
        size = int(buf[index:colon])
        start = colon + 1
        end = start + size
        return buf[start:end], end
    raise HarnessError("bad bencode at offset %d" % index)


def bencode(value):
    if isinstance(value, int) and not isinstance(value, bool):
        return b"i%de" % value
    if isinstance(value, bytes):
        return b"%d:%s" % (len(value), value)
    if isinstance(value, str):
        raw = value.encode("utf-8")
        return b"%d:%s" % (len(raw), raw)
    if isinstance(value, list):
        return b"l" + b"".join(bencode(item) for item in value) + b"e"
    if isinstance(value, dict):
        parts = []
        for key in sorted(value):
            raw_key = key if isinstance(key, bytes) else str(key).encode("utf-8")
            parts.append(bencode(raw_key))
            parts.append(bencode(value[key]))
        return b"d" + b"".join(parts) + b"e"
    raise HarnessError("cannot bencode %r" % (type(value),))


def dict_get(mapping, key):
    raw_key = key if isinstance(key, bytes) else key.encode("utf-8")
    for item_key, value, _raw in mapping:
        if item_key == raw_key:
            return value
    return None


def info_dict_bytes(torrent):
    top, consumed = bdecode_at(torrent, 0)
    if consumed != len(torrent) or not isinstance(top, list):
        raise HarnessError("torrent file is not one bencode dict")
    for key, _value, raw in top:
        if key == b"info":
            return raw
    raise HarnessError("torrent file has no info dict")


class SockIO:
    def __init__(self, sock):
        self.sock = sock
        self.buf = b""

    def readline(self):
        while b"\n" not in self.buf:
            chunk = self.sock.recv(4096)
            if not chunk:
                raise HarnessError("connection closed while reading a line")
            self.buf += chunk
        line, self.buf = self.buf.split(b"\n", 1)
        return line.decode("latin-1")

    def readexactly(self, size):
        while len(self.buf) < size:
            chunk = self.sock.recv(size - len(self.buf))
            if not chunk:
                raise HarnessError("connection closed after %d/%d bytes" % (len(self.buf), size))
            self.buf += chunk
        data = self.buf[:size]
        self.buf = self.buf[size:]
        return data

    def sendall(self, data):
        self.sock.sendall(data)


def sam_tokens(line):
    found = {}
    for token in line.split():
        if "=" in token:
            key, value = token.split("=", 1)
            found[key] = value
    return found


def connect_tcp(host, port, timeout):
    sock = socket.create_connection((host, port), timeout)
    sock.settimeout(timeout)
    return sock


class SamSession:
    """One SAM v3 STREAM session. The control socket stays open."""

    def __init__(self, host, port, timeout):
        self.host = host
        self.port = port
        self.timeout = timeout
        self.session_id = "xdharness" + os.urandom(4).hex()
        self.control = None
        self.io = None

    def open(self):
        sock = connect_tcp(self.host, self.port, self.timeout)
        self.control = sock
        self.io = SockIO(sock)
        hello = self._hello(self.io)
        create = (
            "SESSION CREATE STYLE=STREAM ID=%s DESTINATION=TRANSIENT "
            "SIGNATURE_TYPE=7 i2cp.leaseSetEncType=4 "
            "inbound.quantity=1 outbound.quantity=1 "
            "inbound.length=1 outbound.length=1\n" % self.session_id
        )
        self.io.sendall(create.encode("ascii"))
        line = self.io.readline()
        fields = sam_tokens(line)
        if fields.get("RESULT") != "OK":
            raise HarnessError("SESSION CREATE failed: %s" % line.strip())
        return hello, line.strip()

    def lookup(self, name):
        self.io.sendall(("NAMING LOOKUP NAME=%s\n" % name).encode("ascii"))
        line = self.io.readline()
        fields = sam_tokens(line)
        if fields.get("RESULT") != "OK" or "VALUE" not in fields:
            raise HarnessError("NAMING LOOKUP %s failed: %s" % (name, line.strip()))
        return fields["VALUE"], line.strip()

    def stream(self, destination):
        sock = connect_tcp(self.host, self.port, self.timeout)
        io = SockIO(sock)
        try:
            self._hello(io)
            command = "STREAM CONNECT ID=%s DESTINATION=%s SILENT=false\n" % (
                self.session_id,
                destination,
            )
            io.sendall(command.encode("ascii"))
            line = io.readline()
            fields = sam_tokens(line)
            if fields.get("RESULT") != "OK":
                raise HarnessError("STREAM CONNECT failed: %s" % line.strip())
            return sock, io, line.strip()
        except Exception:
            sock.close()
            raise

    def close(self):
        if self.control is not None:
            self.control.close()
            self.control = None

    @staticmethod
    def _hello(io):
        io.sendall(b"HELLO VERSION MIN=3.0 MAX=3.1\n")
        line = io.readline()
        fields = sam_tokens(line)
        if fields.get("RESULT") != "OK":
            raise HarnessError("SAM HELLO failed: %s" % line.strip())
        return line.strip()


def i2p_b64decode(text):
    # I2P base64 uses "-" for "+" and "~" for "/", and often omits padding.
    standard = text.replace("-", "+").replace("~", "/")
    standard += "=" * ((4 - len(standard) % 4) % 4)
    return base64.b64decode(standard)


def redact_destination(line):
    parts = []
    for token in line.split():
        if token.startswith("DESTINATION="):
            parts.append("DESTINATION=<private-key>")
        else:
            parts.append(token)
    return " ".join(parts)


def public_b32(destination_b64):
    digest = hashlib.sha256(i2p_b64decode(destination_b64)).digest()
    text = base64.b32encode(digest).decode("ascii").lower().rstrip("=")
    return text + ".b32.i2p"


def resolve_destination(session, dest):
    if "." in dest or len(dest) < 500:
        value, line = session.lookup(dest)
        return value, line
    return dest, "given base64 destination"


def bt_handshake(infohash, peer_id):
    reserved = bytearray(8)
    reserved[5] = EXT_BIT_MASK
    return bytes([len(BT_PSTR)]) + BT_PSTR + bytes(reserved) + infohash + peer_id


def parse_bt_handshake(data):
    if len(data) != 68 or data[0] != len(BT_PSTR) or data[1:20] != BT_PSTR:
        raise HarnessError("peer did not send a BitTorrent handshake")
    reserved = data[20:28]
    return {
        "extension": bool(reserved[5] & EXT_BIT_MASK),
        "infohash": data[28:48],
        "peer_id": data[48:68],
    }


def wire_message(message_id, payload=b""):
    return struct.pack(">IB", len(payload) + 1, message_id) + payload


def read_wire_message(io, limit):
    header = io.readexactly(4)
    length = struct.unpack(">I", header)[0]
    if length == 0:
        print("[info] wire keepalive")
        return None, b""
    print("[info] wire length %d" % length)
    if length > limit:
        raise CheckFailure("peer sent a %d byte message, over the %d byte cap" % (length, limit))
    body = io.readexactly(length)
    return body[0], body[1:]


def extended_message(ext_id, payload):
    return wire_message(EXT_MESSAGE, bytes([ext_id]) + payload)


def our_extended_handshake():
    payload = bencode({b"m": {b"ut_metadata": 1}, b"v": "xd-i2p-harness"})
    return extended_message(EXT_HANDSHAKE, payload)


def metadata_request(ext_id, piece):
    payload = bencode({b"msg_type": 0, b"piece": piece})
    return extended_message(ext_id, payload)


def parse_metadata_message(payload):
    mapping, consumed = bdecode_at(payload, 0)
    if not isinstance(mapping, list):
        raise HarnessError("ut_metadata payload is not a dict")
    msg_type = dict_get(mapping, "msg_type")
    piece = dict_get(mapping, "piece")
    total = dict_get(mapping, "total_size")
    return msg_type, piece, total, payload[consumed:]


def expected_block_len(total, piece):
    start = piece * METADATA_BLOCK
    if total <= 0 or start >= total:
        return None
    remain = total - start
    if remain > METADATA_BLOCK:
        return METADATA_BLOCK
    return remain


def piece_count(info):
    mapping, _consumed = bdecode_at(info, 0)
    pieces = dict_get(mapping, "pieces")
    if not isinstance(pieces, bytes) or len(pieces) == 0 or len(pieces) % 20:
        raise HarnessError("info dict has no piece hashes")
    return len(pieces) // 20


def bitfield_message(npieces):
    # BEP 3: after the handshake, send a bitfield. Zero means we have no file pieces yet.
    nbytes = (npieces + 7) // 8 if npieces else 1
    return wire_message(5, b"\x00" * nbytes)


def block_count(total):
    if total <= 0:
        return 0
    return (total + METADATA_BLOCK - 1) // METADATA_BLOCK


def make_peer_id():
    body = b"-PY0001-" + os.urandom(12)
    return body[:20]


def check(ok, message):
    print("[%s] %s" % ("pass" if ok else "fail", message))
    if not ok:
        raise CheckFailure(message)


def run_self_test():
    request = bencode({b"msg_type": 0, b"piece": 0})
    check(request == b"d8:msg_typei0e5:piecei0ee", "request dict encodes in sorted key order")
    data_dict = bencode({b"msg_type": 1, b"piece": 0, b"total_size": 1540})
    check(
        data_dict == b"d8:msg_typei1e5:piecei0e10:total_sizei1540ee",
        "data dict carries the full metadata length as total_size",
    )
    piece = b"abc"
    msg_type, index, total, rest = parse_metadata_message(data_dict + piece)
    check(msg_type == 1 and index == 0 and total == 1540 and rest == piece, "data trailer is the piece bytes")

    info = b"d3:fooi1ee"
    torrent = b"d4:info" + info + b"e"
    check(info_dict_bytes(torrent) == info, "info-dict bytes are taken without re-encoding")

    for total, piece_index, want in (
        (1540, 0, 1540),
        (1540, 1, None),
        (28114, 0, METADATA_BLOCK),
        (28114, 1, 28114 - METADATA_BLOCK),
        (154488, 0, METADATA_BLOCK),
        (154488, 9, 154488 - 9 * METADATA_BLOCK),
        (154488, 10, None),
        (METADATA_BLOCK, 0, METADATA_BLOCK),
        (METADATA_BLOCK, 1, None),
    ):
        got = expected_block_len(total, piece_index)
        check(got == want, "block length total=%d piece=%d -> %s" % (total, piece_index, got))

    handshake = bt_handshake(b"\x11" * 20, b"-PY0001-" + b"x" * 12)
    parsed = parse_bt_handshake(handshake)
    check(len(handshake) == 68 and parsed["extension"], "handshake is 68 bytes and sets the BEP 10 bit")
    sample = base64.b64encode(b"\xfb\xff").decode("ascii").replace("+", "-").replace("/", "~").rstrip("=")
    check(i2p_b64decode(sample) == b"\xfb\xff", "I2P base64 uses - and ~")
    print("[pass] self-test")


def run_sam(args):
    session = SamSession(args.host, args.sam_port, args.timeout)
    try:
        hello, status = session.open()
        print("[pass] %s" % hello)
        print("[pass] %s" % redact_destination(status))
        _destination, line = session.lookup("ME")
        print("[pass] %s" % line)
        raw = i2p_b64decode(_destination)
        print("[pass] public destination is %d bytes" % len(raw))
        print("b32 %s" % public_b32(_destination))
    finally:
        session.close()


def read_extended_handshake(io, limit, deadline):
    while time.monotonic() < deadline:
        message_id, payload = read_wire_message(io, limit)
        if message_id is None:
            continue
        name = MSG_NAMES.get(message_id, "id-%d" % message_id)
        if message_id != EXT_MESSAGE:
            print("[info] peer sent %s (%d bytes)" % (name, len(payload)))
            continue
        if not payload:
            raise HarnessError("empty extended message")
        ext_id, body = payload[0], payload[1:]
        if ext_id != EXT_HANDSHAKE:
            print("[info] peer sent extension id %d before its handshake" % ext_id)
            continue
        mapping, _consumed = bdecode_at(body, 0)
        extensions = dict_get(mapping, "m") or []
        metadata_id = None
        if isinstance(extensions, list):
            metadata_id = dict_get(extensions, "ut_metadata")
        metadata_size = dict_get(mapping, "metadata_size")
        version = dict_get(mapping, "v")
        version_text = version.decode("utf-8", "replace") if isinstance(version, bytes) else ""
        return metadata_id, metadata_size, version_text
    raise HarnessError("timed out waiting for the extended handshake")


def run_metadata(args):
    if args.torrent:
        raw = open(args.torrent, "rb").read()
        info = info_dict_bytes(raw)
        infohash = hashlib.sha1(info).digest()
    else:
        infohash = bytes.fromhex(args.infohash)
    if len(infohash) != 20:
        raise HarnessError("infohash must be 20 bytes")

    session = SamSession(args.host, args.sam_port, args.timeout)
    stream = None
    try:
        hello, _status = session.open()
        print("[pass] %s" % hello)
        destination, looked = resolve_destination(session, args.dest)
        print("[pass] destination %s" % looked)
        stream, io, status = session.stream(destination)
        print("[pass] %s" % status)

        peer_id = make_peer_id()
        io.sendall(bt_handshake(infohash, peer_id))
        remote = parse_bt_handshake(io.readexactly(68))
        shown = remote["peer_id"].decode("latin-1", "replace").replace("\x00", "")
        check(remote["infohash"] == infohash, "peer echoed infohash %s" % remote["infohash"].hex())
        check(remote["extension"], "peer set the BEP 10 extension bit (peer id %s)" % shown)
        io.sendall(our_extended_handshake())
        npieces = piece_count(info) if args.torrent else 0
        bits = bitfield_message(npieces)
        io.sendall(bits)
        print("[info] sent empty bitfield for %d pieces" % npieces)

        deadline = time.monotonic() + args.timeout
        their_id, metadata_size, version = read_extended_handshake(io, args.max_message, deadline)
        check(isinstance(their_id, int) and their_id > 0, "peer advertised ut_metadata id %s version %s" % (their_id, version))
        check(isinstance(metadata_size, int) and metadata_size > 0, "peer advertised metadata_size %s" % metadata_size)

        pieces = block_count(metadata_size)
        assembled = []
        for piece in range(pieces):
            io.sendall(metadata_request(their_id, piece))
            msg_type, got_piece, total_size, data = wait_metadata(io, their_id, args.max_message, deadline)
            want = expected_block_len(metadata_size, piece)
            check(msg_type == 1, "piece %d was data, not msg_type %s" % (piece, msg_type))
            check(got_piece == piece, "piece %d reply names piece %s" % (piece, got_piece))
            check(total_size == metadata_size, "piece %d total_size %s equals metadata_size %d" % (piece, total_size, metadata_size))
            check(len(data) == want, "piece %d length %d, expected %d" % (piece, len(data), want))
            print("[pass] piece %d/%d len=%d total_size=%d" % (piece, pieces, len(data), total_size))
            assembled.append(data)

        metadata = b"".join(assembled)
        digest = hashlib.sha1(metadata).digest()
        check(len(metadata) == metadata_size, "assembled %d bytes, metadata_size is %d" % (len(metadata), metadata_size))
        check(digest == infohash, "assembled info-dict hashes to %s" % digest.hex())
        print("[pass] fetched %d metadata bytes in %d pieces" % (len(metadata), pieces))
    finally:
        if stream is not None:
            stream.close()
        session.close()


def wait_metadata(io, their_id, limit, deadline):
    while time.monotonic() < deadline:
        message_id, payload = read_wire_message(io, limit)
        if message_id is None:
            continue
        if message_id != EXT_MESSAGE or not payload or payload[0] != their_id:
            name = MSG_NAMES.get(message_id, "id-%s" % message_id)
            print("[info] skipping %s while waiting for ut_metadata" % name)
            continue
        return parse_metadata_message(payload[1:])
    raise HarnessError("timed out waiting for a ut_metadata reply")


def parse_args(argv):
    parser = argparse.ArgumentParser(description="Clean-room SAM harness for an XD peer")
    parser.add_argument("--sam", default="127.0.0.1:7656", help="SAM bridge host:port")
    parser.add_argument("--timeout", type=float, default=180, help="seconds for each SAM and I2P step")
    parser.add_argument("--max-message", type=int, default=2 * 1024 * 1024, help="largest BitTorrent message to accept")
    sub = parser.add_subparsers(dest="command", required=True)

    sub.add_parser("self-test", help="check framing locally, without I2P")
    sub.add_parser("sam", help="open a SAM session and print this harness destination")

    metadata = sub.add_parser("metadata", help="fetch ut_metadata from a peer and check BEP 9")
    metadata.add_argument("--dest", required=True, help="peer destination, b32, or name")
    source = metadata.add_mutually_exclusive_group(required=True)
    source.add_argument("--infohash", help="40 hex character infohash")
    source.add_argument("--torrent", help="torrent file whose info-dict is hashed")
    return parser.parse_args(argv)


def main(argv):
    args = parse_args(argv)
    if args.sam.count(":") != 1:
        raise HarnessError("--sam must be host:port")
    host, port_text = args.sam.rsplit(":", 1)
    args.host = host
    args.sam_port = int(port_text)
    if args.command == "self-test":
        run_self_test()
        return 0
    if args.command == "sam":
        run_sam(args)
        return 0
    if args.command == "metadata":
        run_metadata(args)
        return 0
    raise HarnessError("unknown command")


if __name__ == "__main__":
    try:
        sys.exit(main(sys.argv[1:]))
    except CheckFailure:
        sys.exit(1)
    except (HarnessError, OSError, ValueError) as exc:
        print("[fail] %s" % exc, file=sys.stderr)
        sys.exit(1)
