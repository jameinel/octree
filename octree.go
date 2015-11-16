package octree

import (
	"fmt"
)

// Track the counts of everything in an 8-way structure.  The deeper 'depth' is
// set, the more memory is consumed, but the finer grained the counting is.
// This also maps the keys to a concrete object at the lowest layer
type Octree struct {
	count uint32
	// Each layer has 8^n count fields
	layerCounts [][]uint32
	// The last layer maps to a sparse slice of values.
	values [][]*value
}

type value struct {
	r, g, b uint8
	count   uint32
}

func NewOctree(depth int) (*Octree, error) {
	if depth < 1 || depth > 7 {
		return nil, fmt.Errorf("Invalid octree depth: %d", depth)
	}
	layers := make([][]uint32, depth-1)
	size := 1
	for i := range layers {
		size *= 8
		layers[i] = make([]uint32, size)
	}
	values := make([][]*value, size)
	return &Octree{
		layerCounts: layers,
		values:      values,
	}, nil
}

func (o *Octree) Add(r, g, b uint8) {
	o.count++
	index := interleaveRGB(r, g, b)
	for depth, counts := range o.layerCounts {
		layerIndex := (index >> (uint(21 - depth*3)))
		counts[layerIndex]++
	}
	vi := index >> uint(24-len(o.layerCounts)*3)
	// See if we can find this exact value, if not, add it
	// TODO: We could keep the valueSlice in some sort of sorted order, so
	// 	 that we could do faster searching. However, it is easier to
	// 	 just make the octree another depth deeper.
	valueSlice := o.values[vi]
	found := false
	for _, v := range valueSlice {
		if r == v.r && g == v.g && b == v.b {
			v.count++
			found = true
			break
		}
	}
	if !found {
		v := &value{r: r, g: g, b: b, count: 1}
		o.values[vi] = append(valueSlice, v)
	}
}

// The distance^2 to a given value
func dist2ToV(r, g, b uint8, v *value) uint32 {
	d := uint32(v.r) - uint32(r)
	d *= d
	dist2 := d
	d = uint32(v.g) - uint32(g)
	d *= d
	dist2 += d
	d = uint32(v.b) - uint32(b)
	d *= d
	dist2 += d
	return dist2
}

// Given an index into the values slice, figure out what the min and max values
// for that block would be. This is a inclusive boundary [min, max] (max and
// min are inside the block)
func (o *Octree) findBlockMinMax(bindex uint32) (vMin, vMax value) {
	blockShift := uint(24 - len(o.layerCounts)*3)
	index := bindex << blockShift
	rMin, gMin, bMin := interleavedToRGB(index)
	stride := uint8(0xFF) >> uint(len(o.layerCounts))
	vMin = value{r: rMin, g: gMin, b: bMin}
	vMax = value{r: rMin + stride, g: gMin + stride, b: bMin + stride}
	return vMin, vMax
}

func (o *Octree) findMinDist2ToBoundary(r, g, b uint8, vMin, vMax value) uint32 {
	dist2 := uint32(r) - uint32(vMin.r)
	dist2 *= dist2
	minDist2 := dist2
	dist2 = uint32(g) - uint32(vMin.g)
	dist2 *= dist2
	if dist2 < minDist2 {
		minDist2 = dist2
	}
	dist2 = uint32(b) - uint32(vMin.b)
	dist2 *= dist2
	if dist2 < minDist2 {
		minDist2 = dist2
	}
	dist2 = uint32(vMax.r) - uint32(r)
	dist2 *= dist2
	if dist2 < minDist2 {
		minDist2 = dist2
	}
	dist2 = uint32(vMax.g) - uint32(g)
	dist2 *= dist2
	if dist2 < minDist2 {
		minDist2 = dist2
	}
	dist2 = uint32(vMax.b) - uint32(b)
	dist2 *= dist2
	if dist2 < minDist2 {
		minDist2 = dist2
	}
	return minDist2
}

func (o *Octree) FindClosest(r, g, b uint8) value {
	index := interleaveRGB(r, g, b)
	viShift := uint(24 - len(o.layerCounts)*3)
	vi := index >> viShift
	valueSlice := o.values[vi]
	// Pass through looking for an exact match
	for _, v := range valueSlice {
		// Nothing will ever be closer than an exact match
		if r == v.r && g == v.g && b == v.b {
			return *v
		}
	}
	// Now look at everything in this block, looking for something close
	closest := (*value)(nil)
	closestDist2 := uint32(0xFFFFFFFF)
	for _, v := range valueSlice {
		dist2 := dist2ToV(r, g, b, v)
		if closest == nil || dist2 < closestDist2 {
			closestDist2 = dist2
			closest = v
		}
	}
	if closest != nil {
		// We found something in this block, but we might be close
		// enough to an edge that the next block holds things that are
		// actually closer, check where our boundary ends
		vMin, vMax := o.findBlockMinMax(vi)
		minDist2 := o.findMinDist2ToBoundary(r, g, b, vMin, vMax)
		if closestDist2 < minDist2 {
			// nothing outside of this block could be closer than
			// what we found, so we're safe to return it
			return *closest
		}
	}
	// TODO: We should start by checking the 26-neighbors of this block,
	// and then possibly expand to bigger and bigger regions, rather than
	// going straight to brute force.
	// No exact match, start with brute-force search
	for _, values := range o.values {
		for _, v := range values {
			dist2 := dist2ToV(r, g, b, v)
			if closest == nil || dist2 < closestDist2 {
				closestDist2 = dist2
				closest = v
			}
		}
	}
	if closest == nil {
		return value{}
	}
	return *closest
}

// Get a 'neighbor' one less and one greater the value, but cap it at [0,max]
func getBoundedNeighbor(v, max uint8) (uint8, uint8) {
	vMin := v
	if v > 0 {
		vMin = v - 1
	}
	vMax := v
	if v < max {
		vMax = v + 1
	}
	return vMin, vMax
}

// Find all of the blocks that are next to this one.
func (o *Octree) find26NeighborBlocks(bindex uint32) []uint32 {
	// Technically, this is only the 'high order' r g b bits shifted by
	// layer, but it works for finding the correct neighbor indexes
	r, g, b := interleavedToRGB(bindex)
	max := uint8(0x01) << uint(len(o.layerCounts)-1)
	rMin, rMax := getBoundedNeighbor(r, max)
	gMin, gMax := getBoundedNeighbor(g, max)
	bMin, bMax := getBoundedNeighbor(b, max)
	neighbors := make([]uint32, 0, 26)
	// Note: we don't have to worry about overflowing uint8 because
	// len(layerCounts) is always at least 1
	// TODO: We walk in r,g,b order, but the blocks in memory are stored in
	// morton order, for memory purposes, wouldn't it be better to use morton
	// ordering for the blocks?
	for rr := rMin; rr <= rMax; rr++ {
		for gg := gMin; gg <= gMax; gg++ {
			for bb := bMin; bb <= bMax; bb++ {
				if rr == r && gg == g && bb == b {
					continue
				}
				idx := interleaveRGB(rr, gg, bb)
				neighbors = append(neighbors, idx)
			}
		}
	}
	return neighbors
}

// Grab all of the values in the 26 neighbors of this block.
// The 26 neighbors is the 3x3x3 grid excluding the block itself.
// This also knows that it can ignore going past 0 or above 255.
// This also returns the distance to the closest boundary for which there might
// be more points (so if you are at r=0x01, we don't return the distance to 0,
// because there can't be any points on the other side.)
func (o *Octree) find26NeighborValues(bindex uint32) ([]*value, uint32) {
	return nil, 0
}

// This is a mapping from 0-256 uint8 into a spread bits format, where each bit
// in the input gets spread out into the output. (eg 0011 => 000 000 001 001)
// The table itself comes from
// http://www.forceflow.be/2013/10/07/morton-encodingdecoding-through-bit-interleaving-implementations/
// Though it can be built from scratch using a simple split-by-3
// implementation.
var morton256_3D = []uint32{
	0x00000000,
	0x00000001, 0x00000008, 0x00000009, 0x00000040, 0x00000041, 0x00000048, 0x00000049, 0x00000200,
	0x00000201, 0x00000208, 0x00000209, 0x00000240, 0x00000241, 0x00000248, 0x00000249, 0x00001000,
	0x00001001, 0x00001008, 0x00001009, 0x00001040, 0x00001041, 0x00001048, 0x00001049, 0x00001200,
	0x00001201, 0x00001208, 0x00001209, 0x00001240, 0x00001241, 0x00001248, 0x00001249, 0x00008000,
	0x00008001, 0x00008008, 0x00008009, 0x00008040, 0x00008041, 0x00008048, 0x00008049, 0x00008200,
	0x00008201, 0x00008208, 0x00008209, 0x00008240, 0x00008241, 0x00008248, 0x00008249, 0x00009000,
	0x00009001, 0x00009008, 0x00009009, 0x00009040, 0x00009041, 0x00009048, 0x00009049, 0x00009200,
	0x00009201, 0x00009208, 0x00009209, 0x00009240, 0x00009241, 0x00009248, 0x00009249, 0x00040000,
	0x00040001, 0x00040008, 0x00040009, 0x00040040, 0x00040041, 0x00040048, 0x00040049, 0x00040200,
	0x00040201, 0x00040208, 0x00040209, 0x00040240, 0x00040241, 0x00040248, 0x00040249, 0x00041000,
	0x00041001, 0x00041008, 0x00041009, 0x00041040, 0x00041041, 0x00041048, 0x00041049, 0x00041200,
	0x00041201, 0x00041208, 0x00041209, 0x00041240, 0x00041241, 0x00041248, 0x00041249, 0x00048000,
	0x00048001, 0x00048008, 0x00048009, 0x00048040, 0x00048041, 0x00048048, 0x00048049, 0x00048200,
	0x00048201, 0x00048208, 0x00048209, 0x00048240, 0x00048241, 0x00048248, 0x00048249, 0x00049000,
	0x00049001, 0x00049008, 0x00049009, 0x00049040, 0x00049041, 0x00049048, 0x00049049, 0x00049200,
	0x00049201, 0x00049208, 0x00049209, 0x00049240, 0x00049241, 0x00049248, 0x00049249, 0x00200000,
	0x00200001, 0x00200008, 0x00200009, 0x00200040, 0x00200041, 0x00200048, 0x00200049, 0x00200200,
	0x00200201, 0x00200208, 0x00200209, 0x00200240, 0x00200241, 0x00200248, 0x00200249, 0x00201000,
	0x00201001, 0x00201008, 0x00201009, 0x00201040, 0x00201041, 0x00201048, 0x00201049, 0x00201200,
	0x00201201, 0x00201208, 0x00201209, 0x00201240, 0x00201241, 0x00201248, 0x00201249, 0x00208000,
	0x00208001, 0x00208008, 0x00208009, 0x00208040, 0x00208041, 0x00208048, 0x00208049, 0x00208200,
	0x00208201, 0x00208208, 0x00208209, 0x00208240, 0x00208241, 0x00208248, 0x00208249, 0x00209000,
	0x00209001, 0x00209008, 0x00209009, 0x00209040, 0x00209041, 0x00209048, 0x00209049, 0x00209200,
	0x00209201, 0x00209208, 0x00209209, 0x00209240, 0x00209241, 0x00209248, 0x00209249, 0x00240000,
	0x00240001, 0x00240008, 0x00240009, 0x00240040, 0x00240041, 0x00240048, 0x00240049, 0x00240200,
	0x00240201, 0x00240208, 0x00240209, 0x00240240, 0x00240241, 0x00240248, 0x00240249, 0x00241000,
	0x00241001, 0x00241008, 0x00241009, 0x00241040, 0x00241041, 0x00241048, 0x00241049, 0x00241200,
	0x00241201, 0x00241208, 0x00241209, 0x00241240, 0x00241241, 0x00241248, 0x00241249, 0x00248000,
	0x00248001, 0x00248008, 0x00248009, 0x00248040, 0x00248041, 0x00248048, 0x00248049, 0x00248200,
	0x00248201, 0x00248208, 0x00248209, 0x00248240, 0x00248241, 0x00248248, 0x00248249, 0x00249000,
	0x00249001, 0x00249008, 0x00249009, 0x00249040, 0x00249041, 0x00249048, 0x00249049, 0x00249200,
	0x00249201, 0x00249208, 0x00249209, 0x00249240, 0x00249241, 0x00249248, 0x00249249,
}

// See references on Morton encoding and mapping 3 integers into 1 integer with
// the bits intermixed
func interleaveRGB(r, g, b uint8) uint32 {
	return morton256_3D[b] + morton256_3D[g]<<1 + morton256_3D[r]<<2
}

// This inverts the effect of interleaveRGB.
// Note that this is not performance tuned like interleave was. A lot could
// probably be done here to operate on more than 1 bit at a time
func interleavedToRGB(index uint32) (r, g, b uint8) {
	r = 0
	g = 0
	b = 0
	for bit := uint(0); bit < 8; bit++ {
		b |= uint8(index&0x000001) << bit
		index >>= 1
		g |= uint8(index&0x000001) << bit
		index >>= 1
		r |= uint8(index&0x000001) << bit
		index >>= 1
	}
	return
}
