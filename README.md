# Gorilla

Gorilla provides a compression/decompression time-series package based on Facebook's Gorilla.

Pelkonen, T., Franklin, S., Teller, J., Cavallaro, P., Huang, Q., Meza, J., &#38; Veeraraghavan, K. (2015). Gorilla. Proceedings of the VLDB Endowment, 8(12), 1816–1827. https://doi.org/10.14778/2824032.2824078

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

iter := d.Iter()
for iter.Next() {
    t, v := iter.Get()
    fmt.Println(t, v)
}

return iter.Err()
```
