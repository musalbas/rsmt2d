package rsmt2d

import (
	"bytes"
	"errors"
	"fmt"
	"math"
)

const (
	row = iota
	column
)

// ErrUnrepairableDataSquare is thrown when there is insufficient chunks to repair the square.
var ErrUnrepairableDataSquare = errors.New("failed to solve data square")

// ErrByzantineRow is thrown when there is a repaired row does not match the expected row merkle root.
type ErrByzantineRow struct {
	RowNumber uint
}

func (e *ErrByzantineRow) Error() string {
	return fmt.Sprintf("byzantine row: %d", e.RowNumber)
}

// ErrByzantineColumn is thrown when there is a repaired column does not match the expected column merkle root.
type ErrByzantineColumn struct {
	ColumnNumber uint
}

func (e *ErrByzantineColumn) Error() string {
	return fmt.Sprintf("byzantine column: %d", e.ColumnNumber)
}

// RepairExtendedDataSquare repairs an incomplete extended data square, against its expected row and column merkle roots.
// Missing data chunks should be represented as nil.
func RepairExtendedDataSquare(
	rowRoots [][]byte,
	columnRoots [][]byte,
	data [][]byte,
	codec CodecType,
	treeCreatorFn TreeConstructorFn,
) (*ExtendedDataSquare, error) {
	width := int(math.Ceil(math.Sqrt(float64(len(data)))))
	bitMat := newBitMatrix(width)
	var chunkSize int
	for i := range data {
		if data[i] != nil {
			bitMat.SetFlat(i)
			if chunkSize == 0 {
				chunkSize = len(data[i])
			}
		}
	}

	if chunkSize == 0 {
		return nil, ErrUnrepairableDataSquare
	}

	fillerChunk := bytes.Repeat([]byte{0}, chunkSize)
	for i := range data {
		if data[i] == nil {
			data[i] = make([]byte, chunkSize)
			copy(data[i], fillerChunk)
		}
	}

	eds, err := ImportExtendedDataSquare(data, codec, treeCreatorFn)
	if err != nil {
		return nil, err
	}

	err = eds.prerepairSanityCheck(rowRoots, columnRoots, bitMat)
	if err != nil {
		return nil, err
	}

	err = eds.solveCrossword(rowRoots, columnRoots, bitMat)
	if err != nil {
		return nil, err
	}

	return eds, err
}

func (eds *ExtendedDataSquare) solveCrossword(rowRoots [][]byte, columnRoots [][]byte, bitMask bitMatrix) error {
	// TODO(john): re-add eds
	// Keep repeating until the square is solved
	solved := false
	for {
		solved = true
		progressMade := false

		// Loop through every row and column, attempt to rebuild each row or column if incomplete
		for i := 0; i < int(eds.width); i++ {
			for mode := range []int{row, column} {
				var isIncomplete bool
				var isExtendedPartIncomplete bool
				switch mode {
				case row:
					isIncomplete = !bitMask.RowIsOne(i)
					isExtendedPartIncomplete = !bitMask.RowRangeIsOne(i, int(eds.originalDataWidth), int(eds.width))
				case column:
					isIncomplete = !bitMask.ColumnIsOne(i)
					isExtendedPartIncomplete = !bitMask.ColRangeIsOne(i, int(eds.originalDataWidth), int(eds.width))
				default:
					panic(fmt.Sprintf("invalid mode %d", mode))
				}

				if isIncomplete { // row/column incomplete
					// Prepare shares
					shares := make([][]byte, eds.width)
					for j := 0; j < int(eds.width); j++ {
						var vectorData [][]byte
						var r, c int
						switch mode {
						case row:
							r = i
							c = j
							vectorData = eds.Row(uint(i))
						case column:
							r = j
							c = i
							vectorData = eds.Column(uint(i))
						default:
							panic(fmt.Sprintf("invalid mode %d", mode))
						}
						if bitMask.Get(r, c) {
							// As guaranteed by the bitMask, vectorData can't be nil here:
							shares[j] = vectorData[j]
						}
					}

					// Attempt rebuild
					rebuiltShares, err := Decode(shares, eds.codec)
					if err != nil {
						// repair unsuccessful
						solved = false
						continue
					}

					progressMade = true

					if isExtendedPartIncomplete {
						// If needed, rebuild the parity shares too.
						rebuiltExtendedShares, err := Encode(rebuiltShares, eds.codec)
						if err != nil {
							return err
						}
						rebuiltShares = append(rebuiltShares, rebuiltExtendedShares...)
					} else {
						// Otherwise copy them from the EDS.
						rebuiltShares = append(rebuiltShares, shares[eds.width:]...)
					}

					// Check that rebuilt shares matches appropriate root
					_, err = eds.verifyAgainstRoots(rowRoots, columnRoots, mode, uint(i), rebuiltShares)
					if err != nil {
						return err
					}

					// Check that newly completed orthogonal vectors match their new merkle roots
					for j := 0; j < int(eds.width); j++ {
						switch mode {
						case row:
							if !bitMask.Get(i, j) &&
								bitMask.ColumnIsOne(j) {
								_, err := eds.verifyAgainstRoots(rowRoots, columnRoots, column, uint(j), rebuiltShares)
								if err != nil {
									return &ErrByzantineColumn{uint(j)}
								}
							}

						case column:
							if !bitMask.Get(j, i) &&
								bitMask.RowIsOne(j) {
								_, err := eds.verifyAgainstRoots(rowRoots, columnRoots, row, uint(j), rebuiltShares)
								if err != nil {
									return &ErrByzantineRow{uint(j)}
								}
							}

						default:
							panic(fmt.Sprintf("invalid mode %d", mode))
						}
					}

					// Set vector mask to true
					switch mode {
					case row:
						for j := 0; j < int(eds.width); j++ {
							bitMask.Set(i, j)
						}
					case column:
						for j := 0; j < int(eds.width); j++ {
							bitMask.Set(j, i)
						}
					default:
						panic(fmt.Sprintf("invalid mode %d", mode))
					}

					// Insert rebuilt shares into square.
					for p, s := range rebuiltShares {
						switch mode {
						case row:
							eds.setCell(uint(i), uint(p), s)
						case column:
							eds.setCell(uint(p), uint(i), s)
						default:
							panic(fmt.Sprintf("invalid mode %d", mode))
						}
					}
				}
			}
		}

		if solved {
			break
		} else if !progressMade {
			return ErrUnrepairableDataSquare
		}
	}

	return nil
}

func (eds *ExtendedDataSquare) verifyAgainstRoots(rowRoots [][]byte, columnRoots [][]byte, mode int, i uint, shares [][]byte) ([]byte, error) {
	tree := eds.createTreeFn()
	for cell, d := range shares {
		tree.Push(d, SquareIndex{Cell: uint(cell), Axis: i})
	}

	root := tree.Root()

	switch mode {
	case row:
		if !bytes.Equal(root, rowRoots[i]) {
			return nil, &ErrByzantineRow{i}
		}
	case column:
		if !bytes.Equal(root, columnRoots[i]) {
			return nil, &ErrByzantineColumn{i}
		}
	default:
		panic(fmt.Sprintf("invalid mode %d", mode))
	}
	return root, nil
}

func (eds *ExtendedDataSquare) prerepairSanityCheck(rowRoots [][]byte, columnRoots [][]byte, bitMask bitMatrix) error {
	for i := uint(0); i < eds.width; i++ {
		rowIsComplete := bitMask.RowIsOne(int(i))
		colIsComplete := bitMask.ColumnIsOne(int(i))

		// if there's no missing data in the this row
		if noMissingData(eds.Row(i)) {
			// ensure that the roots are equal and that rowMask is a vector
			if rowIsComplete && !bytes.Equal(rowRoots[i], eds.RowRoot(i)) {
				return fmt.Errorf("bad root input: row %d expected %v got %v", i, rowRoots[i], eds.RowRoot(i))
			}
		}

		// if there's no missing data in the this col
		if noMissingData(eds.Column(i)) {
			// ensure that the roots are equal and that rowMask is a vector
			if colIsComplete && !bytes.Equal(columnRoots[i], eds.ColRoot(i)) {
				return fmt.Errorf("bad root input: col %d expected %v got %v", i, columnRoots[i], eds.ColRoot(i))
			}
		}

		if rowIsComplete {
			parityShares, err := Encode(eds.rowSlice(i, 0, eds.originalDataWidth), eds.codec)
			if err != nil {
				return err
			}
			if !bytes.Equal(flattenChunks(parityShares), flattenChunks(eds.rowSlice(i, eds.originalDataWidth, eds.originalDataWidth))) {
				return &ErrByzantineRow{i}
			}
		}

		if colIsComplete {
			parityShares, err := Encode(eds.columnSlice(0, i, eds.originalDataWidth), eds.codec)
			if err != nil {
				return err
			}
			if !bytes.Equal(flattenChunks(parityShares), flattenChunks(eds.columnSlice(eds.originalDataWidth, i, eds.originalDataWidth))) {
				return &ErrByzantineColumn{i}
			}
		}
	}

	return nil
}

func noMissingData(input [][]byte) bool {
	for _, d := range input {
		if d == nil {
			return false
		}
	}
	return true
}
