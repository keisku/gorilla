# Gorilla

Gorilla provides a encoder/decoder package based on Facebook's Gorilla.

Pelkonen, T., Franklin, S., Teller, J., Cavallaro, P., Huang, Q., Meza, J., &#38; Veeraraghavan, K. (2015). Gorilla. Proceedings of the VLDB Endowment, 8(12), 1816–1827. https://doi.org/10.14778/2824032.2824078

## Usage

### Installing

```shell
go get github.com/kei6u/gorilla
```

### Encoder

```go

buf := new(bytes.Buffer)
e := gorilla.NewEncoder(buf)

h := uint32(time.Now().Unix())
if err := e.PutHeader(h); err != nil {
    return err
}

if err := e.Encode(uint32(time.Now().Unix()), 10.0); err != nil {
    return err
}
if err := e.Encode(uint32(time.Now().Unix()), 10.5); err != nil {
    return err
}

return e.Flush()
```

### Decoder

```go

buf := new(bytes.Buffer)

// Encoding data to buf ...

h, err := d.LoadHeader()
if err != nil {
    return err
}

var data []*gorilla.Data
for {
    in := &gorilla.Data{}
    err := d.Decode(in)
    if err == io.EOF {
        break
    }
    if err != nil {
        return err
    }
    data = append(data, in)
}

return data
```
