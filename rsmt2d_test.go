package rsmt2d_test

import (
	"bytes"
	"testing"

	"github.com/celestiaorg/rsmt2d"
)

func TestRSMT2D(t *testing.T) {
	// Size of each share, in bytes
	bufferSize := 64
	// Init new codec
	codec := rsmt2d.NewLeoRSFF8Codec()

	ones := bytes.Repeat([]byte{1}, bufferSize)
	twos := bytes.Repeat([]byte{2}, bufferSize)
	threes := bytes.Repeat([]byte{3}, bufferSize)
	fours := bytes.Repeat([]byte{4}, bufferSize)

	// Compute parity shares
	eds, err := rsmt2d.ComputeExtendedDataSquare(
		[][]byte{
			ones, twos,
			threes, fours,
		},
		codec,
		rsmt2d.NewDefaultTree,
	)
	if err != nil {
		t.Errorf("ComputeExtendedDataSquare failed")
	}

	// Save all shares in flattened form.
	flattened := make([][]byte, 0, eds.Width()*eds.Width())
	for i := uint(0); i < eds.Width(); i++ {
		flattened = append(flattened, eds.Row(i)...)
	}

	// Delete some shares, just enough so that repairing is possible.
	flattened[0], flattened[2], flattened[3] = nil, nil, nil
	flattened[4], flattened[5], flattened[6], flattened[7] = nil, nil, nil, nil
	flattened[8], flattened[9], flattened[10] = nil, nil, nil
	flattened[12], flattened[13] = nil, nil

	// Repair square.
	repaired, err := rsmt2d.RepairExtendedDataSquare(
		eds.RowRoots(),
		eds.ColRoots(),
		flattened,
		codec,
		rsmt2d.NewDefaultTree,
	)
	if err != nil {
		// err contains information to construct a fraud proof
		// See extendeddatacrossword_test.go
		t.Errorf("RepairExtendedDataSquare failed")
	}
	_ = repaired
}
