package gorilla_test

import (
	"bytes"
	"math/rand"
	"testing"
	"time"

	fuzz "github.com/google/gofuzz"
	"github.com/keisku/gorilla"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Compress_Decompress(t *testing.T) {
	type data struct {
		t uint32
		v float64
	}
	header := uint32(time.Now().Unix())

	const dataLen = 50000
	expected := make([]data, dataLen)
	valueFuzz := fuzz.New().NilChance(0)
	ts := header
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < dataLen; i++ {
		if 0 < i && i%10 == 0 {
			ts -= uint32(rand.Intn(100))
		} else {
			ts += uint32(rand.Int31n(100))
		}
		var v float64
		valueFuzz.Fuzz(&v)
		expected[i] = data{ts, v}
	}

	buf := new(bytes.Buffer)

	// Compression
	c, finish, err := gorilla.NewCompressor(buf, header)
	require.Nil(t, err)
	for _, data := range expected {
		require.Nil(t, c.Compress(data.t, data.v))
	}
	require.Nil(t, finish())

	// Decompression
	var actual []data
	d, h, err := gorilla.NewDecompressor(buf)
	require.Nil(t, err)
	assert.Equal(t, header, h)
	iter := d.Iterator()
	for iter.Next() {
		t, v := iter.At()
		actual = append(actual, data{t, v})
	}
	require.Nil(t, iter.Err())
	assert.Equal(t, expected, actual)
}
