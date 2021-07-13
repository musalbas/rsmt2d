// +build leopard

// Note that if the build tag leopard is used, liblibleopard.a
// has to be present where the linker will find it.
// Otherwise go-leopard won't build.
package rsmt2d

import "github.com/celestiaorg/go-leopard"

var _ Codec = leoRSFF8Codec{}
var _ Codec = leoRSFF16Codec{}

func init() {
	registerCodec(LeopardFF8, newLeoRSFF8Codec())
	registerCodec(LeopardFF16, newLeoRSFF16Codec())
}

type leoRSFF8Codec struct{}

func (l leoRSFF8Codec) Encode(data [][]byte) ([][]byte, error) {
	return leopard.Encode(data)
}

func (l leoRSFF8Codec) Decode(data [][]byte) ([][]byte, error) {
	half := len(data) / 2
	return leopard.Decode(data[:half], data[half:])
}

func (l leoRSFF8Codec) maxChunks() int {
	return 128 * 128
}

func newLeoRSFF8Codec() leoRSFF8Codec {
	return leoRSFF8Codec{}
}

type leoRSFF16Codec struct{}

func (leo leoRSFF16Codec) Encode(data [][]byte) ([][]byte, error) {
	return leopard.Encode(data)
}

func (leo leoRSFF16Codec) Decode(data [][]byte) ([][]byte, error) {
	half := len(data) / 2
	return leopard.Decode(data[:half], data[half:])
}

func (leo leoRSFF16Codec) maxChunks() int {
	return 32768 * 32768
}

func newLeoRSFF16Codec() leoRSFF16Codec {
	return leoRSFF16Codec{}
}
