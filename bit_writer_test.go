package gorilla

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"

	fuzz "github.com/google/gofuzz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_bitWriter_writeBit(t *testing.T) {
	tests := []struct {
		name   string
		binary string
		hex    uint8
	}{
		{
			name:   "write 1",
			binary: "00000001",
			hex:    0x1,
		},
		{
			name:   "write 8",
			binary: "00001000",
			hex:    0x8,
		},
		{
			name:   "write 113",
			binary: "01110001",
			hex:    0x71,
		},
		{
			name:   "write 255",
			binary: "11111111",
			hex:    0xff,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			bw := newBitWriter(buf)
			for i := 0; i < len(tt.binary); i++ {
				var err error
				if tt.binary[i] == '1' {
					err = bw.writeBit(one)
				} else {
					err = bw.writeBit(zero)
				}
				require.Nil(t, err)
			}
			assert.Equal(t, tt.hex, buf.Bytes()[0])
		})
	}
}

func Test_bitWriter_writeBits(t *testing.T) {
	var (
		unix = time.Now().Unix()
	)
	type args struct {
		u     uint64
		nbits int
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "write timestamp",
			args: args{
				u:     uint64(unix),
				nbits: 64,
			},
		},
		{
			name: "write 630",
			args: args{
				u:     630,
				nbits: 64,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			bw := newBitWriter(buf)

			err := bw.writeBits(tt.args.u, tt.args.nbits)
			require.Nil(t, err)

			wantBytesLen := tt.args.nbits / 8
			wantBytes := make([]byte, wantBytesLen)
			binary.BigEndian.PutUint64(wantBytes, tt.args.u)

			assert.Equal(t, wantBytesLen, buf.Len())
			assert.Equal(t, wantBytes, buf.Bytes())
		})
	}
}

func Test_bitWriter_writeByte(t *testing.T) {
	f := fuzz.New().NilChance(0)
	for i := 0; i < 100; i++ {
		var b byte
		f.Fuzz(&b)
		buf := new(bytes.Buffer)
		bw := newBitWriter(buf)
		err := bw.writeByte(b)
		require.Nil(t, err)
		assert.Equal(t, 1, buf.Len())
		assert.Equal(t, b, buf.Bytes()[0])
	}
}
