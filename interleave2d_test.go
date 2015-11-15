package octree

import (
	"reflect"
	"runtime"
	"unsafe"

	"gopkg.in/check.v1"
)

type Interleave2DSuite struct{}

var _ = check.Suite(&Interleave2DSuite{})

// These bit interleaving was taken from:
// https://graphics.stanford.edu/~seander/bithacks.html#InterleaveTableObvious
// Note, though, that they are only interleaving 2 values, (x,y) while we want
// to interleave (x,y,z) values
// We do a bit of testing to make sure our implementations are correct, and
// then we do some benchmarking to see what actually makes a difference.

// Results so far:
// 1,907,109 Interleave2DObvious
//   152,208 NoOp
//   369,020 Interleave2DMagic16
//   325,418 Interleave2DMagic32
//   226,913 Interleave2DMorton
//   216,712 Interleave2DMorton2
//   193,111 Interleave2DMorton2Unsafe
//   182,810 Interleave2DMorton -gcflags -B
//   176,910 Interleave2DMorton2 -gcflags -B
//   252,014 Interleave2DMultiply64
// Using a lookup table is still the fastest, though multiplication is quite close.
// Using 'unsafe' is slightly faster than with bounds checking on, but with
// bounds checking off it is actually slower.
// Doing bit magic work to OR the value with shifted versions of itself is a bit
// slower than a Morton lookup. And the 32-bit version is faster than the
// 16-bit version, but we didn't eliminate any computation.
// Using 64bit multiplication is right in the middle (faster than bit shifting,
// slower than lookup tables.)

func interleave2DObvious(x, y uint8) uint16 {
	z := uint16(0)
	for i := uint(0); i < 8; i++ {
		// Select the i-th bit and shift it by i locations for x, and
		// shift it by i+1 for y
		z |= ((uint16(x) & (0x1 << i)) << i) |
			((uint16(y) & (0x1 << i)) << (i + 1))
	}
	return z
}

var morton256 = []uint16{
	0x0000, 0x0001, 0x0004, 0x0005, 0x0010, 0x0011, 0x0014, 0x0015,
	0x0040, 0x0041, 0x0044, 0x0045, 0x0050, 0x0051, 0x0054, 0x0055,
	0x0100, 0x0101, 0x0104, 0x0105, 0x0110, 0x0111, 0x0114, 0x0115,
	0x0140, 0x0141, 0x0144, 0x0145, 0x0150, 0x0151, 0x0154, 0x0155,
	0x0400, 0x0401, 0x0404, 0x0405, 0x0410, 0x0411, 0x0414, 0x0415,
	0x0440, 0x0441, 0x0444, 0x0445, 0x0450, 0x0451, 0x0454, 0x0455,
	0x0500, 0x0501, 0x0504, 0x0505, 0x0510, 0x0511, 0x0514, 0x0515,
	0x0540, 0x0541, 0x0544, 0x0545, 0x0550, 0x0551, 0x0554, 0x0555,
	0x1000, 0x1001, 0x1004, 0x1005, 0x1010, 0x1011, 0x1014, 0x1015,
	0x1040, 0x1041, 0x1044, 0x1045, 0x1050, 0x1051, 0x1054, 0x1055,
	0x1100, 0x1101, 0x1104, 0x1105, 0x1110, 0x1111, 0x1114, 0x1115,
	0x1140, 0x1141, 0x1144, 0x1145, 0x1150, 0x1151, 0x1154, 0x1155,
	0x1400, 0x1401, 0x1404, 0x1405, 0x1410, 0x1411, 0x1414, 0x1415,
	0x1440, 0x1441, 0x1444, 0x1445, 0x1450, 0x1451, 0x1454, 0x1455,
	0x1500, 0x1501, 0x1504, 0x1505, 0x1510, 0x1511, 0x1514, 0x1515,
	0x1540, 0x1541, 0x1544, 0x1545, 0x1550, 0x1551, 0x1554, 0x1555,
	0x4000, 0x4001, 0x4004, 0x4005, 0x4010, 0x4011, 0x4014, 0x4015,
	0x4040, 0x4041, 0x4044, 0x4045, 0x4050, 0x4051, 0x4054, 0x4055,
	0x4100, 0x4101, 0x4104, 0x4105, 0x4110, 0x4111, 0x4114, 0x4115,
	0x4140, 0x4141, 0x4144, 0x4145, 0x4150, 0x4151, 0x4154, 0x4155,
	0x4400, 0x4401, 0x4404, 0x4405, 0x4410, 0x4411, 0x4414, 0x4415,
	0x4440, 0x4441, 0x4444, 0x4445, 0x4450, 0x4451, 0x4454, 0x4455,
	0x4500, 0x4501, 0x4504, 0x4505, 0x4510, 0x4511, 0x4514, 0x4515,
	0x4540, 0x4541, 0x4544, 0x4545, 0x4550, 0x4551, 0x4554, 0x4555,
	0x5000, 0x5001, 0x5004, 0x5005, 0x5010, 0x5011, 0x5014, 0x5015,
	0x5040, 0x5041, 0x5044, 0x5045, 0x5050, 0x5051, 0x5054, 0x5055,
	0x5100, 0x5101, 0x5104, 0x5105, 0x5110, 0x5111, 0x5114, 0x5115,
	0x5140, 0x5141, 0x5144, 0x5145, 0x5150, 0x5151, 0x5154, 0x5155,
	0x5400, 0x5401, 0x5404, 0x5405, 0x5410, 0x5411, 0x5414, 0x5415,
	0x5440, 0x5441, 0x5444, 0x5445, 0x5450, 0x5451, 0x5454, 0x5455,
	0x5500, 0x5501, 0x5504, 0x5505, 0x5510, 0x5511, 0x5514, 0x5515,
	0x5540, 0x5541, 0x5544, 0x5545, 0x5550, 0x5551, 0x5554, 0x5555,
}

var morton256by1 []uint16
var morton256by16 []uint16
var morton256by17 []uint16

func buildMorton256() []uint16 {
	m := make([]uint16, 256)
	i := 0
	for _, a := range []uint16{0x0000, 0x1000, 0x4000, 0x5000} {
		for _, b := range []uint16{0x000, 0x100, 0x400, 0x500} {
			for _, c := range []uint16{0x00, 0x10, 0x40, 0x50} {
				for _, d := range []uint16{0x0, 0x1, 0x4, 0x5} {
					m[i] = a + b + c + d
					i++
				}
			}
		}
	}
	return m
}

// Since we are starting with 8-bit data, we only need to do 1 lookup table
// worth of work
func interleave2DMorton(x, y uint8) uint16 {
	return morton256[y]<<1 | morton256[x]
}

func interleave2DMorton2(x, y uint8) uint16 {
	return (morton256by1[y] | morton256[x])
}

// very unsafe
var mP unsafe.Pointer
var m1P unsafe.Pointer

func interleave2DMorton2Unsafe(x, y uint8) uint16 {
	return (*(*uint16)(unsafe.Pointer(uintptr(m1P) + uintptr(y)*2))) |
		(*(*uint16)(unsafe.Pointer(uintptr(mP) + uintptr(x)*2)))
}

const (
	magicMult1 = uint64(0x0101010101010101)
	magicMask  = uint64(0x8040201008040201)
	magicMult2 = uint64(0x0102040810204081)
)

func interleave2DMultiply64(x, y uint8) uint16 {
	return uint16(
		(((((uint64(x) * magicMult1) & magicMask) * magicMult2) >> 49) & 0x5555) |
			(((((uint64(y) * magicMult1) & magicMask) * magicMult2) >> 48) & 0xAAAA))
}

const (
	// in the C reference, it is doing 2 uint16 into a uint32, and uses arrays,
	// but we can't declare a const array in Go, and we don't need to iterate
	// it anyway
	B0  = uint32(0x55555555)
	B1  = uint32(0x33333333)
	B2  = uint32(0x0F0F0F0F)
	B3  = uint32(0x00FF00FF)
	B0b = uint16(0x5555)
	B1b = uint16(0x3333)
	B2b = uint16(0x0F0F)
	B3b = uint16(0x00FF)
	S0  = 1
	S1  = 2
	S2  = 4
	S3  = 8
)

func interleave2DMagic32(x, y uint8) uint16 {
	xx := uint32(x)
	yy := uint32(y)

	// Because our input space is only 8 bits we don't have to shift-by-8
	//xx = (xx | (xx << S3)) & B3
	xx = (xx | (xx << S2)) & B2
	xx = (xx | (xx << S1)) & B1
	xx = (xx | (xx << S0)) & B0

	//yy = (yy | (yy << S3)) & B3
	yy = (yy | (yy << S2)) & B2
	yy = (yy | (yy << S1)) & B1
	yy = (yy | (yy << S0)) & B0

	return uint16(xx | (yy << 1))
}

func interleave2DMagic16(x, y uint8) uint16 {
	xx := uint16(x)
	yy := uint16(y)

	// xx = (xx | (xx << S3)) & B3b
	xx = (xx | (xx << S2)) & B2b
	xx = (xx | (xx << S1)) & B1b
	xx = (xx | (xx << S0)) & B0b

	// yy = (yy | (yy << S3)) & B3b
	yy = (yy | (yy << S2)) & B2b
	yy = (yy | (yy << S1)) & B1b
	yy = (yy | (yy << S0)) & B0b

	return uint16(xx | (yy << 1))
}

type interleave2DTest struct {
	x, y        uint8
	interleaved uint16
}

var interleave2DTests = []interleave2DTest{
	// Check some obvious interleavings, and check each bit gets mapped to the
	// correct final bit
	{x: 0x00, y: 0x00, interleaved: 0x0000},
	{x: 0x01, y: 0x01, interleaved: 0x0003},
	{x: 0x03, y: 0x00, interleaved: 0x0005},
	{x: 0x03, y: 0x01, interleaved: 0x0007},
	{x: 0x03, y: 0x03, interleaved: 0x000f},
	{x: 0xff, y: 0xff, interleaved: 0xffff},
	{x: 0x01, y: 0x00, interleaved: 0x0001},
	{x: 0x02, y: 0x00, interleaved: 0x0004},
	{x: 0x04, y: 0x00, interleaved: 0x0010},
	{x: 0x08, y: 0x00, interleaved: 0x0040},
	{x: 0x10, y: 0x00, interleaved: 0x0100},
	{x: 0x20, y: 0x00, interleaved: 0x0400},
	{x: 0x40, y: 0x00, interleaved: 0x1000},
	{x: 0x80, y: 0x00, interleaved: 0x4000},
	{x: 0x00, y: 0x01, interleaved: 0x0002},
	{x: 0x00, y: 0x02, interleaved: 0x0008},
	{x: 0x00, y: 0x04, interleaved: 0x0020},
	{x: 0x00, y: 0x08, interleaved: 0x0080},
	{x: 0x00, y: 0x10, interleaved: 0x0200},
	{x: 0x00, y: 0x20, interleaved: 0x0800},
	{x: 0x00, y: 0x40, interleaved: 0x2000},
	{x: 0x00, y: 0x80, interleaved: 0x8000},
}

func (*Interleave2DSuite) TestInterleave2DObvious(c *check.C) {
	for _, vals := range interleave2DTests {
		interleaved := interleave2DObvious(vals.x, vals.y)
		c.Check(interleaved,
			check.Equals, vals.interleaved,
			check.Commentf("expected %x %x to become %x not %x",
				vals.x, vals.y, vals.interleaved, interleaved))
	}
}

func funcName(f interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
}

func checkInterleave2DMatchesObvious(c *check.C, f func(x, y uint8) uint16) {
	for _, vals := range interleave2DTests {
		interleaved := f(vals.x, vals.y)
		c.Check(interleaved,
			check.Equals, vals.interleaved,
			check.Commentf("expected %v(0x%x,0x%x) = 0x%x not 0x%x",
				funcName(f),
				vals.x, vals.y,
				vals.interleaved, interleaved))
	}
	for y := uint8(0); y < 0xFF; y++ {
		for x := uint8(0); x < 0xFF; x++ {
			c.Assert(f(x, y), check.Equals, interleave2DObvious(x, y))
		}
		c.Assert(f(0xFF, y), check.Equals, interleave2DObvious(0xFF, y))
	}
	c.Assert(f(0xFF, 0xFF),
		check.Equals, interleave2DObvious(0xFF, 0xFF))
}

func (*Interleave2DSuite) TestInterleave2DMorton(c *check.C) {
	checkInterleave2DMatchesObvious(c, interleave2DMorton)
}

func (*Interleave2DSuite) TestInterleave2DMorton2(c *check.C) {
	checkInterleave2DMatchesObvious(c, interleave2DMorton2)
}

func (*Interleave2DSuite) TestInterleave2DMorton2Unsafe(c *check.C) {
	checkInterleave2DMatchesObvious(c, interleave2DMorton2Unsafe)
}

func (*Interleave2DSuite) TestInterleave2DMultiply64(c *check.C) {
	checkInterleave2DMatchesObvious(c, interleave2DMultiply64)
}

func (*Interleave2DSuite) TestInterleave2DMagic32(c *check.C) {
	checkInterleave2DMatchesObvious(c, interleave2DMagic32)
}

func (*Interleave2DSuite) TestInterleave2DMagic16(c *check.C) {
	checkInterleave2DMatchesObvious(c, interleave2DMagic16)
}

func benchInterleave2D(c *check.C, f func(x, y uint8) uint16) {
	for i := 0; i < c.N; i++ {
		for y := uint8(0); y < 255; y++ {
			for x := uint8(0); x < 255; x++ {
				f(x, y)
			}
			f(0xFF, y)
		}
		f(0xFF, 0xFF)
	}
}

func (*Interleave2DSuite) BenchmarkInterleave2DObvious(c *check.C) {
	benchInterleave2D(c, interleave2DObvious)
}

func (*Interleave2DSuite) BenchmarkInterleave2DMorton(c *check.C) {
	benchInterleave2D(c, interleave2DMorton)
}

func (*Interleave2DSuite) BenchmarkInterleave2DMorton2(c *check.C) {
	benchInterleave2D(c, interleave2DMorton2)
}

func (*Interleave2DSuite) BenchmarkInterleave2DMorton2Unsafe(c *check.C) {
	benchInterleave2D(c, interleave2DMorton2Unsafe)
}

func (*Interleave2DSuite) BenchmarkInterleave2DMultiply64(c *check.C) {
	benchInterleave2D(c, interleave2DMultiply64)
}

func (*Interleave2DSuite) BenchmarkInterleave2DMagic32(c *check.C) {
	benchInterleave2D(c, interleave2DMagic32)
}

func (*Interleave2DSuite) BenchmarkInterleave2DMagic16(c *check.C) {
	benchInterleave2D(c, interleave2DMagic16)
}

func (*Interleave2DSuite) BenchmarkInterleave2DNoOp(c *check.C) {
	// This gives a baseline for just what it costs to make 65536 function
	// calls
	benchInterleave2D(c, func(x, y uint8) uint16 { return 0 })
}

func init() {
	morton256by1 = make([]uint16, 256)
	morton256by16 = make([]uint16, 256)
	morton256by17 = make([]uint16, 256)
	for idx, m := range morton256 {
		morton256by1[idx] = m << 1
		morton256by16[idx] = m << 16
		morton256by17[idx] = m << 17
	}
	mP = unsafe.Pointer(&morton256[0])
	m1P = unsafe.Pointer(&morton256by1[0])
}
