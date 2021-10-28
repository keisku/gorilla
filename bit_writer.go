package gorilla

import (
	"fmt"
	"io"
)

type bitWriter struct {
	w      io.Writer
	buffer byte
	count  uint8 // How many right-most bits are available for writing in the current byte (the last byte of the buffer).
}

// newBitWriter returns a writer that buffers bits and write the resulting bytes to 'w'
func newBitWriter(w io.Writer) *bitWriter {
	return &bitWriter{
		w: w, count: 8,
	}
}

// writeBit writes a single bit.
func (b *bitWriter) writeBit(bit bit) error {
	if bit {
		b.buffer |= 1 << (b.count - 1)
	}

	b.count--

	if b.count == 0 {
		if _, err := b.w.Write([]byte{b.buffer}); err != nil {
			return fmt.Errorf("failed to write a bit: %w", err)
		}
		b.buffer = 0
		b.count = 8
	}

	return nil
}

// writeBits writes the nbits right-most bits of u64 to the buffer in left-to-right order.
func (b *bitWriter) writeBits(u64 uint64, nbits int) error {
	u64 <<= (64 - uint(nbits))
	for nbits >= 8 {
		byt := byte(u64 >> 56)
		err := b.writeByte(byt)
		if err != nil {
			return err
		}
		u64 <<= 8
		nbits -= 8
	}

	for nbits > 0 {
		err := b.writeBit((u64 >> 63) == 1)
		if err != nil {
			return err
		}
		u64 <<= 1
		nbits--
	}

	return nil
}

// writeByte writes a single byte to the stream, regardless of alignment
func (b *bitWriter) writeByte(byt byte) error {
	// Complete the last byte with the leftmost b.buffer bits from byt.
	b.buffer |= byt >> (8 - b.count)

	if _, err := b.w.Write([]byte{b.buffer}); err != nil {
		return fmt.Errorf("failed to write a byte: %w", err)
	}
	b.buffer = byt << b.count

	return nil
}

// flush empties the currently in-process byte by filling it with 'bit'.
func (b *bitWriter) flush(bit bit) error {
	for b.count != 8 {
		err := b.writeBit(bit)
		if err != nil {
			return err
		}
	}

	return nil
}
