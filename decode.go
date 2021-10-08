package gorilla

import (
	"errors"
	"fmt"
	"io"
	"math"
)

// Decoder is a Facebook's paper based encoder.
// Link to the paper: https://www.vldb.org/pvldb/vol8/p1816-teller.pdf
type Decoder struct {
	br            *bitReader
	header        uint32
	t             uint32
	delta         uint32
	leadingZeros  uint8
	trailingZeros uint8
	value         uint64
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		br: newBitReader(r),
	}
}

// LoadHeader loads the starting timestamp.
func (d *Decoder) LoadHeader() (header uint32, err error) {
	t, err := d.br.readBits(32)
	if err != nil {
		return 0, fmt.Errorf("failed to decode header: %w", err)
	}
	d.header = uint32(t)
	return d.header, nil
}

// Decode read and decode data, then bind it to an argument.
func (d *Decoder) Decode(data *Data) error {
	var err error
	var read *Data
	if d.t == 0 {
		read, err = d.readFirst()
		if err != nil {
			return err
		}
	} else {
		read, err = d.read()
		if err != nil {
			return err
		}
	}
	data.UnixTimestamp = read.UnixTimestamp
	data.Value = read.Value
	return nil
}

func (d *Decoder) readFirst() (*Data, error) {
	delta, err := d.br.readBits(firstDeltaBits)
	if err != nil {
		return nil, fmt.Errorf("failed to read first sample: %w", err)
	}
	if delta == 1<<firstDeltaBits-1 {
		return nil, io.EOF
	}

	value, err := d.br.readBits(64)
	if err != nil {
		return nil, fmt.Errorf("failed to read first sample value: %w", err)
	}

	d.delta = uint32(delta)
	d.t = d.header + d.delta
	d.value = value

	return &Data{
		UnixTimestamp: d.t,
		Value:         math.Float64frombits(d.value),
	}, nil
}

func (d *Decoder) read() (*Data, error) {
	t, err := d.readTimestamp()
	if err != nil {
		return nil, err
	}

	v, err := d.readValue()
	if err != nil {
		return nil, err
	}

	return &Data{t, v}, nil
}

func (d *Decoder) readTimestamp() (uint32, error) {
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

func (d *Decoder) readValue() (float64, error) {
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

// read delta of delta
func (d *Decoder) dodBitsN() (n uint, err error) {
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
