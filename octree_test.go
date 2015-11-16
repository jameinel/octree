package octree

import (
	"testing"

	"gopkg.in/check.v1"
)

func TestAll(t *testing.T) {
	check.TestingT(t)
}

type OctTreeSuite struct{}

var _ = check.Suite(&OctTreeSuite{})

func (*OctTreeSuite) TestNewOctree(c *check.C) {
	oct, err := NewOctree(1)
	c.Assert(err, check.IsNil)
	c.Assert(oct, check.NotNil)
	c.Check(len(oct.layerCounts), check.Equals, 0)
	c.Check(oct.count, check.Equals, uint32(0))
	c.Check(len(oct.values), check.Equals, 1)
}

func (*OctTreeSuite) TestNewOctreeDepth2(c *check.C) {
	oct, err := NewOctree(2)
	c.Assert(err, check.IsNil)
	c.Assert(oct, check.NotNil)
	c.Check(oct.layerCounts, check.HasLen, 1)
	c.Check(oct.count, check.Equals, uint32(0))
	c.Check(oct.layerCounts[0], check.HasLen, 8)
	c.Check(len(oct.values), check.Equals, 8)
}

func (*OctTreeSuite) TestNewOctreeDepth3(c *check.C) {
	oct, err := NewOctree(3)
	c.Assert(err, check.IsNil)
	c.Assert(oct, check.NotNil)
	c.Check(oct.layerCounts, check.HasLen, 2)
	c.Check(oct.count, check.Equals, uint32(0))
	c.Check(oct.layerCounts[0], check.HasLen, 8)
	c.Check(oct.layerCounts[1], check.HasLen, 64)
	c.Check(len(oct.values), check.Equals, 64)
}

func (*OctTreeSuite) TestNewOctreeInvalid(c *check.C) {
	oct, err := NewOctree(0)
	c.Assert(err, check.ErrorMatches, "Invalid octree depth: 0")
	c.Assert(oct, check.IsNil)
	oct, err = NewOctree(10)
	c.Assert(err, check.ErrorMatches, "Invalid octree depth: 10")
	c.Assert(oct, check.IsNil)
}

func checkAdding(c *check.C, r, g, b uint8, l0block, l1block int) {
	oct, err := NewOctree(3)
	c.Assert(err, check.IsNil)
	c.Assert(oct, check.NotNil)
	c.Assert(oct.layerCounts, check.HasLen, 2)
	expLayer0 := make([]uint32, 8)
	expLayer0[l0block]++
	expLayer1 := make([]uint32, 64)
	expLayer1[l1block]++
	oct.Add(r, g, b)
	c.Check(oct.count, check.Equals, uint32(1))
	c.Check(oct.layerCounts[0], check.DeepEquals, expLayer0)
	c.Check(oct.layerCounts[1], check.DeepEquals, expLayer1)
	for i, blockValues := range oct.values {
		if i == l1block {
			v := &value{r: r, g: g, b: b, count: 1}
			c.Check(blockValues, check.DeepEquals, []*value{v})
		} else {
			c.Check(blockValues, check.DeepEquals, []*value(nil))
		}
	}
	c.Check(oct.layerCounts[1], check.DeepEquals, expLayer1)
}

func (*OctTreeSuite) TestAddingDepth3(c *check.C) {
	// r=g=b=0 must go into the very first bucket at all levels
	checkAdding(c, 0, 0, 0, 0, 0)
	// r=g=b = 0xff must go into the last slot
	checkAdding(c, 0xFF, 0xFF, 0xFF, 7, 63)
	// r=g=b=1 still fits in the first box
	checkAdding(c, 0x01, 0x01, 0x01, 0, 0)
	// b=0x80 is enough to move us by 1 box at the top layer, which is 8 boxes at the next layer
	checkAdding(c, 0x00, 0x00, 0x80, 1, 8)
	// g=0x80 moves us by 2 boxes at the top, and thus 16 at the second
	checkAdding(c, 0x00, 0x80, 0x00, 2, 16)
	// r=0x80 moves us by 4 boxes at the top, and thus 32 at the second
	checkAdding(c, 0x80, 0x00, 0x00, 4, 32)
	// r=g=b=0x80 moves us by 7 at the top, and 7*8=56 at the second
	checkAdding(c, 0x80, 0x80, 0x80, 7, 56)
	// r=g=b=0xC0 moves us by the same 7 at top, but by 7*8+4+2+1 at the second
	checkAdding(c, 0xC0, 0xC0, 0xC0, 7, 63)
}

func checkInterleaveAndBack(c *check.C, r, g, b uint8, woven uint32) {
	c.Check(interleaveRGB(r, g, b), check.Equals, woven)
	r2, g2, b2 := interleavedToRGB(woven)
	c.Check(r2, check.Equals, r)
	c.Check(g2, check.Equals, g)
	c.Check(b2, check.Equals, b)
}

func (*OctTreeSuite) TestInterleaveRGB(c *check.C) {
	checkInterleaveAndBack(c, 0x00, 0x00, 0x00, 0x000000)
	checkInterleaveAndBack(c, 0x01, 0x01, 0x01, 0x000007)
	checkInterleaveAndBack(c, 0xFF, 0xFF, 0xFF, 0xFFFFFF)
}

func (*OctTreeSuite) TestRepeatedAdds(c *check.C) {
	oct, err := NewOctree(3)
	c.Assert(err, check.IsNil)
	c.Assert(oct, check.NotNil)
	c.Assert(oct.layerCounts, check.HasLen, 2)
	oct.Add(0, 0, 0)
	oct.Add(0, 0, 0)
	oct.Add(0, 0, 0)
	expLayer0 := make([]uint32, 8)
	expLayer0[0] += 3
	expLayer1 := make([]uint32, 64)
	expLayer1[0] += 3
	c.Check(oct.count, check.Equals, uint32(3))
	c.Check(oct.layerCounts[0], check.DeepEquals, expLayer0)
	c.Check(oct.layerCounts[1], check.DeepEquals, expLayer1)
	for i, blockValues := range oct.values {
		if i == 0 {
			v := &value{r: 0, g: 0, b: 0, count: 3}
			c.Check(blockValues, check.DeepEquals, []*value{v})
		} else {
			c.Check(blockValues, check.DeepEquals, []*value(nil))
		}
	}
	c.Check(oct.layerCounts[1], check.DeepEquals, expLayer1)
}

func (*OctTreeSuite) TestAddNearby(c *check.C) {
	oct, err := NewOctree(3)
	c.Assert(err, check.IsNil)
	c.Assert(oct, check.NotNil)
	c.Assert(oct.layerCounts, check.HasLen, 2)
	oct.Add(0, 0, 0)
	oct.Add(0, 0, 1)
	oct.Add(0, 1, 0)
	oct.Add(1, 0, 0)
	expLayer0 := make([]uint32, 8)
	expLayer0[0] += 4
	expLayer1 := make([]uint32, 64)
	expLayer1[0] += 4
	c.Check(oct.count, check.Equals, uint32(4))
	c.Check(oct.layerCounts[0], check.DeepEquals, expLayer0)
	c.Check(oct.layerCounts[1], check.DeepEquals, expLayer1)
	for i, blockValues := range oct.values {
		if i == 0 {
			// TODO: We shouldn't depend on the sort order of this slice, but
			// for now, we have a deterministic ordering anyway
			exp := []*value{
				&value{r: 0, g: 0, b: 0, count: 1},
				&value{r: 0, g: 0, b: 1, count: 1},
				&value{r: 0, g: 1, b: 0, count: 1},
				&value{r: 1, g: 0, b: 0, count: 1},
			}
			c.Check(blockValues, check.DeepEquals, exp)
		} else {
			c.Check(blockValues, check.DeepEquals, []*value(nil))
		}
	}
	c.Check(oct.layerCounts[1], check.DeepEquals, expLayer1)
}

func (*OctTreeSuite) TestFindClosestExact(c *check.C) {
	oct, err := NewOctree(5)
	c.Assert(err, check.IsNil)
	c.Assert(oct, check.NotNil)
	oct.Add(0, 0, 0)
	oct.Add(0, 0, 1)
	oct.Add(0xFF, 0, 0)
	oct.Add(0, 0xFF, 0)
	oct.Add(0, 0, 0xFF)
	c.Check(oct.FindClosest(0, 0, 0), check.DeepEquals,
		value{r: 0, g: 0, b: 0, count: 1})
}

func (*OctTreeSuite) TestFindClosestNearby(c *check.C) {
	oct, err := NewOctree(5)
	c.Assert(err, check.IsNil)
	c.Assert(oct, check.NotNil)
	oct.Add(0, 0, 0)
	oct.Add(0xFF, 0, 0)
	oct.Add(0, 0xFF, 0)
	oct.Add(0, 0, 0xFF)
	c.Check(oct.FindClosest(0, 0, 1), check.DeepEquals,
		value{r: 0, g: 0, b: 0, count: 1})
}

func (*OctTreeSuite) TestFindClosestWithDistraction(c *check.C) {
	// We need to put 1 value in the same octree block, but the actual
	// 'closest' value is in the octree just before this one.
	oct, err := NewOctree(3)
	c.Assert(err, check.IsNil)
	c.Assert(oct, check.NotNil)
	// We create one just on the boundary of its block
	oct.Add(0x40, 0x00, 0x00)
	index := interleaveRGB(0x40, 0x00, 0x00)
	c.Check(index, check.Equals, uint32(0x100000))
	// the r=0x40 ends up in the 4th block
	c.Check(index>>18, check.Equals, uint32(4))
	c.Check(oct.values[4], check.DeepEquals,
		[]*value{
			&value{r: 0x40, g: 0x00, b: 0x00, count: 1},
		})
	// We add another one that is in the first block, but will actually be
	// farther than our search location.
	oct.Add(0x00, 0x00, 0x00)
	index = interleaveRGB(0x00, 0x00, 0x00)
	c.Check(index, check.Equals, uint32(0x000000))
	c.Check(index>>18, check.Equals, uint32(0))
	// the r=0x40 ends up in the 4th block
	c.Check(oct.values[0], check.DeepEquals,
		[]*value{
			&value{r: 0x00, g: 0x00, b: 0x00, count: 1},
		})
	c.Check(oct.values[4], check.DeepEquals,
		[]*value{
			&value{r: 0x40, g: 0x00, b: 0x00, count: 1},
		})
	// Now we search for the very edge of the first block, which should
	// find the item in the other block.
	c.Check(oct.FindClosest(0x39, 0, 0), check.DeepEquals,
		value{r: 0x40, g: 0, b: 0, count: 1})
}

func (*OctTreeSuite) TestFindClosestNextOctree(c *check.C) {
	oct, err := NewOctree(5)
	c.Assert(err, check.IsNil)
	c.Assert(oct, check.NotNil)
	oct.Add(0xFF, 0, 0)
	oct.Add(0, 0xFF, 0)
	oct.Add(0, 0, 0xFF)
	c.Check(oct.FindClosest(0xE0, 0, 0), check.DeepEquals,
		value{r: 0xFF, g: 0, b: 0, count: 1})
}

func (*OctTreeSuite) TestFindClosestEmptyOctree(c *check.C) {
	oct, err := NewOctree(5)
	c.Assert(err, check.IsNil)
	c.Assert(oct, check.NotNil)
	c.Check(oct.FindClosest(0xE0, 0, 0), check.DeepEquals,
		value{r: 0, g: 0, b: 0, count: 0})
}

func checkMinMax(c *check.C, oct *Octree, index uint32,
	rMin, gMin, bMin, rMax, gMax, bMax uint8) {
	vMin, vMax := oct.findBlockMinMax(index)
	c.Check(vMin, check.DeepEquals, value{r: rMin, g: gMin, b: bMin})
	c.Check(vMax, check.DeepEquals, value{r: rMax, g: gMax, b: bMax})
}

func (*OctTreeSuite) TestFindBlockMinMax2Layer(c *check.C) {
	oct, err := NewOctree(2)
	c.Assert(err, check.IsNil)
	c.Check(len(oct.values), check.Equals, 8)
	checkMinMax(c, oct, 0, 0x00, 0x00, 0x00, 0x7F, 0x7F, 0x7F)
	checkMinMax(c, oct, 1, 0x00, 0x00, 0x80, 0x7F, 0x7F, 0xFF)
	checkMinMax(c, oct, 2, 0x00, 0x80, 0x00, 0x7F, 0xFF, 0x7F)
	checkMinMax(c, oct, 3, 0x00, 0x80, 0x80, 0x7F, 0xFF, 0xFF)
	checkMinMax(c, oct, 4, 0x80, 0x00, 0x00, 0xFF, 0x7F, 0x7F)
	checkMinMax(c, oct, 5, 0x80, 0x00, 0x80, 0xFF, 0x7F, 0xFF)
	checkMinMax(c, oct, 6, 0x80, 0x80, 0x00, 0xFF, 0xFF, 0x7F)
	checkMinMax(c, oct, 7, 0x80, 0x80, 0x80, 0xFF, 0xFF, 0xFF)
}

func (*OctTreeSuite) TestFindBlockMinMax7Layer(c *check.C) {
	oct, err := NewOctree(6)
	c.Assert(err, check.IsNil)
	c.Assert(len(oct.values), check.Equals, 32768)
	checkMinMax(c, oct, 0, 0x00, 0x00, 0x00, 0x07, 0x07, 0x07)
	checkMinMax(c, oct, 1, 0x00, 0x00, 0x08, 0x07, 0x07, 0x0F)
	checkMinMax(c, oct, 2, 0x00, 0x08, 0x00, 0x07, 0x0F, 0x07)
	checkMinMax(c, oct, 3, 0x00, 0x08, 0x08, 0x07, 0x0F, 0x0F)
	checkMinMax(c, oct, 4, 0x08, 0x00, 0x00, 0x0F, 0x07, 0x07)
	checkMinMax(c, oct, 5, 0x08, 0x00, 0x08, 0x0F, 0x07, 0x0F)
	checkMinMax(c, oct, 6, 0x08, 0x08, 0x00, 0x0F, 0x0F, 0x07)
	checkMinMax(c, oct, 7, 0x08, 0x08, 0x08, 0x0F, 0x0F, 0x0F)
	checkMinMax(c, oct, 32767, 0xF8, 0xF8, 0xF8, 0xFF, 0xFF, 0xFF)
}
