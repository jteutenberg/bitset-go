package bitset

import (
	"fmt"
	"math"
	"math/bits"
)

const (
	Bit     uint64 = 1
	AllBits uint64 = 0xFFFFFFFFFFFFFFFF
)

type IntSet struct {
	minValue uint
	maxValue uint

	// then a bitset if this is not just an interval between min and max
	vs      []uint64 // nil for intervals
	vsStart uint     //value offset, a multiple of 64

	count uint //size is no greater than this
}

func NewIntSet() *IntSet {
	set := IntSet{minValue: math.MaxUint, maxValue: 0, vs: nil, vsStart: 0, count: 0}
	return &set
}

func NewIntSetCapacity(capacity int) *IntSet {
	set := IntSet{minValue: math.MaxUint, maxValue: 0, vs: make([]uint64, capacity/64+1), vsStart: 0, count: 0}
	return &set
}

func NewIntSetFromInts(values []int) *IntSet {
	var max int
	for _, v := range values {
		if v > max {
			max = v
		}
	}
	set := IntSet{minValue: math.MaxUint, maxValue: 0, vs: make([]uint64, max/64+1), vsStart: 0, count: 0}
	s := &set
	s.AddInts(values)
	return s
}

func NewIntSetFromUInts(values []uint) *IntSet {
	var max uint
	for _, v := range values {
		if v > max {
			max = v
		}
	}

	set := IntSet{minValue: math.MaxUint, maxValue: 0, vs: make([]uint64, max/64+1), vsStart: 0, count: 0}
	s := &set
	for _, v := range values {
		s.Add(v)
	}
	return s
}

func NewIntSetFromInterval(min, max uint) *IntSet {
	set := IntSet{minValue: min, maxValue: max, vs: nil, vsStart: 0, count: max - min + 1}
	return &set
}

func (set *IntSet) AddInts(values []int) {
	for _, v := range values {
		set.Add(uint(v))
	}
}

func (set *IntSet) intersectMinMax(other *IntSet) (uint, uint) {
	minV := set.minValue
	maxV := set.maxValue
	if minV < other.minValue {
		minV = other.minValue
	}
	if maxV > other.maxValue {
		maxV = other.maxValue
	}
	return minV, maxV
}

func (set *IntSet) unionMinMax(other *IntSet) (uint, uint) {
	minV := set.minValue
	maxV := set.maxValue
	if minV > other.minValue {
		minV = other.minValue
	}
	if maxV < other.maxValue {
		maxV = other.maxValue
	}
	return minV, maxV
}

func (set *IntSet) Contains(x uint) bool {
	if x < set.minValue || x > set.maxValue {
		return false
	}
	if set.vs == nil {
		return true
	}
	index := (x - set.vsStart) >> 6
	subIndex := x & 0x3F
	return (set.vs[index] & (Bit << subIndex)) != 0
}

func (set *IntSet) promoteToBitSet() {
	if set.vs != nil {
		return
	}
	// start from one uint before the min value, avoid underflow
	var lowestValue uint
	if set.minValue >= 64 {
		lowestValue = set.minValue - 64
	}
	// round to the beginning of the uint
	set.vsStart = ((lowestValue) >> 6) << 6
	start := (set.minValue - set.vsStart) >> 6
	end := (set.maxValue - set.vsStart) >> 6
	set.vs = make([]uint64, end-start+5)
	// set all values between start and end
	for i := start + 1; i < end; i++ {
		set.vs[i] = AllBits
	}
	// now handle the start and end
	startMask := AllBits << (set.minValue & 0x3F)
	endMask := AllBits >> (63 - (set.maxValue & 0x3F))
	if start == end {
		set.vs[start] = startMask & endMask
	} else {
		set.vs[start] = startMask
		set.vs[end] = endMask
	}
}

func (set *IntSet) Add(x uint) {
	// test for extending an interval
	if set.vs == nil {
		// test for adding to an empty interval
		if set.minValue > set.maxValue {
			set.minValue = x
			set.maxValue = x
			set.count = 1
			return
		}
		if x == set.minValue-1 {
			set.minValue = x
			set.count++
			return
		}
		if x == set.maxValue+1 {
			set.maxValue = x
			set.count++
			return
		}
		if x >= set.minValue && x <= set.maxValue {
			return // already in the interval
		}
		// otherwise, promote to a bitset and keep going
		set.promoteToBitSet()
	}

	// note: x - set.vsStart could be negative, handle later
	index := (x - set.vsStart) >> 6
	// generate the bit within a uint
	subIndex := x & 0x3F
	bit := Bit << subIndex

	if x < set.minValue {
		set.minValue = x
		if x < set.vsStart {
			//TODO: re-allocate the vs slice and update vsStart
			index = (x - set.vsStart) >> 6
		}
		set.vs[index] |= bit
		set.count++
		if x > set.maxValue {
			set.maxValue = x
		}
		return
	}

	if x > set.maxValue {
		set.maxValue = x
		// allocate more space at the end if necessary
		if int(index) >= len(set.vs) {
			newVs := make([]uint64, index+2)
			copy(newVs, set.vs)
			set.vs = newVs
		}
		// a fresh uint to write to
		set.vs[index] |= bit
		set.count++
		return
	}

	old := set.vs[index]
	if (old & bit) != 0 { //already exists
		return
	}
	set.vs[index] = old | bit
	set.count++
}

func (set *IntSet) Remove(x uint) {
	if x < set.minValue || x > set.maxValue {
		return
	}
	if set.vs == nil {
		if x == set.minValue {
			set.minValue++
			set.count--
		} else if x == set.maxValue {
			set.maxValue--
			set.count--
		} else {
			set.promoteToBitSet()
		}
		if set.minValue > set.maxValue {
			// empty set
			set.minValue = math.MaxUint
			set.maxValue = 0
			return
		}
	}

	index := (x - set.vsStart) >> 6
	subIndex := x & 0x3F
	bit := Bit << subIndex
	old := set.vs[index]
	if (old & bit) == 0 {
		return //nothing to remove
	}
	set.vs[index] ^= bit
	set.count--
}

func (set *IntSet) IsEmpty() bool {
	return set.minValue > set.maxValue
}

func (set *IntSet) Clear() {
	set.minValue = math.MaxUint
	set.maxValue = 0
	set.count = 0
	if set.vs == nil {
		return
	}
	start := (set.minValue - set.vsStart) >> 6
	end := (set.maxValue - set.vsStart) >> 6
	for ; start <= end; start++ {
		set.vs[start] = 0
	}
	//set.end = 0
	//set.start = uint(len(set.vs)) + 1
}

func (set *IntSet) GetFirstID() (bool, uint) {
	if set.IsEmpty() {
		return false, 0
	}
	return true, set.minValue
}

func (set *IntSet) CountIntersection(other *IntSet) uint {
	if other.vs == nil && set.vs != nil {
		// bit set : interval
		return other.CountIntersection(set)
	}
	minV, maxV := set.intersectMinMax(other)
	if minV > maxV {
		return 0
	}
	if set.vs == nil {
		// Interval : interval
		if other.vs == nil {
			return maxV - minV + 1
		}
		// Interval : bit set
		count := 0
		start := (minV - other.vsStart) >> 6
		end := (maxV - other.vsStart) >> 6
		for i := start + 1; i < end; i++ {
			count += bits.OnesCount64(other.vs[i])
		}
		// mask at either end
		startMask := AllBits << (minV & 0x3F)
		endMask := AllBits >> (63 - (maxV & 0x3F))
		if start == end {
			otherBits := startMask & other.vs[start] & endMask
			return uint(bits.OnesCount64(otherBits))
		}
		startBits := startMask & other.vs[start]
		endBits := endMask & other.vs[end]
		count += bits.OnesCount64(startBits) + bits.OnesCount64(endBits)
		return uint(count)
	}
	// 4.  bit set : bit set
	start := (minV - set.vsStart) >> 6
	end := (maxV - set.vsStart) >> 6
	offset := ((minV - other.vsStart) >> 6) - start
	count := 0
	for ; start <= end; start++ {
		count += bits.OnesCount64(set.vs[start] & other.vs[start+offset])
	}
	return uint(count)
}

func (set *IntSet) CountIntersectionTo(other *IntSet, maxCount int) uint {
	// TODO: speed this up repeating the loop from CountIntersection
	if set.vs == nil {
		return set.CountIntersection(other)
	}
	if other.vs == nil {
		return other.CountIntersection(set)
	}

	minV, maxV := set.intersectMinMax(other)
	start := (minV - set.vsStart) >> 6
	end := (maxV - set.vsStart) >> 6
	offset := ((minV - other.vsStart) >> 6) - start
	count := 0
	for ; start <= end && count < maxCount; start++ {
		count += bits.OnesCount64(set.vs[start] & other.vs[start-offset])
	}
	return uint(count)
}

/**
 * Remove values from set so that it only contains
 * values that are also in other
 **/
func (set *IntSet) Intersect(other *IntSet) {
	minV, maxV := set.intersectMinMax(other)
	if minV > maxV {
		if set.vs != nil {
			// TODO: clear the bits
		}
		set.minValue = math.MaxUint
		set.maxValue = 0
		set.count = 0
		return
	}
	if set.vs == nil {
		if other.vs == nil {
			set.minValue = minV
			set.maxValue = maxV
			set.count = maxV - minV + 1
			return
		}
		set.promoteToBitSet()
		// TODO: interval : bit set
	}
	// TODO: bit set: interval
	// set some to zero, then mask the edges
	// similar to what is done when promoting

	start := (minV - set.vsStart) >> 6
	end := (maxV - set.vsStart) >> 6
	offset := ((minV - other.vsStart) >> 6) - start
	// clear everything up to the new minimum
	oldStart := (set.minValue - set.vsStart) >> 6
	oldEnd := (set.maxValue - set.vsStart) >> 6
	for ; oldStart < start; oldStart++ {
		set.vs[oldStart] = 0
	}
	for ; oldEnd > end; oldEnd-- {
		set.vs[oldEnd] = 0
	}
	for i := start; i <= end; i++ {
		set.vs[i] &= other.vs[i+offset]
	}
}

func (set *IntSet) RemoveAll(other *IntSet) {
	minV, maxV := set.intersectMinMax(other)
	if minV > maxV {
		return // no intersection
	}
	if set.vs == nil {
		if other.vs == nil {
			// check for shrinking interval
			if other.maxValue <= set.maxValue && other.minValue <= set.minValue {
				set.minValue = other.maxValue + 1
				if set.minValue > set.maxValue {
					set.minValue = math.MaxUint
					set.maxValue = 0
					set.count = 0
					return
				}
				set.count = set.maxValue - set.minValue + 1
				return
			}
			if other.maxValue >= set.maxValue && other.minValue >= set.minValue {
				set.maxValue = other.minValue - 1
				if set.minValue > set.maxValue {
					set.minValue = math.MaxUint
					set.maxValue = 0
					set.count = 0
					return
				}
				set.count = set.maxValue - set.minValue + 1
				return
			}
			// must be split in two then, so promote and continue
			set.promoteToBitSet()
		}
	}
	if other.vs == nil {
		// TODO: remove the interval from the bit set
	}

	// find the intersecting indices
	start := (minV - set.vsStart) >> 6
	end := (maxV - set.vsStart) >> 6
	offset := ((minV - other.vsStart) >> 6) - start
	for i := start; i <= end; i++ {
		set.vs[i] &= (^other.vs[i+offset])
	}
	if minV == set.minValue {
		// TODO: update min
	}
	if maxV == set.maxValue {
		// TODO: update max
	}
}

func (set *IntSet) AddAll(other *IntSet) {
	if set.maxValue >= other.maxValue && set.minValue <= other.minValue {
		return
	}
	minV, maxV := set.unionMinMax(other)
	if minV > maxV {
		// both are empty
		return
	}
	if set.vs == nil {
		if maxV == set.maxValue && minV == set.minValue {
			// nothing added
			return
		}
		if other.vs == nil {
			set.maxValue = maxV
			set.minValue = minV
			set.count = maxV - minV + 1
			return
		}
		// otherwise, promote to a bitset and keep going
		set.promoteToBitSet()
	}
	// TODO: handle adding an interval to bit set
	// TODO: may need to reallocate
	// update vsStart if necessary
	start := (other.minValue - set.vsStart) >> 6
	end := (other.maxValue - set.vsStart) >> 6
	offset := ((other.minValue - other.vsStart) >> 6) - start
	if end >= uint(len(set.vs)) {
		newVs := make([]uint64, end+2)
		copy(newVs, set.vs)
		set.vs = newVs
	}
	for i := start; i <= end; i++ {
		set.vs[i] |= other.vs[i+offset]
	}
	set.maxValue = maxV
	set.minValue = minV
}

func (set *IntSet) Union(other *IntSet) *IntSet {
	// TODO
	return nil
}

func (set *IntSet) Intersection(other *IntSet) *IntSet {
	// TODO
	return nil
}

func (set *IntSet) GetNextID(x uint) (bool, uint) {
	x++
	if x > set.maxValue {
		return false, 0
	}
	if x < set.minValue {
		// skip to the first value
		x = set.minValue
	}
	if set.vs == nil {
		return true, x
	}
	//check for any bit later at this uint
	index := (x - set.vsStart) >> 6
	subIndex := x & 0x3F
	filter := AllBits << subIndex
	if (filter & set.vs[index]) == 0 {
		index++
		subIndex = 0
	}
	// then skip any all-zero indices
	end := (set.maxValue - set.vsStart) >> 6
	for index <= end && set.vs[index] == 0 {
		index++
	}
	if index > end {
		return false, 0
	}
	v := set.vs[index] >> subIndex
	zs := uint(bits.TrailingZeros64(v))
	return true, (index << 6) + subIndex + zs
}

func (set *IntSet) AsInts() []int {
	//consider inlining the loop to speed this up
	ids := make([]int, 0, set.count)
	for ok, id := set.GetFirstID(); ok; ok, id = set.GetNextID(id) {
		ids = append(ids, int(id))
	}
	return ids
}
func (set *IntSet) AsUints() []uint {
	ids := make([]uint, 0, set.count)
	for ok, id := set.GetFirstID(); ok; ok, id = set.GetNextID(id) {
		ids = append(ids, id)
	}
	return ids
}

func (set *IntSet) CountMembers() uint {
	if set.vs == nil {
		return set.count
	}
	count := 0
	start := (set.minValue - set.vsStart) >> 6
	end := (set.minValue - set.vsStart) >> 6
	for i := start; i <= end; i++ {
		count += bits.OnesCount64(set.vs[i])
	}
	set.count = uint(count)
	return set.count
}

// Size gets an upper bound on the size of this set. After a call to CountMembers this value
// is accurate until elements are removed.
func (set *IntSet) Size() uint {
	return set.count
}

func (set *IntSet) String() string {
	s := "{"
	first := true
	for ok, v := set.GetFirstID(); ok; ok, v = set.GetNextID(v) {
		if first {
			first = false
			s = fmt.Sprint(s, v)
		} else {
			s = fmt.Sprint(s, ",", v)
		}
	}
	return s + "}"
}
