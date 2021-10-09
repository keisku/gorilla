package gorilla_test

import (
	"bytes"
	"math/rand"
	"testing"
	"time"

	fuzz "github.com/google/gofuzz"
	"github.com/kei6u/gorilla"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Compress_Decompress(t *testing.T) {
	type Data struct {
		t uint32
		v float64
	}
	header := uint32(time.Now().Unix())

	const dataLen = 50000
	valueFuzz := fuzz.New().NilChance(0)
	data := make([]*Data, dataLen)
	ts := header
	for i := 0; i < dataLen; i++ {
		ts += uint32(rand.Int31n(10000))
		var v float64
		valueFuzz.Fuzz(&v)
		data[i] = &Data{ts, v}
	}

	buf := new(bytes.Buffer)

	// Compression
	c, finish, err := gorilla.NewCompressor(buf, header)
	require.Nil(t, err)
	for _, d := range data {
		require.Nil(t, c.Compress(d.t, d.v))
	}
	require.Nil(t, finish())

	// Decompression
	var actual []*Data
	d, h, err := gorilla.NewDecompressor(buf)
	require.Nil(t, err)
	assert.Equal(t, header, h)
	iter := d.Iter()
	for iter.Next() {
		t, v := iter.Get()
		actual = append(actual, &Data{t, v})
	}
	require.Nil(t, iter.Err())
	assert.Equal(t, data, actual)
}
