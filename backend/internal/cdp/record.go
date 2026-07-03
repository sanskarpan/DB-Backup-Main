// Package cdp implements journal-based Continuous Data Protection (CDP) and
// Change Data Capture (CDC).
//
// The core primitive is an append-only change Journal that assigns every
// change a monotonically increasing, gap-free log sequence number (LSN) and
// records a caller-supplied timestamp. Changes are layered over a base
// snapshot, so replaying the ordered set of records up to a chosen LSN or
// timestamp reconstructs the exact state of the data at that point. This
// enables fine-grained point-in-time recovery (PITR) down to a single change.
//
// Everything in this package performs real, fsync-durable file I/O; there is
// no simulation. Timestamps are supplied by the caller (ChangeRecord.Time)
// rather than read from the wall clock so that recovery is deterministic and
// testable.
package cdp

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"time"
)

// Operation identifies the kind of mutation captured by a ChangeRecord.
type Operation uint8

const (
	// OpInsert records the creation of a new row for a key.
	OpInsert Operation = iota + 1
	// OpUpdate records a replacement of the row stored under a key.
	OpUpdate
	// OpDelete records the removal of the row stored under a key.
	OpDelete
)

// String returns the human-readable name of the operation.
func (o Operation) String() string {
	switch o {
	case OpInsert:
		return "insert"
	case OpUpdate:
		return "update"
	case OpDelete:
		return "delete"
	default:
		return "unknown"
	}
}

// valid reports whether the operation is one of the known kinds.
func (o Operation) valid() bool {
	return o == OpInsert || o == OpUpdate || o == OpDelete
}

// ChangeRecord is a single captured change to a logical table.
//
// Key uniquely identifies the affected row within its Table. After holds the
// serialized post-image of the row for Insert and Update operations and is
// ignored for Delete. Time is the caller-supplied event timestamp used for
// timestamp-based recovery; it is never derived from the wall clock inside
// this package. LSN is assigned by the Journal on Append and is zero until the
// record has been durably written.
type ChangeRecord struct {
	Table string
	Op    Operation
	Key   string
	After []byte
	Time  time.Time
	LSN   uint64
}

// errCorruptRecord is returned when a record frame fails its integrity check.
var errCorruptRecord = errors.New("cdp: corrupt journal record")

// errTornRecord signals a partially written trailing record, which is treated
// as a clean end-of-journal rather than corruption.
var errTornRecord = errors.New("cdp: torn trailing record")

// recordMagic prefixes every encoded payload to guard against accidental
// framing on foreign data.
const recordMagic uint32 = 0x43445052 // "CDPR".

// encodeRecord serializes a record into a self-describing, CRC-protected frame.
//
// The frame layout is: a uint32 payload length, the payload bytes, and a
// trailing uint32 CRC32 (IEEE) of the payload. The payload itself begins with
// recordMagic so that a decoder can reject foreign or misaligned data.
func encodeRecord(r ChangeRecord) []byte {
	var payload bytes.Buffer
	writeU32(&payload, recordMagic)
	writeU64(&payload, r.LSN)
	writeI64(&payload, r.Time.UnixNano())
	payload.WriteByte(byte(r.Op))
	writeBytes(&payload, []byte(r.Table))
	writeBytes(&payload, []byte(r.Key))
	writeBytes(&payload, r.After)

	body := payload.Bytes()
	crc := crc32.ChecksumIEEE(body)

	var frame bytes.Buffer
	writeU32(&frame, uint32(len(body)))
	frame.Write(body)
	writeU32(&frame, crc)
	return frame.Bytes()
}

// decodeRecord parses a single frame from data and returns the record together
// with the number of bytes consumed. It returns errTornRecord when data does
// not yet contain a complete frame (a partially flushed trailing write) and
// errCorruptRecord when the frame is complete but fails validation.
func decodeRecord(data []byte) (ChangeRecord, int, error) {
	const u32 = 4
	if len(data) < u32 {
		return ChangeRecord{}, 0, errTornRecord
	}
	bodyLen := int(binary.BigEndian.Uint32(data[:u32]))
	frameLen := u32 + bodyLen + u32
	if len(data) < frameLen {
		return ChangeRecord{}, 0, errTornRecord
	}
	body := data[u32 : u32+bodyLen]
	wantCRC := binary.BigEndian.Uint32(data[u32+bodyLen : frameLen])
	if crc32.ChecksumIEEE(body) != wantCRC {
		return ChangeRecord{}, 0, errCorruptRecord
	}

	rec, err := decodeBody(body)
	if err != nil {
		return ChangeRecord{}, 0, err
	}
	return rec, frameLen, nil
}

// decodeBody parses the payload portion of a validated frame.
func decodeBody(body []byte) (ChangeRecord, error) {
	rd := &reader{buf: body}
	magic, err := rd.u32()
	if err != nil || magic != recordMagic {
		return ChangeRecord{}, errCorruptRecord
	}
	lsn, err := rd.u64()
	if err != nil {
		return ChangeRecord{}, errCorruptRecord
	}
	nanos, err := rd.i64()
	if err != nil {
		return ChangeRecord{}, errCorruptRecord
	}
	opByte, err := rd.byte()
	if err != nil {
		return ChangeRecord{}, errCorruptRecord
	}
	table, err := rd.bytes()
	if err != nil {
		return ChangeRecord{}, errCorruptRecord
	}
	key, err := rd.bytes()
	if err != nil {
		return ChangeRecord{}, errCorruptRecord
	}
	after, err := rd.bytes()
	if err != nil {
		return ChangeRecord{}, errCorruptRecord
	}
	op := Operation(opByte)
	if !op.valid() {
		return ChangeRecord{}, fmt.Errorf("%w: invalid op %d", errCorruptRecord, opByte)
	}
	return ChangeRecord{
		Table: string(table),
		Op:    op,
		Key:   string(key),
		After: after,
		Time:  time.Unix(0, nanos).UTC(),
		LSN:   lsn,
	}, nil
}

// writeU32 appends a big-endian uint32 to buf.
func writeU32(buf *bytes.Buffer, v uint32) {
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], v)
	buf.Write(b[:])
}

// writeU64 appends a big-endian uint64 to buf.
func writeU64(buf *bytes.Buffer, v uint64) {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], v)
	buf.Write(b[:])
}

// writeI64 appends a big-endian int64 to buf.
func writeI64(buf *bytes.Buffer, v int64) {
	writeU64(buf, uint64(v))
}

// writeBytes appends a uint32 length prefix followed by the raw bytes.
func writeBytes(buf *bytes.Buffer, v []byte) {
	writeU32(buf, uint32(len(v)))
	buf.Write(v)
}

// reader is a minimal big-endian cursor over a payload buffer.
type reader struct {
	buf []byte
	pos int
}

// errShortBuffer is returned by reader helpers when the payload ends early.
var errShortBuffer = errors.New("cdp: short payload buffer")

func (r *reader) u32() (uint32, error) {
	if r.pos+4 > len(r.buf) {
		return 0, errShortBuffer
	}
	v := binary.BigEndian.Uint32(r.buf[r.pos : r.pos+4])
	r.pos += 4
	return v, nil
}

func (r *reader) u64() (uint64, error) {
	if r.pos+8 > len(r.buf) {
		return 0, errShortBuffer
	}
	v := binary.BigEndian.Uint64(r.buf[r.pos : r.pos+8])
	r.pos += 8
	return v, nil
}

func (r *reader) i64() (int64, error) {
	v, err := r.u64()
	if err != nil {
		return 0, err
	}
	return int64(v), nil
}

func (r *reader) byte() (byte, error) {
	if r.pos+1 > len(r.buf) {
		return 0, errShortBuffer
	}
	v := r.buf[r.pos]
	r.pos++
	return v, nil
}

func (r *reader) bytes() ([]byte, error) {
	n, err := r.u32()
	if err != nil {
		return nil, err
	}
	if r.pos+int(n) > len(r.buf) {
		return nil, errShortBuffer
	}
	out := make([]byte, n)
	copy(out, r.buf[r.pos:r.pos+int(n)])
	r.pos += int(n)
	return out, nil
}
