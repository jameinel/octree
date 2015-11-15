Basic Octree implementation that uses Morton indexing to arrange the octree
blocks.

This was designed with RGB colors in mind, though it would be pretty easy to
extend to other types. You could change the parameters to 10-bit shorts in a
uint32 or change the inner index to 64-bits to get 21-bit uint32s.
