package rsmt2d

import "fmt"

const (
	LeopardFF16 = "LeopardFF16"
	LeopardFF8  = "LeopardFF8"
	RSGF8       = "RSFG8"
)

type Codec interface {
	Encode(data [][]byte) ([][]byte, error)
	Decode(data [][]byte) ([][]byte, error)
	New() Codec
	// maxChunks returns the max. number of chunks each code supports in a 2D square.
	maxChunks() int
}

// codecs is a global map used for keeping track of which codecs are included during testing
var codecs = make(map[string]Codec)

func registerCodec(ct string, codec Codec) {
	if codecs[ct] != nil {
		panic(fmt.Sprintf("%v already registered", codec))
	}
	codecs[ct] = codec
}

func NewLeoRSFF16Codec() Codec {
	if codec, has := codecs[LeopardFF16]; has {
		return codec
	}
	panic("cannot use codec LeopardFF16 without 'leopard' build tag")
}

func NewLeoRSFF8Codec() Codec {
	if codec, has := codecs[LeopardFF8]; has {
		return codec
	}
	panic("cannot use codec LeopardFF8 without 'leopard' build tag")
}
