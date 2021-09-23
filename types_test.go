package types

import (
	"math"
	"reflect"
	"testing"
)

type recursive struct {
	Inner []*recursive
}

func newRecursive(depth int) recursive {
	var r recursive
	inner := &r.Inner
	for i := 0; i < depth-1; i++ {
		*inner = []*recursive{new(recursive)}
		inner = &((*inner)[0].Inner)
	}
	*inner = []*recursive{&r}
	return r
}

type recursive2 struct {
	Inner *recursive2
}

func newRecursive2(depth int) recursive2 {
	var r recursive2
	inner := &r.Inner
	for i := 0; i < depth-1; i++ {
		*inner = new(recursive2)
		inner = &((*inner).Inner)
	}
	*inner = &r
	return r
}

func TestLessEqual(t *testing.T) {
	for _, test := range []struct {
		l     interface{}
		r     interface{}
		less  bool
		equal bool
	}{
		// Primitives.
		{bool(false), bool(true), true, false},
		{bool(true), bool(false), false, false},
		{bool(false), bool(false), false, true},

		{int(0), int(1), true, false},
		{int(2), int(1), false, false},
		{int(2), int(2), false, true},

		{int8(0), int8(1), true, false},
		{int8(2), int8(1), false, false},
		{int8(2), int8(2), false, true},

		{int16(0), int16(1), true, false},
		{int16(2), int16(1), false, false},
		{int16(2), int16(2), false, true},

		{int32(0), int32(1), true, false},
		{int32(2), int32(1), false, false},
		{int32(2), int32(2), false, true},

		{int64(0), int64(1), true, false},
		{int64(2), int64(1), false, false},
		{int64(2), int64(2), false, true},

		{uint(0), uint(1), true, false},
		{uint(2), uint(1), false, false},
		{uint(2), uint(2), false, true},

		{uint8(0), uint8(1), true, false},
		{uint8(2), uint8(1), false, false},
		{uint8(2), uint8(2), false, true},

		{uint16(0), uint16(1), true, false},
		{uint16(2), uint16(1), false, false},
		{uint16(2), uint16(2), false, true},

		{uint32(0), uint32(1), true, false},
		{uint32(2), uint32(1), false, false},
		{uint32(2), uint32(2), false, true},

		{uint64(0), uint64(1), true, false},
		{uint64(2), uint64(1), false, false},
		{uint64(2), uint64(2), false, true},

		{uintptr(0), uintptr(1), true, false},
		{uintptr(2), uintptr(1), false, false},
		{uintptr(2), uintptr(2), false, true},

		{float32(0), float32(1), true, false},
		{float32(2), float32(1), false, false},
		{float32(2), float32(2), false, true},
		{float32(math.NaN()), float32(2), true, false},
		{float32(2), float32(math.NaN()), false, false},
		{float32(math.NaN()), float32(math.NaN()), false, true},
		{float32(math.Inf(-1)), float32(2), true, false},
		{float32(2), float32(math.Inf(-1)), false, false},
		{float32(math.Inf(-1)), float32(math.Inf(-1)), false, true},

		{float64(0), float64(1), true, false},
		{float64(2), float64(1), false, false},
		{float64(2), float64(2), false, true},
		{float64(math.NaN()), float64(2), true, false},
		{float64(2), float64(math.NaN()), false, false},
		{float64(math.NaN()), float64(math.NaN()), false, true},
		{float64(math.Inf(-1)), float64(2), true, false},
		{float64(2), float64(math.Inf(-1)), false, false},
		{float64(math.Inf(-1)), float64(math.Inf(-1)), false, true},

		// Complex.

		// Real takes priority.
		{complex64(complex(0, 9)), complex64(complex(1, 0)), true, false},
		{complex64(complex(2, 9)), complex64(complex(1, 0)), false, false},
		{complex64(complex(2, 2)), complex64(complex(2, 2)), false, true},
		{complex64(complex(math.NaN(), 0)), complex64(complex(2, 0)), true, false},
		{complex64(complex(2, 0)), complex64(complex(math.NaN(), 0)), false, false},
		{complex64(complex(math.NaN(), 0)), complex64(complex(math.NaN(), 0)), false, true},
		{complex64(complex(math.Inf(-1), 9)), complex64(complex(2, 0)), true, false},
		{complex64(complex(2, 0)), complex64(complex(math.Inf(-1), 0)), false, false},
		{complex64(complex(math.Inf(-1), 0)), complex64(complex(math.Inf(-1), 0)), false, true},

		// Equal real compares imaginary.
		{complex64(complex(1, 0)), complex64(complex(1, 1)), true, false},
		{complex64(complex(1, 2)), complex64(complex(1, 1)), false, false},
		{complex64(complex(1, 2)), complex64(complex(1, 2)), false, true},
		{complex64(complex(1, math.NaN())), complex64(complex(1, 2)), true, false},
		{complex64(complex(1, 2)), complex64(complex(1, math.NaN())), false, false},
		{complex64(complex(1, math.NaN())), complex64(complex(1, math.NaN())), false, true},
		{complex64(complex(1, math.Inf(-1))), complex64(complex(1, 2)), true, false},
		{complex64(complex(1, 2)), complex64(complex(1, math.Inf(-1))), false, false},
		{complex64(complex(1, math.Inf(-1))), complex64(complex(1, math.Inf(-1))), false, true},

		// Real takes priority.
		{complex128(complex(0, 9)), complex128(complex(1, 0)), true, false},
		{complex128(complex(2, 9)), complex128(complex(1, 0)), false, false},
		{complex128(complex(2, 2)), complex128(complex(2, 2)), false, true},
		{complex128(complex(math.NaN(), 0)), complex128(complex(2, 0)), true, false},
		{complex128(complex(2, 0)), complex128(complex(math.NaN(), 0)), false, false},
		{complex128(complex(math.NaN(), 0)), complex128(complex(math.NaN(), 0)), false, true},
		{complex128(complex(math.Inf(-1), 9)), complex128(complex(2, 0)), true, false},
		{complex128(complex(2, 0)), complex128(complex(math.Inf(-1), 0)), false, false},
		{complex128(complex(math.Inf(-1), 0)), complex128(complex(math.Inf(-1), 0)), false, true},

		// Equal real compares imaginary.
		{complex128(complex(1, 0)), complex128(complex(1, 1)), true, false},
		{complex128(complex(1, 2)), complex128(complex(1, 1)), false, false},
		{complex128(complex(1, 2)), complex128(complex(1, 2)), false, true},
		{complex128(complex(1, math.NaN())), complex128(complex(1, 2)), true, false},
		{complex128(complex(1, 2)), complex128(complex(1, math.NaN())), false, false},
		{complex128(complex(1, math.NaN())), complex128(complex(1, math.NaN())), false, true},
		{complex128(complex(1, math.Inf(-1))), complex128(complex(1, 2)), true, false},
		{complex128(complex(1, 2)), complex128(complex(1, math.Inf(-1))), false, false},
		{complex128(complex(1, math.Inf(-1))), complex128(complex(1, math.Inf(-1))), false, true},

		// Slice & array.
		{[]int{}, []int{1}, true, false},
		{[]int{0}, []int{1}, true, false},
		{[]int{2}, []int{}, false, false},
		{[]int{2}, []int{0}, false, false},
		{[]int{}, []int{}, false, true},
		{[]int{1}, []int{1}, false, true},

		{[...]int{0}, [...]int{1}, true, false},
		{[...]int{2}, [...]int{0}, false, false},
		{[...]int{1}, [...]int{1}, false, true},
		{[...]int{}, [...]int{}, false, true},

		// Chan.
		{make(chan int), func() chan int { c := make(chan int, 1); c <- 1; return c }(), true, false}, // right has one elem
		{make(chan int), make(chan int), false, true},
		{func() chan int { c := make(chan int, 1); c <- 1; return c }(), func() chan int { c := make(chan int, 1); return c }(), false, false}, // left has one elem

		// Map.
		{
			l:     map[int]int{},
			r:     map[int]int{1: 1},
			less:  true,
			equal: false,
		},
		{
			l:     map[int]int{},
			r:     map[int]int{},
			less:  false,
			equal: true,
		},
		{
			l:     map[int]int{1: 1},
			r:     map[int]int{},
			less:  false,
			equal: false,
		},

		{
			l:     map[int]int{1: 1},
			r:     map[int]int{2: 1},
			less:  true,
			equal: false,
		},
		{
			l:     map[int]int{2: 2},
			r:     map[int]int{2: 2},
			less:  false,
			equal: true,
		},
		{
			l:     map[int]int{2: 2, 3: 3},
			r:     map[int]int{2: 2, 1: 1},
			less:  false,
			equal: false,
		},

		{
			l:     map[int]int{2: 2, 3: 1},
			r:     map[int]int{2: 2, 3: 3},
			less:  true,
			equal: false,
		},
		{
			l:     map[int]int{2: 2, 3: 3},
			r:     map[int]int{2: 2, 3: 3},
			less:  false,
			equal: true,
		},
		{
			l:     map[int]int{2: 2, 3: 4},
			r:     map[int]int{2: 2, 3: 3},
			less:  false,
			equal: false,
		},

		// Ptr / struct.
		{(*int)(nil), new(int), true, false},
		{(*int)(nil), (*int)(nil), false, true},
		{new(int), (*int)(nil), false, false},

		{newRecursive(1), newRecursive(2), true, false},
		{newRecursive(1), newRecursive(1), false, true},
		{newRecursive(2), newRecursive(1), false, false},

		{newRecursive2(1), newRecursive2(2), true, false},
		{newRecursive2(1), newRecursive2(1), false, true},
		{newRecursive2(2), newRecursive2(1), false, false},

		{&struct {
			F int
			G bool
			h int
		}{0, true, 9}, &struct {
			F int
			G bool
			h int
		}{1, true, 3}, true, false},
		{&struct {
			F int
			G bool
			h int
		}{1, true, 2}, &struct {
			F int
			G bool
			h int
		}{1, true, 2}, false, true},
		{&struct {
			F int
			G bool
			h int
		}{1, true, 1}, &struct {
			F int
			G bool
			h int
		}{1, false, 3}, false, false},

		// String.
		{"a", "b", true, false},
		{"b", "b", false, true},
		{"c", "b", false, false},

		//
	} {
		lt, eq := Less(test.l, test.r), Equal(test.l, test.r)
		if lt != test.less {
			t.Errorf("l %v r %v, got less? %v, exp less? %v", test.l, test.r, lt, test.less)
		}
		if eq != test.equal {
			t.Errorf("l %v r %v, got equal? %v, exp equal? %v", test.l, test.r, eq, test.equal)
		}

		cmp := Compare(test.l, test.r)
		if test.less && cmp != -1 {
			t.Errorf("l %v r %v, compare? %v, exp less? %v", test.l, test.r, cmp, test.less)
		}
		if test.equal && cmp != 0 {
			t.Errorf("l %v r %v, compare? %v, exp equal? %v", test.l, test.r, cmp, test.equal)
		}
		if (!test.less && !test.equal) && cmp != 1 {
			t.Errorf("l %v r %v, compare? %v, exp greater? %v", test.l, test.r, cmp, (!test.less && !test.equal))
		}
	}
}

func TestSort(t *testing.T) {
	for _, test := range []struct {
		in  interface{}
		exp interface{}
	}{
		{3, 3},
		{[...]int{2, 3, 4, 1}, [...]int{2, 3, 4, 1}}, // unaddressable, unsortable
		{[...]int{}, [...]int{}},
		{&[...]int{9, 3, 4, 1}, &[...]int{1, 3, 4, 9}},

		{[]bool{false, true, true, false}, []bool{false, false, true, true}},
		{[]int{2, 3, 4, 1}, []int{1, 2, 3, 4}},
		{[]int8{2, 3, 4, 1}, []int8{1, 2, 3, 4}},
		{[]int16{2, 3, 4, 1}, []int16{1, 2, 3, 4}},
		{[]int32{2, 3, 4, 1}, []int32{1, 2, 3, 4}},
		{[]int64{2, 3, 4, 1}, []int64{1, 2, 3, 4}},
		{[]uint{2, 3, 4, 1}, []uint{1, 2, 3, 4}},
		{[]uint8{2, 3, 4, 1}, []uint8{1, 2, 3, 4}},
		{[]uint16{2, 3, 4, 1}, []uint16{1, 2, 3, 4}},
		{[]uint32{2, 3, 4, 1}, []uint32{1, 2, 3, 4}},
		{[]uint64{2, 3, 4, 1}, []uint64{1, 2, 3, 4}},
		{[]uintptr{2, 3, 4, 1}, []uintptr{1, 2, 3, 4}},
		{[]float32{2, 3, 4, 1}, []float32{1, 2, 3, 4}},
		{[]float64{2, 3, 4, 1}, []float64{1, 2, 3, 4}},
		{[]string{"foo", "bar", "baz"}, []string{"bar", "baz", "foo"}},
		{[]*int{nil, nil}, []*int{nil, nil}},
		{(*int)(nil), (*int)(nil)},

		{
			[]struct {
				A int
				b int
			}{{3, 1}, {2, 2}, {1, 3}},
			[]struct {
				A int
				b int
			}{{1, 3}, {2, 2}, {3, 1}},
		},

		{
			struct {
				A []int
				b int
				B map[string][]int
				C map[string]string
			}{
				A: []int{3, 2, 1},
				B: map[string][]int{
					"foo": []int{5, 4, 3, 2},
					"bar": []int{2, 3, 4, 5},
				},
				C: map[string]string{
					"a": "a",
					"b": "b",
				},
			},
			struct {
				A []int
				b int // unexported, skipped
				B map[string][]int
				C map[string]string // unsortable values
			}{
				A: []int{1, 2, 3},
				B: map[string][]int{
					"foo": []int{2, 3, 4, 5},
					"bar": []int{2, 3, 4, 5},
				},
				C: map[string]string{
					"a": "a",
					"b": "b",
				},
			},
		},

		//
	} {
		Sort(test.in)
		if !reflect.DeepEqual(test.in, test.exp) {
			t.Errorf("got %v != exp %v", test.in, test.exp)
		}
	}
}

func TestDistinctInplaceInts(t *testing.T) {
	for _, test := range []struct {
		in  []int
		exp []int
	}{
		{
			nil,
			nil,
		},
		{
			[]int{1},
			[]int{1},
		},
		{
			[]int{1, 2, 3, 4, 5},
			[]int{1, 2, 3, 4, 5},
		},
		{
			[]int{5, 4, 3, 2, 1},
			[]int{1, 2, 3, 4, 5},
		},
		{
			[]int{1, 2, 2, 3, 4, 5},
			[]int{1, 2, 3, 4, 5},
		},
		{
			[]int{3, 2, 4, 5, 3, 4, 2, 5},
			[]int{2, 3, 4, 5},
		},
		{
			[]int{1, 2, 2, 2, 2, 2, 2, 3, 3, 3, 3, 4, 4, 4, 4, 4, 4},
			[]int{1, 2, 3, 4},
		},

		//
	} {
		DistinctInPlace(&test.in)
		if !reflect.DeepEqual(test.in, test.exp) {
			t.Errorf("got %v != exp %v", test.in, test.exp)
		}
	}
}

func TestDistinctInPlaceRecursive(t *testing.T) {
	in := []recursive{newRecursive(3), newRecursive(1), newRecursive(1), newRecursive(1), newRecursive(3)}
	exp := []recursive{newRecursive(3), newRecursive(1)}
	DistinctInPlace(&in)
	if !reflect.DeepEqual(in, exp) {
		// DeepEqual doesn't actually compare as good as we need, but
		// manual checking confirms what we expect.
		t.Errorf("got %v != exp %v", in, exp)
	}
	Sort(newRecursive(5))
	Sort(newRecursive2(5))
}
