# Gorilla

[![Go Reference](https://pkg.go.dev/badge/github.com/keisku/gorilla.svg)](https://pkg.go.dev/github.com/keisku/gorilla)
[![.github/workflows/test.yaml](https://github.com/keisku/gorilla/actions/workflows/test.yaml/badge.svg)](https://github.com/keisku/gorilla/actions/workflows/test.yaml)


An effective time-series data compression method based on Facebook's Gorilla.
It uses delta-of-delta timestamps and XOR'd floating-points values to reduce the data size.
This is because most data points arrived at a fixed interval and the value in most time-series does not change significantly.

Pelkonen, T., Franklin, S., Teller, J., Cavallaro, P., Huang, Q., Meza, J., &#38; Veeraraghavan, K. (2015). Gorilla. Proceedings of the VLDB Endowment, 8(12), 1816–1827. https://doi.org/10.14778/2824032.2824078

## Notes

### Use timestamp seconds to keep high compression rate

Using milliseconds as a timestamp increases the amount of bits because timestamps arrives at no longer evenly interval.
A delta of delta second timestamp which arrives at a certain interval is basically 0, 1 or -1, so it can keep small data size.

### As the value fraction increases, the compression ratio decreases

This method converts each value to a 64-bit sequence of floating-point numbers and XORed with one previous data point. If the values are the same, they will be zero, so encoding them will only require one bit of zero.
Also, instead of recording the result of the XOR in 64 bits each time, `gorilla` compresses the bits from the beginning that are followed by 0 (LeadingZeros) and the bits from the end that are followed by 0 (TrailingZeros) and record the rest of the bit sequence.
In addition, if there are more digits in the LeadingZeros and TrailingZeros than in the previous value, they are left alone, and only the rest of the bit sequence is recorded.
Otherwise, the new LeadingZeros and TrailingZeros values and the rest of the bit sequence are recorded.
`gorilla` requires fewer bits if the value is 63.0, 63.5, or some other floating-point number with many zero bits from the middle to the end of the mantissa part. However, for a number like 0.1, many bits in the mantissa are non-zero, so the LeadingZeros and TrailingZeros values become small, and it takes a lot of bits to record the rest of the bit sequence.

### Compression window should be longer than 2 hours

Using the same compressing scheme over larger time periods allows us to achieve better compression ratios.
However, queries that wish to read data over a short time range might need to expend additional computational resources on decompressing data.
The paper shows the average compression ratio for the time series as we change the block size.
One can see that blocks that extend longer than two hours provide diminishing returns for compressed size.
A two-hour block allows us to achieve a compression ratio of 1.37 bytes per data point.

## Usage

### Installing

```shell
go get github.com/keisku/gorilla
```

### Compressor

```go

buf := new(bytes.Buffer)
header := uint32(time.Now().Unix())

c, finish, err := gorilla.NewCompressor(buf, header)
if err != nil {
    return err
}

if err := c.Compress(uint32(time.Now().Unix()), 10.0); err != nil {
    return err
}
if err := c.Compress(uint32(time.Now().Unix()), 10.5); err != nil {
    return err
}

return finish()
```

### Decompressor

```go

buf := new(bytes.Buffer)

// Compressing time-series data to buf ...

d, h, err := gorilla.NewDecompressor(buf)
if err != nil {
    return err
}

fmt.Printf("header: %v\n", h)

iter := d.Iterator()
for iter.HasNext() {
    t, v := iter.Next()
    fmt.Println(t, v)
}

return iter.Err()
```
