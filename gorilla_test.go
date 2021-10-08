package gorilla_test

import (
	"bytes"
	"io"
	"math/rand"
	"testing"
	"time"

	fuzz "github.com/google/gofuzz"
	"github.com/kei6u/gorilla"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Encode_Decode(t *testing.T) {
	t0 := uint32(time.Now().Unix())

	const dataLen = 50000
	valueFuzz := fuzz.New().NilChance(0)
	data := make([]*gorilla.Data, dataLen)
	ts := t0
	for i := 0; i < dataLen; i++ {
		ts += uint32(rand.Int31n(10000))
		var v float64
		valueFuzz.Fuzz(&v)
		data[i] = &gorilla.Data{ts, v}
	}

	buf := new(bytes.Buffer)

	// Encode
	e := gorilla.NewEncoder(buf)
	e.PutHeader(t0)
	for _, d := range data {
		require.Nil(t, e.Encode(*d))
	}
	require.Nil(t, e.Flush())

	// Decode
	d := gorilla.NewDecoder(buf)
	h, err := d.LoadHeader()

	require.Nil(t, err)
	assert.Equal(t, t0, h)
	var inputData []*gorilla.Data
	for {
		in := &gorilla.Data{}
		err := d.Decode(in)
		if err == io.EOF {
			break
		}
		require.Nil(t, err)
		inputData = append(inputData, in)
	}
	assert.Equal(t, data, inputData)
}
