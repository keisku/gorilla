package gorilla

import (
	"fmt"
	"io"
	"math"
)

const (
	firstDeltaBits = 14
)

// Compressor compresses time-series data based on Facebook's paper.
// Link to the paper: https://www.vldb.org/pvldb/vol8/p1816-teller.pdf
type Compressor struct {
	bw            *bitWriter
	header        int32
	t             int32
	tDelta        int32
	leadingZeros  uint8
	trailingZeros uint8
	value         uint64
}

// NewCompressor initialize Compressor and returns a function to be invoked
// at the end of compressing.
func NewCompressor(w io.Writer, header uint32) (c *Compressor, finish func() error, err error) {
	c = &Compressor{
		header:       int32(header),
		bw:           newBitWriter(w),
		leadingZeros: math.MaxUint8,
	}
	if err := c.bw.writeBits(uint64(header), 32); err != nil {
		return nil, nil, fmt.Errorf("failed to write header: %w", err)
	}
	return c, c.finish, nil
}

// Compress compresses time-series data and write.
func (c *Compressor) Compress(t uint32, v float64) error {
	// First time to compress.
	if c.t == 0 {
		if int32(t)-c.header < 0 {
			// Prevent overflowing of uint64(delta).
			// TODO: Consider the better way to handle the case that
			// `t` is smaller than `c.header`.
			t = uint32(c.header)
		}
		delta := int32(t) - c.header
		c.t = int32(t)
		c.tDelta = delta
		c.value = math.Float64bits(v)

		if err := c.bw.writeBits(uint64(delta), firstDeltaBits); err != nil {
			return fmt.Errorf("failed to write first timestamp: %w", err)
		}
		// The first value is stored with no compression.
		if err := c.bw.writeBits(c.value, 64); err != nil {
			return fmt.Errorf("failed to write first value: %w", err)
		}
		return nil
	}
	return c.compress(t, v)
}

func (c *Compressor) compress(t uint32, v float64) error {
	if err := c.compressTimestamp(t); err != nil {
		return fmt.Errorf("failed to compress timestamp: %w", err)
	}
	if err := c.compressValue(v); err != nil {
		return fmt.Errorf("failed to compress value: %w", err)
	}
	return nil
}

func (c *Compressor) compressTimestamp(t uint32) error {
	delta := int32(t) - c.t
	dod := int64(delta) - int64(c.tDelta) // delta of delta
	c.t = int32(t)
	c.tDelta = delta

	// | DoD         | Header bits | Value bits | Total bits |
	// |-------------|-------------|------------|------------|
	// | 0           | 0           | 0          | 1          |
	// | -63, 64     | 10          | 7          | 9          |
	// | -255, 256   | 110         | 9          | 12         |
	// | -2047, 2048 | 1110        | 12         | 16         |
	// | > 2048      | 1111        | 32         | 36         |
	switch {
	case dod == 0:
		if err := c.bw.writeBit(zero); err != nil {
			return fmt.Errorf("failed to write timestamp zero: %w", err)
		}

	case -63 <= dod && dod <= 64:
		// 0x02 == '10'
		if err := c.bw.writeBits(0x02, 2); err != nil {
			return fmt.Errorf("failed to write 2 bits header: %w", err)
		}
		if err := writeInt64Bits(c.bw, dod, 7); err != nil {
			return fmt.Errorf("failed to write 7 bits dod: %w", err)
		}

	case -255 <= dod && dod <= 256:
		// 0x06 == '110'
		if err := c.bw.writeBits(0x06, 3); err != nil {
			return fmt.Errorf("failed to write 3 bits header: %w", err)
		}
		if err := writeInt64Bits(c.bw, dod, 9); err != nil {
			return fmt.Errorf("failed to write 9 bits dod: %w", err)
		}

	case -2047 <= dod && dod <= 2048:
		// 0x0E == '1110'
		if err := c.bw.writeBits(0x0E, 4); err != nil {
			return fmt.Errorf("failed to write 4 bits header: %w", err)
		}
		if err := writeInt64Bits(c.bw, dod, 12); err != nil {
			return fmt.Errorf("failed to write 12 bits dod: %w", err)
		}

	default:
		// 0x0F == '1111'
		if err := c.bw.writeBits(0x0F, 4); err != nil {
			return fmt.Errorf("failed to write 4 bits header: %w", err)
		}
		if err := writeInt64Bits(c.bw, dod, 32); err != nil {
			return fmt.Errorf("failed to write 32 bits dod: %w", err)
		}
	}

	return nil
}

func writeInt64Bits(bw *bitWriter, i int64, nbits uint) error {
	var u uint64
	if i >= 0 || nbits >= 64 {
		u = uint64(i)
	} else {
		u = uint64(1<<nbits + i)
	}
	return bw.writeBits(u, int(nbits))
}

func (c *Compressor) compressValue(v float64) error {
	value := math.Float64bits(v)
	xor := c.value ^ value
	c.value = value

	// Value is the same as previous.
	if xor == 0 {
		return c.bw.writeBit(zero)
	}

	leadingZeros := leardingZeros(xor)
	trailingZeros := trailingZeros(xor)

	if err := c.bw.writeBit(one); err != nil {
		return fmt.Errorf("failed to write one bit: %w", err)
	}

	// If the block of meaningful bits falls within the block of previous meaningful bits,
	// i.c., there are at least as many leading zeros and as many trailing zeros as with the previous value
	// use that information for the block position and just store the meaningful XORed valuc.
	if c.leadingZeros <= leadingZeros && c.trailingZeros <= trailingZeros {
		if err := c.bw.writeBit(zero); err != nil {
			return fmt.Errorf("failed to write zero bit: %w", err)
		}
		significantBits := int(64 - c.leadingZeros - c.trailingZeros)
		if err := c.bw.writeBits(xor>>c.trailingZeros, significantBits); err != nil {
			return fmt.Errorf("failed to write xor value: %w", err)
		}
		return nil
	}

	c.leadingZeros = leadingZeros
	c.trailingZeros = trailingZeros

	if err := c.bw.writeBit(one); err != nil {
		return fmt.Errorf("failed to write one bit: %w", err)
	}
	if err := c.bw.writeBits(uint64(leadingZeros), 5); err != nil {
		return fmt.Errorf("failed to write leading zeros: %w", err)
	}
	// Note that if leading == trailing == 0, then sigbits == 64.
	// But that value doesn't actually fit into the 6 bits we havc.
	// Luckily, we never need to encode 0 significant bits,
	// since that would put us in the other case (vDelta == 0).
	// So instead we write out a 0 and adjust it back to 64 on unpacking.
	significantBits := 64 - leadingZeros - trailingZeros
	if err := c.bw.writeBits(uint64(significantBits), 6); err != nil {
		return fmt.Errorf("failed to write significant bits: %w", err)
	}
	if err := c.bw.writeBits(xor>>c.trailingZeros, int(significantBits)); err != nil {
		return fmt.Errorf("failed to write xor value")
	}
	return nil
}

func leardingZeros(v uint64) uint8 {
	var mask uint64 = 0x8000000000000000
	var ret uint8 = 0
	for ; ret < 64 && v&mask == 0; ret++ {
		mask >>= 1
	}
	return ret
}

func trailingZeros(v uint64) uint8 {
	var mask uint64 = 0x0000000000000001
	var ret uint8 = 0
	for ; ret < 64 && v&mask == 0; ret++ {
		mask <<= 1
	}
	return ret
}

// finish compresses the finish marker and flush bits with zero bits padding for byte-align.
func (c *Compressor) finish() error {
	if c.t == 0 {
		// Add finish marker with delta = 0x3FFF (firstDeltaBits = 14 bits), and first value = 0
		err := c.bw.writeBits(1<<firstDeltaBits-1, firstDeltaBits)
		if err != nil {
			return err
		}
		err = c.bw.writeBits(0, 64)
		if err != nil {
			return err
		}
		return c.bw.flush(zero)
	}

	// Add finish marker with deltaOfDelta = 0xFFFFFFFF, and value xor = 0
	err := c.bw.writeBits(0x0F, 4)
	if err != nil {
		return err
	}
	err = c.bw.writeBits(0xFFFFFFFF, 32)
	if err != nil {
		return err
	}
	err = c.bw.writeBit(zero)
	if err != nil {
		return err
	}
	return c.bw.flush(zero)
}
