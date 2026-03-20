// Package webpenc implements a minimal VP8L (WebP lossless) encoder.
//
// This encoder produces valid WebP lossless images without requiring CGO or
// external libraries. It uses Huffman coding for each ARGB channel but does
// not apply LZ77 back-references or spatial transforms, so output files are
// larger than those produced by libwebp.
package webpenc

import (
	"bytes"
	"container/heap"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"io"
)

const maxCodeLen = 15 // VP8L maximum Huffman code length

// Encode writes img as a WebP lossless (VP8L) image to w.
func Encode(w io.Writer, img image.Image) error {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width <= 0 || height <= 0 || width > 16384 || height > 16384 {
		return fmt.Errorf("webpenc: image size %dx%d out of range", width, height)
	}

	npix := width * height
	gPix := make([]byte, npix)
	rPix := make([]byte, npix)
	bPix := make([]byte, npix)
	aPix := make([]byte, npix)
	hasAlpha := false

	idx := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			gPix[idx] = c.G
			rPix[idx] = c.R
			bPix[idx] = c.B
			aPix[idx] = c.A
			if c.A != 255 {
				hasAlpha = true
			}
			idx++
		}
	}

	bw := &bitWriter{}

	// VP8L signature byte.
	bw.writeBits(0x2f, 8)

	// Image size header: width-1 (14) | height-1 (14) | alpha (1) | version (3).
	hdr := uint32(width-1) | uint32(height-1)<<14
	if hasAlpha {
		hdr |= 1 << 28
	}
	bw.writeBits(hdr, 32)

	// No transforms.
	bw.writeBits(0, 1)
	// No color cache.
	bw.writeBits(0, 1)
	// No meta Huffman codes (single code group for entire image).
	bw.writeBits(0, 1)

	// Five Huffman codes: Green (280), Red (256), Blue (256), Alpha (256), Distance (40).
	const greenAlphaSize = 256 + 24

	gCodes := buildAndWriteHuffman(bw, byteFreq(gPix), greenAlphaSize)
	rCodes := buildAndWriteHuffman(bw, byteFreq(rPix), 256)
	bCodes := buildAndWriteHuffman(bw, byteFreq(bPix), 256)
	aCodes := buildAndWriteHuffman(bw, byteFreq(aPix), 256)

	// Distance: simple code, 1 unused symbol.
	bw.writeBits(1, 1) // simple_code
	bw.writeBits(0, 1) // num_symbols-1 = 0
	bw.writeBits(0, 1) // is_first=false → 1-bit symbol value 0

	// Encode pixels.
	for i := range npix {
		emit(bw, gCodes[gPix[i]])
		emit(bw, rCodes[rPix[i]])
		emit(bw, bCodes[bPix[i]])
		emit(bw, aCodes[aPix[i]])
	}

	bw.flush()
	return writeRIFF(w, bw.buf.Bytes())
}

func byteFreq(data []byte) []int {
	freq := make([]int, 256)
	for _, v := range data {
		freq[v]++
	}
	return freq
}

func emit(bw *bitWriter, c huffCode) {
	if c.length > 0 {
		bw.writeBits(c.code, c.length)
	}
}

func writeRIFF(w io.Writer, vp8l []byte) error {
	pad := len(vp8l) & 1
	fileSize := uint32(4 + 8 + len(vp8l) + pad) //nolint:gosec // bounded by 16384*16384*4+overhead, fits uint32
	chunkSize := uint32(len(vp8l))               //nolint:gosec // same bound

	var buf bytes.Buffer
	buf.Grow(8 + int(fileSize))
	buf.WriteString("RIFF")
	_ = binary.Write(&buf, binary.LittleEndian, fileSize)
	buf.WriteString("WEBP")
	buf.WriteString("VP8L")
	_ = binary.Write(&buf, binary.LittleEndian, chunkSize)
	buf.Write(vp8l)
	if pad != 0 {
		buf.WriteByte(0)
	}

	_, err := w.Write(buf.Bytes())
	return err
}

// ─── Bit Writer (LSB-first) ─────────────────────────────────────────────────

type bitWriter struct {
	buf   bytes.Buffer
	acc   uint64
	nbits uint
}

func (w *bitWriter) writeBits(val uint32, nbits uint) {
	w.acc |= uint64(val) << w.nbits
	w.nbits += nbits
	for w.nbits >= 8 {
		w.buf.WriteByte(byte(w.acc & 0xFF)) //nolint:gosec // intentional truncation to lowest byte
		w.acc >>= 8
		w.nbits -= 8
	}
}

func (w *bitWriter) flush() {
	if w.nbits > 0 {
		w.buf.WriteByte(byte(w.acc & 0xFF)) //nolint:gosec // intentional truncation to lowest byte
		w.acc = 0
		w.nbits = 0
	}
}

// ─── Huffman Codes ──────────────────────────────────────────────────────────

type huffCode struct {
	code   uint32
	length uint
}

// buildAndWriteHuffman builds a Huffman code from freq, writes it to the
// bitstream, and returns the code table indexed by symbol.
func buildAndWriteHuffman(bw *bitWriter, freq []int, alphaSize int) []huffCode {
	if len(freq) < alphaSize {
		ext := make([]int, alphaSize)
		copy(ext, freq)
		freq = ext
	}

	var used []int
	for i, f := range freq {
		if f > 0 {
			used = append(used, i)
		}
	}

	codes := make([]huffCode, alphaSize)

	switch len(used) {
	case 0:
		writeSimpleCode(bw, 0, -1)
		return codes
	case 1:
		writeSimpleCode(bw, used[0], -1)
		return codes
	case 2:
		s0, s1 := used[0], used[1]
		if s0 > s1 {
			s0, s1 = s1, s0
		}
		writeSimpleCode(bw, s0, s1)
		codes[s0] = huffCode{code: 0, length: 1}
		codes[s1] = huffCode{code: 1, length: 1}
		return codes
	}

	// 3+ symbols: build Huffman tree.
	lengths := huffmanLengths(freq, alphaSize)

	// If any code length exceeds 15, fall back to flat 8-bit codes.
	for _, l := range lengths {
		if l > maxCodeLen {
			lengths = make([]uint, alphaSize)
			for i := 0; i < alphaSize && i < 256; i++ {
				lengths[i] = 8
			}
			break
		}
	}

	codes = canonicalCodes(lengths)
	writeNormalCode(bw, lengths, alphaSize)
	return codes
}

// writeSimpleCode writes a VP8L simple Huffman code for 1 or 2 symbols.
// Pass sym1 < 0 for a single-symbol code.
func writeSimpleCode(bw *bitWriter, sym0, sym1 int) {
	s0 := uint32(sym0) //nolint:gosec // sym0 is always 0..255
	bw.writeBits(1, 1) // simple_code = true
	if sym1 < 0 {
		bw.writeBits(0, 1) // num_symbols-1 = 0
		if sym0 < 2 {
			bw.writeBits(0, 1)
			bw.writeBits(s0, 1)
		} else {
			bw.writeBits(1, 1)
			bw.writeBits(s0, 8)
		}
	} else {
		s1 := uint32(sym1) //nolint:gosec // sym1 is always 0..255
		bw.writeBits(1, 1) // num_symbols-1 = 1
		if sym0 < 2 {
			bw.writeBits(0, 1)
			bw.writeBits(s0, 1)
		} else {
			bw.writeBits(1, 1)
			bw.writeBits(s0, 8)
		}
		bw.writeBits(s1, 8)
	}
}

// ─── Huffman Tree Building ──────────────────────────────────────────────────

type hnode struct {
	freq int
	sym  int // leaf symbol; -1 for internal
	l, r *hnode
}

type hheap []*hnode

func (h hheap) Len() int { return len(h) }
func (h hheap) Less(i, j int) bool {
	if h[i].freq != h[j].freq {
		return h[i].freq < h[j].freq
	}
	return h[i].sym < h[j].sym
}
func (h hheap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *hheap) Push(x any) { *h = append(*h, x.(*hnode)) }
func (h *hheap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

func huffmanLengths(freq []int, size int) []uint {
	lengths := make([]uint, size)

	var nodes hheap
	for i := range size {
		if freq[i] > 0 {
			nodes = append(nodes, &hnode{freq: freq[i], sym: i})
		}
	}
	if len(nodes) < 2 {
		for _, n := range nodes {
			lengths[n.sym] = 1
		}
		return lengths
	}

	heap.Init(&nodes)
	for nodes.Len() > 1 {
		a := heap.Pop(&nodes).(*hnode)
		b := heap.Pop(&nodes).(*hnode)
		ms := a.sym
		if b.sym >= 0 && (ms < 0 || b.sym < ms) {
			ms = b.sym
		}
		heap.Push(&nodes, &hnode{freq: a.freq + b.freq, sym: ms, l: a, r: b})
	}

	var walk func(*hnode, uint)
	walk = func(n *hnode, d uint) {
		if n.l == nil {
			lengths[n.sym] = d
			return
		}
		walk(n.l, d+1)
		walk(n.r, d+1)
	}
	walk(nodes[0], 0)
	return lengths
}

// ─── Canonical Code Assignment ──────────────────────────────────────────────

func canonicalCodes(lengths []uint) []huffCode {
	n := len(lengths)
	codes := make([]huffCode, n)

	var maxLen uint
	for _, l := range lengths {
		if l > maxLen {
			maxLen = l
		}
	}
	if maxLen == 0 {
		return codes
	}

	blCount := make([]int, maxLen+1)
	for _, l := range lengths {
		if l > 0 {
			blCount[l]++
		}
	}

	nextCode := make([]uint32, maxLen+1)
	code := uint32(0)
	for bits := uint(1); bits <= maxLen; bits++ {
		code = (code + uint32(blCount[bits-1])) << 1
		nextCode[bits] = code
	}

	for sym := range n {
		l := lengths[sym]
		if l > 0 {
			codes[sym] = huffCode{
				code:   reverseBits(nextCode[l], l),
				length: l,
			}
			nextCode[l]++
		}
	}
	return codes
}

func reverseBits(v uint32, n uint) uint32 {
	var r uint32
	for i := uint(0); i < n; i++ {
		r = (r << 1) | (v & 1)
		v >>= 1
	}
	return r
}

// ─── Normal Huffman Code Serialization ──────────────────────────────────────

var clOrder = [19]int{17, 18, 0, 1, 2, 3, 4, 5, 16, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}

func writeNormalCode(bw *bitWriter, lengths []uint, alphaSize int) {
	// RLE-encode the code lengths.
	rlSym, rlExtra, rlExBits := rleEncode(lengths, alphaSize)

	// Build Huffman code for the code-length symbols (0..18).
	clFreq := make([]int, 19)
	for _, s := range rlSym {
		clFreq[s]++
	}
	clLen := codeLengthCodeLengths(clFreq)
	clCodes := canonicalCodes(clLen)

	bw.writeBits(0, 1) // simple_code = false

	// Determine how many code-length-code entries to write.
	numCL := 4
	for i := 18; i >= 4; i-- {
		if clLen[clOrder[i]] != 0 {
			numCL = i + 1
			break
		}
	}
	bw.writeBits(uint32(numCL-4), 4) //nolint:gosec // numCL is 4..19
	for i := range numCL {
		bw.writeBits(uint32(clLen[clOrder[i]]), 3) //nolint:gosec // code lengths are 0..7
	}

	// max_symbol = alphabet_size (default).
	bw.writeBits(0, 1)

	// Write the RLE-encoded code lengths.
	for i, sym := range rlSym {
		emit(bw, clCodes[sym])
		if rlExBits[i] > 0 {
			bw.writeBits(uint32(rlExtra[i]), rlExBits[i])
		}
	}
}

// codeLengthCodeLengths builds a Huffman code for the 19-symbol code-length
// alphabet, with code lengths capped at 7 (the 3-bit maximum).
func codeLengthCodeLengths(freq []int) []uint {
	active := 0
	single := -1
	for i, f := range freq {
		if f > 0 {
			active++
			single = i
		}
	}

	if active <= 1 {
		lengths := make([]uint, 19)
		if single >= 0 {
			lengths[single] = 1
		}
		return lengths
	}

	lengths := huffmanLengths(freq, 19)

	// Cap at 7 bits.
	for _, l := range lengths {
		if l > 7 {
			// Fall back to flat codes for all active symbols.
			bits := uint(1)
			for (1 << bits) < active {
				bits++
			}
			if bits > 7 {
				bits = 7
			}
			flat := make([]uint, 19)
			for i, f := range freq {
				if f > 0 {
					flat[i] = bits
				}
			}
			return flat
		}
	}
	return lengths
}

// rleEncode encodes a slice of code lengths using VP8L's run-length scheme.
func rleEncode(lengths []uint, size int) (syms []int, extras []int, exBits []uint) {
	i := 0
	for i < size {
		l := lengths[i]
		if l == 0 {
			j := i + 1
			for j < size && lengths[j] == 0 {
				j++
			}
			cnt := j - i
			i = j
			for cnt > 0 {
				if cnt >= 11 {
					n := cnt
					if n > 138 {
						n = 138
					}
					syms = append(syms, 18)
					extras = append(extras, n-11)
					exBits = append(exBits, 7)
					cnt -= n
				} else if cnt >= 3 {
					syms = append(syms, 17)
					extras = append(extras, cnt-3)
					exBits = append(exBits, 3)
					cnt = 0
				} else {
					syms = append(syms, 0)
					extras = append(extras, 0)
					exBits = append(exBits, 0)
					cnt--
				}
			}
		} else {
			// Emit one literal code length.
			li := int(l) //nolint:gosec // code lengths are 0..15
			syms = append(syms, li)
			extras = append(extras, 0)
			exBits = append(exBits, 0)
			i++

			// Encode runs of the same value with symbol 16 (repeat previous).
			j := i
			for j < size && lengths[j] == l {
				j++
			}
			run := j - i
			i = j

			for run >= 3 {
				n := run
				if n > 6 {
					n = 6
				}
				syms = append(syms, 16)
				extras = append(extras, n-3)
				exBits = append(exBits, 2)
				run -= n
			}
			for run > 0 {
				syms = append(syms, li)
				extras = append(extras, 0)
				exBits = append(exBits, 0)
				run--
			}
		}
	}
	return
}
