# Gorilla

Gorilla provides a compression/decompression time-series package based on Facebook's Gorilla.

Pelkonen, T., Franklin, S., Teller, J., Cavallaro, P., Huang, Q., Meza, J., &#38; Veeraraghavan, K. (2015). Gorilla. Proceedings of the VLDB Endowment, 8(12), 1816–1827. https://doi.org/10.14778/2824032.2824078

## What is it?

This package provides the effective time-series data compression method.
The method follows Facebook's Gorilla paper, so can save a lot of storage footprint.
In a nutshell, it uses delta-of-delta timestamps and XOR'd floating-points values because most data points arrived at a fixed interval and the value in most time-series does not change significantly.

## Notes

### Use timestamp seconds to keep high compression rate

The usage of milliseconds as a timestamp will increase the amount of bits because timestamps are no longer evenly interval.
A delta of delta second timestamp which arrives at a certain interval is basically 0, 1 or -1, so this package can keep small data size.

### As the value of the fraction increases, the compression ratio decreases.

Each value is converted to a 64-bit sequence of floating point numbers and XORed with one previous data point. If the values are exactly the same, they will be zero, so encoding them will only require one bit of zero.
Also, instead of recording the result of the XOR in 64 bits each time, we encode the bits from the beginning that are followed by 0 (LeadingZeros) and the bits from the end that are followed by 0 (TrailingZeros), and record the rest of the bit sequence.
In addition, if there are more digits in the LeadingZeros and TrailingZeros than in the previous value, they are left alone and only the rest of the bit sequence is recorded.
Otherwise, the new LeadingZeros and TrailingZeros values and the rest of the bit sequence are recorded.
This encoding scheme requires fewer bits if the value is 63.0, 63.5, or some other floating-point number with many zero bits from the middle to the end of the mantissa part. However, for a number like 0.1, many bits in the mantissa are non-zero, so the LeadingZeros and TrailingZeros values become small, and it takes a lot of bits to record the rest of the bit sequence.

## Usage

### Installing

```shell
go get github.com/kei6u/gorilla
```

### Compressor

```go

buf := new(bytes.Buffer)
header := uint32(time.Now().Unix())

c, finish, err := gorilla.NewCompressor(buf, header)
if err != nil {
    return err
}

if err := e.Compress(uint32(time.Now().Unix()), 10.0); err != nil {
    return err
}
if err := e.Compress(uint32(time.Now().Unix()), 10.5); err != nil {
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

iter := d.Iter()
for iter.Next() {
    t, v := iter.Get()
    fmt.Println(t, v)
}

return iter.Err()
```
