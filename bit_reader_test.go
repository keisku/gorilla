package gorilla

import (
	"bytes"
	"testing"

	fuzz "github.com/google/gofuzz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_bitReader_readBit(t *testing.T) {
	tests := []struct {
		name       string
		byteToRead byte
		want       bit
		wantErr    error
	}{
		{
			name:       "read 00000001",
			byteToRead: 0x1,
			want:       zero,
			wantErr:    nil,
		},
		{
			name:       "read 11111111",
			byteToRead: 0xff,
			want:       one,
			wantErr:    nil,
		},
		{
			name:       "read 11000011",
			byteToRead: 0xc3,
			want:       one,
			wantErr:    nil,
		},
		{
			name:       "read 10000000",
			byteToRead: 0x80,
			want:       one,
			wantErr:    nil,
		},
		{
			name:       "read 01111111",
			byteToRead: 0x7f,
			want:       zero,
			wantErr:    nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			err := buf.WriteByte(tt.byteToRead)
			require.Nil(t, err)
			b := newBitReader(buf)
			got, err := b.readBit()
			assert.Equal(t, tt.wantErr, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_bitReader_readBits(t *testing.T) {
	tests := []struct {
		name       string
		nbits      int
		byteToRead byte
		want       uint64
		wantErr    error
	}{
		{
			name:       "read a bit from 00000001",
			nbits:      1,
			byteToRead: 0x1,
			want:       0,
			wantErr:    nil,
		},
		{
			name:       "read 5 bits from 00000001",
			nbits:      5,
			byteToRead: 0x1,
			want:       0,
			wantErr:    nil,
		},
		{
			name:       "read 8 bits from 00000001",
			nbits:      8,
			byteToRead: 0x1,
			want:       0x1,
			wantErr:    nil,
		},
		{
			name:       "read a bit from 11111111",
			nbits:      1,
			byteToRead: 0xff,
			want:       0x1,
			wantErr:    nil,
		},
		{
			name:       "read 5 bits from 11111111",
			nbits:      5,
			byteToRead: 0xff,
			want:       0x1f,
			wantErr:    nil,
		},
		{
			name:       "read 8 bits from 11111111",
			nbits:      8,
			byteToRead: 0xff,
			want:       0xff,
			wantErr:    nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			err := buf.WriteByte(tt.byteToRead)
			require.Nil(t, err)
			b := newBitReader(buf)
			got, err := b.readBits(tt.nbits)
			assert.Equal(t, tt.wantErr, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_bitReader_readByte(t *testing.T) {
	f := fuzz.New().NilChance(0)
	for i := 0; i < 100; i++ {
		var b byte
		f.Fuzz(&b)
		buf := new(bytes.Buffer)
		require.Nil(t, buf.WriteByte(b))
		br := newBitReader(buf)
		got, err := br.readByte()
		require.Nil(t, err)
		assert.Equal(t, b, got)
	}
}
