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

	cardinalityInvalidated bool
	cardinality            uint
}

func NewIntSet() *IntSet {
	set := IntSet{minValue: math.MaxUint, maxValue: 0, vs: nil, vsStart: 0, cardinalityInvalidated: false, cardinality: 0}
	return &set
}

func NewIntSetCapacity(capacity int) *IntSet {
	set := IntSet{minValue: math.MaxUint, maxValue: 0, vs: make([]uint64, capacity/64+1), vsStart: 0, cardinalityInvalidated: false, cardinality: 0}
	return &set
}

func NewIntSetFromInts(values []int) *IntSet {
	var max int
	for _, v := range values {
		if v > max {
			max = v
		}
	}
	set := IntSet{minValue: math.MaxUint, maxValue: 0, vs: make([]uint64, max/64+1), vsStart: 0, cardinalityInvalidated: false, cardinality: 0}
	s := &set
	for _, v := range values {
		s.Add(uint(v))
	}
	return s
}

func NewIntSetFromUInts(values []uint) *IntSet {
	var max uint
	for _, v := range values {
		if v > max {
			max = v
		}
	}

	set := IntSet{minValue: math.MaxUint, maxValue: 0, vs: make([]uint64, max/64+1), vsStart: 0, cardinalityInvalidated: false, cardinality: 0}
	s := &set
	for _, v := range values {
		s.Add(v)
	}
	return s
}

func NewIntSetFromInterval(min, max uint) *IntSet {
	set := IntSet{minValue: min, maxValue: max, vs: nil, vsStart: 0, cardinalityInvalidated: false, cardinality: max - min + 1}
	return &set
}

func (set *IntSet) Clone() *IntSet {
	if set.vs == nil {
		return NewIntSetFromInterval(set.minValue, set.maxValue)
	}
	clone := IntSet{minValue: set.minValue, maxValue: set.maxValue, vs: make([]uint64, len(set.vs)), vsStart: set.vsStart, cardinalityInvalidated: set.cardinalityInvalidated, cardinality: set.cardinality}
	copy(clone.vs, set.vs)
	return &clone
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

func (set *IntSet) Add(x uint) *IntSet {
	// test for extending an interval
	if set.vs == nil {
		// test for adding to an empty interval
		if set.IsEmpty() {
			set.minValue = x
			set.maxValue = x
			set.cardinality = 1
			return set
		}
		if x == set.minValue-1 {
			set.minValue = x
			set.cardinality++
			return set
		}
		if x == set.maxValue+1 {
			set.maxValue = x
			set.cardinality++
			return set
		}
		if x >= set.minValue && x <= set.maxValue {
			return set // already in the interval
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
		set.cardinality++
		if x > set.maxValue {
			set.maxValue = x
		}
		return set
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
		set.cardinality++
		return set
	}

	old := set.vs[index]
	if (old & bit) != 0 { //already exists
		return set
	}
	set.vs[index] = old | bit
	set.cardinality++
	return set
}

func (set *IntSet) Remove(x uint) *IntSet {
	if x < set.minValue || x > set.maxValue {
		return set
	}
	if set.vs == nil {
		needPromote := true
		if x == set.minValue {
			set.minValue++
			set.cardinality--
			needPromote = false
		} else if x == set.maxValue {
			set.maxValue--
			set.cardinality--
			needPromote = false
		}
		if set.IsEmpty() {
			set.minValue = math.MaxUint
			set.maxValue = 0
			return set
		}
		if needPromote {
			set.promoteToBitSet()
		} else {
			return set
		}
	}

	index := (x - set.vsStart) >> 6
	subIndex := x & 0x3F
	bit := Bit << subIndex
	old := set.vs[index]
	if (old & bit) == 0 {
		return set //nothing to remove
	}
	set.vs[index] ^= bit
	set.cardinality--
	return set
}

func (set *IntSet) IsSubsetOf(other *IntSet) bool {
	return set.CountIntersection(other) == set.Size()
}

func (set *IntSet) IsDisjointFrom(other *IntSet) bool {
	return set.CountIntersectionTo(other, 1) == 0
}

func (set *IntSet) IsEmpty() bool {
	return set.minValue > set.maxValue
}

func (set *IntSet) Clear() *IntSet {
	set.minValue = math.MaxUint
	set.maxValue = 0
	set.cardinality = 0
	set.cardinalityInvalidated = false
	if set.vs == nil {
		return set
	}
	start := (set.minValue - set.vsStart) >> 6
	end := (set.maxValue - set.vsStart) >> 6
	for ; start <= end; start++ {
		set.vs[start] = 0
	}
	return set
}

func (set *IntSet) GetFirstValue() (bool, uint) {
	if set.IsEmpty() {
		return false, 0
	}
	return true, set.minValue
}

func (set *IntSet) GetLastValue() (bool, uint) {
	if set.IsEmpty() {
		return false, 0
	}
	return true, set.maxValue
}

func (set *IntSet) GetNextValue(x uint) (bool, uint) {
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
	return true, (index << 6) + subIndex + zs + set.vsStart
}

func (set *IntSet) GetPrevValue(x uint) (bool, uint) {
	x--
	if x < set.minValue {
		return false, 0
	}
	if x > set.maxValue {
		x = set.maxValue
	}
	if set.vs == nil {
		return true, x
	}
	//check for any bit earlier at this uint
	index := (x - set.vsStart) >> 6
	subIndex := x & 0x3F
	filter := ^(AllBits << (subIndex + 1))
	if (filter & set.vs[index]) == 0 {
		index--
		subIndex = 63
	}
	// then skip any all-zero indices
	end := (set.minValue - set.vsStart) >> 6
	for index > end && set.vs[index] == 0 {
		index--
	}
	// take care with avoiding underflow
	if set.vs[index] == 0 {
		if index == end {
			return false, 0
		}
		index--
	}
	if index < end {
		return false, 0
	}
	// shuffle the bit in question to the left end
	v := set.vs[index] << (63 - subIndex)
	// any remaining zeros need to be removed from the value too
	zs := uint(bits.LeadingZeros64(v))
	return true, (index << 6) + subIndex - zs + set.vsStart
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
	otherStart := ((minV - other.vsStart) >> 6)
	count := 0
	for i := start; i <= end; i++ {
		count += bits.OnesCount64(set.vs[i] & other.vs[i-start+otherStart])
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
	otherStart := ((minV - other.vsStart) >> 6)
	count := 0
	for i := start; i <= end && count < maxCount; i++ {
		count += bits.OnesCount64(set.vs[i] & other.vs[i-start+otherStart])
	}
	return uint(count)
}

/**
 * Remove values from set so that it only contains
 * values that are also in other
 **/
func (set *IntSet) Intersection(other *IntSet) *IntSet {
	minV, maxV := set.intersectMinMax(other)
	if minV > maxV {
		if set.vs != nil {
			// clear all bits
			start := (set.minValue - set.vsStart) >> 6
			end := (set.maxValue - set.vsStart) >> 6
			for i := start; i <= end; i++ {
				set.vs[i] = 0
			}
			set.cardinality = 0
			set.cardinalityInvalidated = false
			return set
		}
		set.minValue = math.MaxUint
		set.maxValue = 0
		set.cardinality = 0
		return set
	}
	if set.vs == nil {
		if other.vs == nil {
			set.minValue = minV
			set.maxValue = maxV
			set.cardinality = maxV - minV + 1
			return set
		}
		// shrink to the shared interval first
		set.minValue = minV
		set.maxValue = maxV
		set.promoteToBitSet()
	}

	start := (minV - set.vsStart) >> 6
	end := (maxV - set.vsStart) >> 6
	otherStart := ((minV - other.vsStart) >> 6)
	// clear everything up to the new minimum
	oldStart := (set.minValue - set.vsStart) >> 6
	oldEnd := (set.maxValue - set.vsStart) >> 6
	for ; oldStart < start; oldStart++ {
		set.vs[oldStart] = 0
	}
	// and down to the new maximum
	for ; oldEnd > end; oldEnd-- {
		set.vs[oldEnd] = 0
	}
	if other.vs == nil {
		// only have to handle the two edges
		startMask := AllBits << (minV & 0x3F)
		endMask := AllBits >> (63 - (maxV & 0x3F))
		if start == end {
			set.vs[start] = set.vs[start] & startMask & endMask
		} else {
			set.vs[start] = set.vs[start] & startMask
			set.vs[end] = set.vs[end] & endMask
		}
		set.cardinality = 0
		set.cardinalityInvalidated = false
	} else {
		// bit set : bit set intersection
		for i := start; i <= end; i++ {
			set.vs[i] &= other.vs[i-start+otherStart]
		}
		// update min and max values
		_, set.minValue = set.GetNextValue(minV - 1)
		// TODO: update max value. Make a GetPrevID
		set.cardinality = 0 // unknown
		set.cardinalityInvalidated = true
	}
	return set
}

// The union less the intersection
func (set *IntSet) SymmetricDifference(other *IntSet) *IntSet {
	if set.minValue > other.maxValue || set.maxValue < other.minValue {
		// no intersection, so return the union
		return set.Union(other)
	}
	intersection := set.Clone().Intersection(other)
	return set.Union(other).Difference(intersection)
}

func (set *IntSet) Difference(other *IntSet) *IntSet {
	minV, maxV := set.intersectMinMax(other)
	if minV > maxV {
		return set // no intersection
	}
	if set.vs == nil {
		if other.vs == nil {
			// check for shrinking interval
			if other.maxValue <= set.maxValue && other.minValue <= set.minValue {
				set.minValue = other.maxValue + 1
				if set.IsEmpty() {
					set.minValue = math.MaxUint
					set.maxValue = 0
					set.cardinality = 0
					return set
				}
				set.cardinality = set.maxValue - set.minValue + 1
				return set
			}
			if other.maxValue >= set.maxValue && other.minValue >= set.minValue {
				set.maxValue = other.minValue - 1
				if set.IsEmpty() {
					set.minValue = math.MaxUint
					set.maxValue = 0
					set.cardinality = 0
					return set
				}
				set.cardinality = set.maxValue - set.minValue + 1
				return set
			}
			// must be split in two then, so promote and continue
			set.promoteToBitSet()
		}
	}
	if other.vs == nil {
		// remove the interval from the bit set
		start := (minV - set.vsStart) >> 6
		end := (maxV - set.vsStart) >> 6

		if start == (other.minValue-set.vsStart)>>6 {
			// remove from the first uint, masked. A small min value removes more
			set.vs[start] = set.vs[start] & (AllBits >> (63 - (other.minValue & 0x3F)))
			start++
		}
		if end == (other.maxValue-set.vsStart)>>6 {
			// remove from the last uint
			set.vs[end] = set.vs[end] & (AllBits << (other.maxValue & 0x3F))
			end--
		}
		for i := start; i <= end; i++ {
			set.vs[i] = 0
		}
		set.cardinalityInvalidated = true
		return set
	}

	// find the intersecting indices
	start := (minV - set.vsStart) >> 6
	end := (maxV - set.vsStart) >> 6
	otherStart := ((minV - other.vsStart) >> 6)
	for i := start; i <= end; i++ {
		set.vs[i] &= (^other.vs[i-start+otherStart])
	}
	if minV == set.minValue {
		// TODO: update min
	}
	if maxV == set.maxValue {
		// TODO: update max
	}
	return set
}

func (set *IntSet) Union(other *IntSet) *IntSet {
	minV, maxV := set.unionMinMax(other)
	if minV > maxV {
		// both are empty
		return set
	}
	if set.vs == nil {
		if maxV == set.maxValue && minV == set.minValue {
			// nothing added
			return set
		}
		if other.vs == nil {
			set.maxValue = maxV
			set.minValue = minV
			set.cardinality = maxV - minV + 1
			return set
		}
		// otherwise, promote to a bitset and keep going
		set.promoteToBitSet()
	}
	if minV < set.vsStart {
		// reallocate the vs slice and update vsStart
		newStart := (minV >> 6)
		if newStart > 0 {
			newStart--
		}
		newEnd := (maxV >> 6) + 5
		newVs := make([]uint64, newEnd-newStart+1)
		copy(newVs[(set.vsStart>>6)-newStart:], set.vs)
		set.vs = newVs
		set.vsStart = newStart << 6
	}
	if maxV > set.maxValue {
		// possibly reallocate the vs slice and update maxValue
		newEnd := (maxV - set.vsStart) >> 6
		if newEnd >= uint(len(set.vs)) {
			newVs := make([]uint64, newEnd+5)
			copy(newVs, set.vs)
			set.vs = newVs
		}
	}
	if other.vs == nil {
		// add the interval to the bit set
		start := (other.minValue - set.vsStart) >> 6
		end := (other.maxValue - set.vsStart) >> 6
		startMask := AllBits << (other.minValue & 0x3F)
		endMask := AllBits >> (63 - (other.maxValue & 0x3F))
		if start == end {
			set.vs[start] |= startMask & endMask
		} else {
			set.vs[start] |= startMask
			set.vs[end] |= endMask
		}
		for i := start + 1; i < end; i++ {
			set.vs[i] |= AllBits
		}
		return set
	}
	start := (other.minValue - set.vsStart) >> 6
	end := (other.maxValue - set.vsStart) >> 6
	otherStart := ((other.minValue - other.vsStart) >> 6)
	if end >= uint(len(set.vs)) {
		newVs := make([]uint64, end+2)
		copy(newVs, set.vs)
		set.vs = newVs
	}
	for i := start; i <= end; i++ {
		set.vs[i] |= other.vs[i-start+otherStart]
	}
	set.maxValue = maxV
	set.minValue = minV
	return set
}

func (set *IntSet) AsInts() []int {
	//consider inlining the loop to speed this up
	ids := make([]int, 0, set.Size())
	for ok, id := set.GetFirstValue(); ok; ok, id = set.GetNextValue(id) {
		ids = append(ids, int(id))
	}
	return ids
}
func (set *IntSet) AsUints() []uint {
	ids := make([]uint, 0, set.Size())
	for ok, id := set.GetFirstValue(); ok; ok, id = set.GetNextValue(id) {
		ids = append(ids, id)
	}
	return ids
}

func (set *IntSet) countMembers() uint {
	if set.vs == nil {
		return set.cardinality
	}
	count := 0
	start := (set.minValue - set.vsStart) >> 6
	end := (set.maxValue - set.vsStart) >> 6
	for i := start; i <= end; i++ {
		count += bits.OnesCount64(set.vs[i])
	}
	set.cardinality = uint(count)
	set.cardinalityInvalidated = false
	return set.cardinality
}

// Size gets an upper bound on the size of this set. After a call to CountMembers this value
// is accurate until elements are removed.
func (set *IntSet) Size() uint {
	if set.cardinalityInvalidated {
		set.countMembers()
	}
	return set.cardinality
}

func (set *IntSet) String() string {
	// any empty set
	if set.IsEmpty() {
		return "{}"
	}
	s := "{"
	// longer intervals, print as a range
	if set.vs == nil && set.maxValue-set.minValue > 10 {
		return fmt.Sprint(s, set.minValue, "..", set.maxValue, "}")
	}
	first := true
	count := 0
	for ok, v := set.GetFirstValue(); ok; ok, v = set.GetNextValue(v) {
		if count > 20 {
			s = fmt.Sprint(s, "...", set.maxValue)
			break
		}
		if first {
			first = false
			s = fmt.Sprint(s, v)
		} else {
			s = fmt.Sprint(s, ",", v)
		}
	}
	return s + "}"
}
