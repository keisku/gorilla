package gorilla

import (
	"errors"
	"fmt"
	"io"
	"math"
)

// Compressor decompresses time-series data based on Facebook's paper.
// Link to the paper: https://www.vldb.org/pvldb/vol8/p1816-teller.pdf
type Decompressor struct {
	br            *bitReader
	header        uint32
	t             uint32
	delta         uint32
	leadingZeros  uint8
	trailingZeros uint8
	value         uint64
}

// NewDecompressor initializes Decompressor and returns decompressed header.
func NewDecompressor(r io.Reader) (d *Decompressor, header uint32, err error) {
	d = &Decompressor{
		br: newBitReader(r),
	}
	h, err := d.br.readBits(32)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to decode header: %w", err)
	}
	d.header = uint32(h)
	return d, d.header, nil
}

// Iter returns an iterator of decompressor.
func (d *Decompressor) Iter() *DecompressorIter {
	return &DecompressorIter{0, 0, nil, d}
}

// DecompressorIter is an iterator of Decompressor.
type DecompressorIter struct {
	t   uint32
	v   float64
	err error
	d   *Decompressor
}

// Get returns decompressed time-series data.
func (d *DecompressorIter) Get() (t uint32, v float64) {
	return d.t, d.v
}

// Err returns error during decompression.
func (d *DecompressorIter) Err() error {
	if errors.Is(d.err, io.EOF) {
		return nil
	}
	return d.err
}

// Next proceeds decompressing time-series data unitil EOF.
func (d *DecompressorIter) Next() bool {
	if d.d.t == 0 {
		d.t, d.v, d.err = d.d.decompressFirst()
	} else {
		d.t, d.v, d.err = d.d.decompress()
	}
	return d.err == nil
}

func (d *Decompressor) decompressFirst() (t uint32, v float64, err error) {
	delta, err := d.br.readBits(firstDeltaBits)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to decompress delta at first: %w", err)
	}
	if delta == 1<<firstDeltaBits-1 {
		return 0, 0, io.EOF
	}

	value, err := d.br.readBits(64)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to decompress value at first: %w", err)
	}

	d.delta = uint32(delta)
	d.t = d.header + d.delta
	d.value = value

	return d.t, math.Float64frombits(d.value), nil
}

func (d *Decompressor) decompress() (t uint32, v float64, err error) {
	t, err = d.decompressTimestamp()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to decompress timestamp: %w", err)
	}

	v, err = d.decompressValue()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to decompress value: %w", err)
	}

	return t, v, nil
}

func (d *Decompressor) decompressTimestamp() (uint32, error) {
	n, err := d.dodBitsN()
	if err != nil {
		return 0, err
	}

	if n == 0 {
		d.t += d.delta
		return d.t, nil
	}

	bits, err := d.br.readBits(int(n))
	if err != nil {
		return 0, fmt.Errorf("failed to read timestamp: %w", err)
	}

	if n == 32 && bits == 0xFFFFFFFF {
		return 0, io.EOF
	}

	var dod int64 = int64(bits)
	if n != 32 && 1<<(n-1) < int64(bits) {
		dod = int64(bits - 1<<n)
	}

	d.delta += uint32(dod)
	d.t += d.delta
	return d.t, nil
}

func (d *Decompressor) decompressValue() (float64, error) {
	bit, err := d.br.readBit()
	if err != nil {
		return 0, fmt.Errorf("failed to read value: %w", err)
	}
	if bit {
		bit, err := d.br.readBit()
		if err != nil {
			return 0, fmt.Errorf("failed to read value: %w", err)
		}
		if bit {
			// New leading and trailing zeros
			leadingZeros, err := d.br.readBits(5)
			if err != nil {
				return 0, fmt.Errorf("failed to read value: %w", err)
			}
			significantBits, err := d.br.readBits(6)
			if err != nil {
				return 0, fmt.Errorf("failed to read value: %w", err)
			}
			if significantBits == 0 {
				significantBits = 64
			}

			d.leadingZeros = uint8(leadingZeros)
			d.trailingZeros = 64 - uint8(significantBits) - d.leadingZeros
		}
		valueBits, err := d.br.readBits(int(64 - d.leadingZeros - d.trailingZeros))
		if err != nil {
			return 0, fmt.Errorf("failed to read value: %w", err)
		}
		valueBits <<= uint64(d.trailingZeros)
		d.value ^= valueBits
	}

	return math.Float64frombits(d.value), nil
}

func (d *Decompressor) dodBitsN() (n uint, err error) {
	var dod byte
	for i := 0; i < 4; i++ {
		dod <<= 1
		b, err := d.br.readBit()
		if err != nil {
			return 0, err
		}
		if b {
			dod |= 1
		} else {
			break
		}
	}

	switch dod {
	case 0x00: // 0
		return 0, nil
	case 0x02: // 10
		return 7, nil
	case 0x06: // 110
		return 9, nil
	case 0x0E: // 1110
		return 12, nil
	case 0x0F: // 1111
		return 32, nil
	default:
		return 0, errors.New("invalid bit header for bit length to read")
	}
}
