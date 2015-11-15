package octree

import (
	//"unsafe"

	"gopkg.in/check.v1"
)

type Interleave3DSuite struct{}

var _ = check.Suite(&Interleave3DSuite{})

// These bit interleaving was taken from:
// https://graphics.stanford.edu/~seander/bithacks.html#InterleaveTableObvious
// And adapted to 3D. There is also some reference material from
// http://stackoverflow.com/questions/1024754/how-to-compute-a-3d-morton-number-interleave-the-bits-of-3-ints
// and http://www.forceflow.be/2013/10/07/morton-encodingdecoding-through-bit-interleaving-implementations/
// We do a bit of testing to make sure our implementations are correct, and
// then we do some benchmarking to see what actually makes a difference.

// Results so far:
// 599,834,300 Interleave3DObvious
//  37,622,152 NoOp
// 117,356,710 Interleave3DMagic32
//  76,654,385 Interleave3DLUT
//  59,463,402 Interleave3DLUT -gcflags -B
// Shifting bits to make space is about 5x faster than the 'obvious' way, but a
// LUT is still faster yet. With bounds checking disabled we are less than 2x
// slower than a NoOp. With bounds checking on, we're still 7.8x faster than
// the 'obvious' way.

func interleave3DObvious(x, y, z uint8) uint32 {
	val := uint32(0)
	xx := uint32(x)
	yy := uint32(y)
	zz := uint32(z)
	for i := uint(0); i < 8; i++ {
		// Select the i-th bit and shift it by i locations for x, and
		// shift it by i+1 for y
		bit := uint32(0x1) << i
		val |= (xx&bit)<<(2*i) |
			(yy&bit)<<(2*i+1) |
			(zz&bit)<<(2*i+2)
	}
	return val
}

const (
	// The source was doing 21bits of a uint32 into a uint64, but we're just
	// doing 3-uint8s into a uint32
	B0c = uint32(0x249249)
	B1c = uint32(0x0c30c3)
	B2c = uint32(0x00f00f)
	S0c = 2
	S1c = 4
	S2c = 8
)

func interleave3DMagic32(x, y, z uint8) uint32 {
	xx := uint32(x)
	yy := uint32(y)
	zz := uint32(z)

	xx = (xx | (xx << S2c)) & B2c
	xx = (xx | (xx << S1c)) & B1c
	xx = (xx | (xx << S0c)) & B0c

	yy = (yy | (yy << S2c)) & B2c
	yy = (yy | (yy << S1c)) & B1c
	yy = (yy | (yy << S0c)) & B0c

	zz = (zz | (zz << S2c)) & B2c
	zz = (zz | (zz << S1c)) & B1c
	zz = (zz | (zz << S0c)) & B0c

	return xx | (yy << 1) | (zz << 2)
}

func splitBy3(x uint8) uint32 {
	xx := uint32(x)
	xx = (xx | (xx << S2c)) & B2c
	xx = (xx | (xx << S1c)) & B1c
	xx = (xx | (xx << S0c)) & B0c
	return xx
}

func interleave3DLUT(x, y, z uint8) uint32 {
	return morton256_3D[x] + morton256_3D[y]<<1 + morton256_3D[z]<<2
}

type interleave3DTest struct {
	x, y, z     uint8
	interleaved uint32
}

var interleave3DTests = []interleave3DTest{
	// Check some obvious interleavings, and check each bit gets mapped to the
	// correct final bit
	{x: 0x00, y: 0x00, z: 0x00, interleaved: 0x000000},
	{x: 0xff, y: 0xff, z: 0xff, interleaved: 0xffffff},
	{x: 0x01, y: 0x01, z: 0x01, interleaved: 0x000007},
	{x: 0x01, y: 0x00, z: 0x00, interleaved: 0x000001},
	{x: 0x02, y: 0x00, z: 0x00, interleaved: 0x000008},
	{x: 0x04, y: 0x00, z: 0x00, interleaved: 0x000040},
	{x: 0x08, y: 0x00, z: 0x00, interleaved: 0x000200},
	{x: 0x10, y: 0x00, z: 0x00, interleaved: 0x001000},
	{x: 0x20, y: 0x00, z: 0x00, interleaved: 0x008000},
	{x: 0x40, y: 0x00, z: 0x00, interleaved: 0x040000},
	{x: 0x80, y: 0x00, z: 0x00, interleaved: 0x200000},
	{x: 0x00, y: 0x01, z: 0x00, interleaved: 0x000002},
	{x: 0x00, y: 0x02, z: 0x00, interleaved: 0x000010},
	{x: 0x00, y: 0x04, z: 0x00, interleaved: 0x000080},
	{x: 0x00, y: 0x08, z: 0x00, interleaved: 0x000400},
	{x: 0x00, y: 0x10, z: 0x00, interleaved: 0x002000},
	{x: 0x00, y: 0x20, z: 0x00, interleaved: 0x010000},
	{x: 0x00, y: 0x40, z: 0x00, interleaved: 0x080000},
	{x: 0x00, y: 0x80, z: 0x00, interleaved: 0x400000},
	{x: 0x00, y: 0x00, z: 0x01, interleaved: 0x000004},
	{x: 0x00, y: 0x00, z: 0x02, interleaved: 0x000020},
	{x: 0x00, y: 0x00, z: 0x04, interleaved: 0x000100},
	{x: 0x00, y: 0x00, z: 0x08, interleaved: 0x000800},
	{x: 0x00, y: 0x00, z: 0x10, interleaved: 0x004000},
	{x: 0x00, y: 0x00, z: 0x20, interleaved: 0x020000},
	{x: 0x00, y: 0x00, z: 0x40, interleaved: 0x100000},
	{x: 0x00, y: 0x00, z: 0x80, interleaved: 0x800000},
}

func (*Interleave3DSuite) TestSplitBy3(c *check.C) {
	c.Check(splitBy3(0x01), check.Equals, uint32(0x000001))
	c.Check(splitBy3(0x02), check.Equals, uint32(0x000008))
	c.Check(splitBy3(0x04), check.Equals, uint32(0x000040))
	c.Check(splitBy3(0x08), check.Equals, uint32(0x000200))
	c.Check(splitBy3(0x10), check.Equals, uint32(0x001000))
	c.Check(splitBy3(0x20), check.Equals, uint32(0x008000))
	c.Check(splitBy3(0x40), check.Equals, uint32(0x040000))
	c.Check(splitBy3(0x80), check.Equals, uint32(0x200000))
}

func (*Interleave3DSuite) TestMortonTable(c *check.C) {
	for x := 0; x < 256; x++ {
		xx := uint8(x)
		c.Check(splitBy3(xx), check.Equals, morton256_3D[x])
	}
}

func (*Interleave3DSuite) TestInterleave3DObvious(c *check.C) {
	for _, vals := range interleave3DTests {
		interleaved := interleave3DObvious(vals.x, vals.y, vals.z)
		c.Check(interleaved,
			check.Equals, vals.interleaved,
			check.Commentf("expected %x %x %x to become %06x not %06x",
				vals.x, vals.y, vals.z, vals.interleaved, interleaved))
	}
}

func checkInterleave3DMatchesObvious(c *check.C, f func(x, y, z uint8) uint32) {
	for _, vals := range interleave3DTests {
		interleaved := f(vals.x, vals.y, vals.z)
		c.Check(interleaved,
			check.Equals, vals.interleaved,
			check.Commentf("expected %v(0x%x,0x%x) = 0x%x not 0x%x",
				funcName(f),
				vals.x, vals.y,
				vals.interleaved, interleaved))
	}
	// In the 2D case we could do an exhaustive search, but it turns out that
	// is prohibitively slow in 3D (algorithms are slower, an the space is 256x
	// bigger). So we pick a unique stride for each dimension and just sample
	// it.
	for z := 0; z <= 0xFF; z += 7 {
		zz := uint8(z)
		for y := 0; y <= 0xFF; y += 5 {
			yy := uint8(y)
			for x := 0; x <= 0xFF; x += 3 {
				xx := uint8(x)
				c.Assert(f(xx, yy, zz), check.Equals,
					interleave3DObvious(xx, yy, zz))
			}
		}
	}
	c.Assert(f(0xFF, 0xFF, 0xFF),
		check.Equals, interleave3DObvious(0xFF, 0xFF, 0xFF))
}

func (*Interleave3DSuite) TestInterleave3DMagic32(c *check.C) {
	checkInterleave3DMatchesObvious(c, interleave3DMagic32)
}

func (*Interleave3DSuite) TestInterleave3DLUT(c *check.C) {
	checkInterleave3DMatchesObvious(c, interleave3DLUT)
}

func benchInterleave3D(c *check.C, f func(x, y, z uint8) uint32) {
	for i := 0; i < c.N; i++ {
		for z := uint8(0); z < 255; z++ {
			for y := uint8(0); y < 255; y++ {
				for x := uint8(0); x < 255; x++ {
					f(x, y, z)
				}
				f(0xFF, y, z)
			}
			f(0xFF, 0xFF, z)
		}
		f(0xFF, 0xFF, 0xFF)
	}
}

func (*Interleave3DSuite) BenchmarkInterleave3DObvious(c *check.C) {
	benchInterleave3D(c, interleave3DObvious)
}

func (*Interleave3DSuite) BenchmarkInterleave3DMagic32(c *check.C) {
	benchInterleave3D(c, interleave3DMagic32)
}

func (*Interleave3DSuite) BenchmarkInterleave3DLUT(c *check.C) {
	benchInterleave3D(c, interleave3DLUT)
}

func (*Interleave3DSuite) BenchmarkInterleave3DNoOp(c *check.C) {
	// This gives a baseline for just what it costs to make 65536 function
	// calls
	benchInterleave3D(c, func(x, y, z uint8) uint32 { return 0 })
}

///
/// func init() {
/// 	morton256by1 = make([]uint16, 256)
/// 	morton256by16 = make([]uint16, 256)
/// 	morton256by17 = make([]uint16, 256)
/// 	for idx, m := range morton256 {
/// 		morton256by1[idx] = m << 1
/// 		morton256by16[idx] = m << 16
/// 		morton256by17[idx] = m << 17
/// 	}
/// 	mP = unsafe.Pointer(&morton256[0])
/// 	m1P = unsafe.Pointer(&morton256by1[0])
/// 	m16P = unsafe.Pointer(&morton256by16[0])
/// 	m17P = unsafe.Pointer(&morton256by17[0])
/// }
