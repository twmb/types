// Package types contains helper functions for arbitrary types (deep less,
// deep equal, deep sort).
package types

import (
	"fmt"
	"math"
	"reflect"
	"sort"
	"unsafe"
)

// Less returns whether l is deeply less than r. The input values must have the
// same type, or this will panic.
//
// Structs are deeply less if each field in order is either less or equal to
// the comparison field. If every field is equal, this returns false. Only
// public fields are compared.
//
// Nil pointers are less than non-nil pointers. Recursive types compare less
// following all other rules, or if they recurse sooner.
//
// Slices are less if they are shorter, or if each element in order is less
// than or equal to the other. If all elements are equal and the sizes are
// equal, this returns false.
//
// Maps are less if they are shorter. If they are of equal size, all keys are
// treated as a slice and they are compared following the same logic used for
// slices. If all keys are equal, then all values are compared following
// similar slice logic.
//
// Floats are less if they are NaN, or using a simple comparison.
//
// Complexes are less if their real is less. If the real is equal, then
// complexes are less if their imaginary is less.
//
// Chans are less if they have fewer buffered elements.
//
// Functions, interfaces, and unsafe pointers are never less than each other.
func Less(l, r interface{}) bool {
	lt, _ := lteq(newPointers(), reflect.ValueOf(l), reflect.ValueOf(r))
	return lt
}

// Equal returns whether l is deeply equal to r.The input values must have the
// same type, or this will panic.
//
// Structs are deeply equal if each field is equal. Unlike reflect, this
// function compares only public fields.
//
// Nil pointers are equal to nil pointers. Recursive types are equal following
// all other rules, or if they recurse at the same time.
//
// Slices are equal if they have the same length and each element is deeply
// equal.
//
// Maps are equal if they have the same length and all keys and values are
// deeply equal.
//
// Floats are equal if they are both NaN, or if they are both not NaN and
// are equal.
//
// Complexes are equal if their real is equal and their imaginary is equal.
//
// Chans are equal if they have the same amount of buffered elements.
//
// Functions, interfaces, and unsafe pointers equal if their pointers are
// equal.
func Equal(l, r interface{}) bool {
	_, eq := lteq(newPointers(), reflect.ValueOf(l), reflect.ValueOf(r))
	return eq
}

// Compare returns whether l is less than, equal to, or larger than r,
// following the same rules as Less and Equal.
func Compare(l, r interface{}) int {
	lt, eq := lteq(newPointers(), reflect.ValueOf(l), reflect.ValueOf(r))
	if lt {
		return -1
	} else if eq {
		return 0
	}
	return 1
}

type pointers map[unsafe.Pointer]struct{}

func newPointers() *pointers {
	var p pointers
	return &p
}

func (p *pointers) hasOrAdd(ptr unsafe.Pointer) bool {
	if *p == nil {
		*p = make(map[unsafe.Pointer]struct{})
	}
	_, has := (*p)[ptr]
	if !has {
		(*p)[ptr] = struct{}{}
	}
	return has
}

func (p pointers) remove(ptr unsafe.Pointer) {
	delete(p, ptr)
}

func lteq(p *pointers, lv, rv reflect.Value) (lt, eq bool) {
	t := lv.Type()
	if t != rv.Type() {
		panic("unequal types")
	}

	if k := t.Kind(); k != reflect.Struct {
		return lteqKind(p, k, lv, rv)
	}

	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if sf.PkgPath != "" {
			continue
		}

		lt, eq := lteqKind(p, sf.Type.Kind(), lv.Field(i), rv.Field(i))
		if !eq {
			return lt, false
		}
	}

	return false, true
}

func lteqKind(p *pointers, k reflect.Kind, lv, rv reflect.Value) (lt, eq bool) {
	switch k {
	case reflect.Bool:
		l, r := lv.Bool(), rv.Bool()
		return !l && r, l == r
	case reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64:
		return i64lt(lv.Int(), rv.Int())
	case reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Uintptr:
		return u64lt(lv.Uint(), rv.Uint())
	case reflect.Float32,
		reflect.Float64:
		return f64lt(lv.Float(), rv.Float())
	case reflect.Complex64,
		reflect.Complex128:
		return c128lt(lv.Complex(), rv.Complex())
	case reflect.Chan:
		ll, lr := lv.Len(), rv.Len()
		return ll < lr, ll == lr
	case reflect.Func,
		reflect.Interface,
		reflect.UnsafePointer:
		return false, lv.Interface() == rv.Interface()
	case reflect.String:
		l, r := lv.String(), rv.String()
		return l < r, l == r
	case reflect.Struct:
		return lteq(p, lv, rv)

	case reflect.Array,
		reflect.Slice:
		ll, lr := lv.Len(), rv.Len()
		lt, eq = ll < lr, ll == lr
		if eq {
			for i := 0; i < lr; i++ {
				lt, eq = lteq(p, lv.Index(i), rv.Index(i))
				if !eq {
					return lt, false
				}
			}
		}
		return lt, eq

	case reflect.Map:
		ll, lr := lv.Len(), rv.Len()
		lt, eq = ll < lr, ll == lr
		if eq {
			lkeys := lv.MapKeys()
			rkeys := rv.MapKeys()
			for _, keys := range &[...][]reflect.Value{
				lkeys,
				rkeys,
			} {
				sort.Slice(keys, func(i, j int) bool {
					lt, _ := lteq(p, keys[i], keys[j])
					return lt
				})
			}

			for i, lk := range lkeys {
				rk := rkeys[i]
				lt, eq = lteq(p, lk, rk)
				if !eq {
					return lt, false
				}
			}
			iter := lv.MapRange()
			for iter.Next() {
				lv := iter.Value()
				rv := rv.MapIndex(iter.Key())
				lt, eq = lteq(p, lv, rv)
				if !eq {
					return lt, false
				}
			}
		}
		return lt, eq

	case reflect.Ptr:
		if lv.IsNil() {
			return !rv.IsNil(), rv.IsNil()
		} else if rv.IsNil() {
			return false, false
		}

		lptr, rptr := unsafe.Pointer(lv.Pointer()), unsafe.Pointer(rv.Pointer())
		lhas, rhas := p.hasOrAdd(lptr), p.hasOrAdd(rptr)
		if !lhas {
			defer p.remove(lptr)
		}
		if !rhas {
			defer p.remove(rptr)
		}

		if lhas {
			return !rhas, rhas
		} else if rhas {
			return false, false
		}

		lv, rv = reflect.Indirect(lv), reflect.Indirect(rv)
		k = lv.Type().Kind()

		return lteqKind(p, k, lv, rv)

	default:
		return false, false // reflect.Invalid
	}
}

func i64lt(l, r int64) (lt, eq bool) {
	if l == r {
		return false, true
	}
	return l < r, false
}

func u64lt(l, r uint64) (lt, eq bool) {
	if l == r {
		return false, true
	}
	return l < r, false
}

func f64lt(l, r float64) (lt, eq bool) {
	if math.IsNaN(l) {
		rnan := math.IsNaN(r)
		return !rnan, rnan
	}
	if math.IsNaN(r) {
		return false, false
	}
	if math.IsInf(l, -1) {
		rinf := math.IsInf(r, -1)
		return !rinf, rinf
	}
	if math.IsInf(r, -1) {
		return false, false
	}
	return l < r, l == r
}

func c128lt(l, r complex128) (lt, eq bool) {
	lt, eq = f64lt(real(l), real(r))
	if eq {
		lt, eq = f64lt(imag(l), imag(r))
	}
	return lt, eq
}

// Sort deeply sorts any slice anywhere within s, traversing into maps, slices,
// and exported struct fields. Any non-primitive type is less than the other
// following the rules of Less.
//
// Note that this function performs value copies. This must not be used to sort
// types that are not safe to copy. For example, this must not sort
// []struct{sync.Mutex}, but it can sort []*struct{sync.Mutex}.
func Sort(s interface{}) {
	innerSort(newPointers(), reflect.ValueOf(s))
}

func setSlice(v reflect.Value, h *reflect.SliceHeader) {
	h.Data = uintptr(unsafe.Pointer(v.Pointer()))
	h.Len = v.Len()
	h.Cap = v.Len()
}

func innerSort(p *pointers, v reflect.Value) (sortable bool) {
	t := v.Type()
	switch v.Type().Kind() {
	case reflect.Ptr:
		if v.IsNil() {
			return true
		}
		ptr := unsafe.Pointer(v.Pointer())
		has := p.hasOrAdd(ptr)
		if has {
			return true
		}
		defer p.remove(ptr)
		return innerSort(p, reflect.Indirect(v))
	case reflect.Array:
		if v.Len() == 0 {
			return true
		}
		i0 := v.Index(0)
		if !i0.CanAddr() {
			return false
		}
		fallthrough

	case reflect.Slice:
		v = v.Slice(0, v.Len())
		switch t.Elem().Kind() {
		case reflect.Bool:
			var slice []bool
			setSlice(v, (*reflect.SliceHeader)(unsafe.Pointer(&slice)))
			sort.Slice(slice, func(i, j int) bool { return !slice[i] && slice[j] })
		case reflect.Int:
			var slice []int
			setSlice(v, (*reflect.SliceHeader)(unsafe.Pointer(&slice)))
			sort.Slice(slice, func(i, j int) bool { return slice[i] < slice[j] })
		case reflect.Int8:
			var slice []int8
			setSlice(v, (*reflect.SliceHeader)(unsafe.Pointer(&slice)))
			sort.Slice(slice, func(i, j int) bool { return slice[i] < slice[j] })
		case reflect.Int16:
			var slice []int16
			setSlice(v, (*reflect.SliceHeader)(unsafe.Pointer(&slice)))
			sort.Slice(slice, func(i, j int) bool { return slice[i] < slice[j] })
		case reflect.Int32:
			var slice []int32
			setSlice(v, (*reflect.SliceHeader)(unsafe.Pointer(&slice)))
			sort.Slice(slice, func(i, j int) bool { return slice[i] < slice[j] })
		case reflect.Int64:
			var slice []int64
			setSlice(v, (*reflect.SliceHeader)(unsafe.Pointer(&slice)))
			sort.Slice(slice, func(i, j int) bool { return slice[i] < slice[j] })
		case reflect.Uint:
			var slice []uint
			setSlice(v, (*reflect.SliceHeader)(unsafe.Pointer(&slice)))
			sort.Slice(slice, func(i, j int) bool { return slice[i] < slice[j] })
		case reflect.Uint8:
			var slice []uint8
			setSlice(v, (*reflect.SliceHeader)(unsafe.Pointer(&slice)))
			sort.Slice(slice, func(i, j int) bool { return slice[i] < slice[j] })
		case reflect.Uint16:
			var slice []uint16
			setSlice(v, (*reflect.SliceHeader)(unsafe.Pointer(&slice)))
			sort.Slice(slice, func(i, j int) bool { return slice[i] < slice[j] })
		case reflect.Uint32:
			var slice []uint32
			setSlice(v, (*reflect.SliceHeader)(unsafe.Pointer(&slice)))
			sort.Slice(slice, func(i, j int) bool { return slice[i] < slice[j] })
		case reflect.Uint64:
			var slice []uint64
			setSlice(v, (*reflect.SliceHeader)(unsafe.Pointer(&slice)))
			sort.Slice(slice, func(i, j int) bool { return slice[i] < slice[j] })
		case reflect.Uintptr:
			var slice []uintptr
			setSlice(v, (*reflect.SliceHeader)(unsafe.Pointer(&slice)))
			sort.Slice(slice, func(i, j int) bool { return slice[i] < slice[j] })
		case reflect.Float32:
			var slice []float32
			setSlice(v, (*reflect.SliceHeader)(unsafe.Pointer(&slice)))
			sort.Slice(slice, func(i, j int) bool { return slice[i] < slice[j] })
		case reflect.Float64:
			var slice []float64
			setSlice(v, (*reflect.SliceHeader)(unsafe.Pointer(&slice)))
			sort.Slice(slice, func(i, j int) bool { return slice[i] < slice[j] })
		case reflect.String:
			var slice []string
			setSlice(v, (*reflect.SliceHeader)(unsafe.Pointer(&slice)))
			sort.Slice(slice, func(i, j int) bool { return slice[i] < slice[j] })
		default:
			sort.Slice(v.Interface(), func(i, j int) bool { lt, _ := lteq(p, v.Index(i), v.Index(j)); return lt })
		}
	case reflect.Map:
		iter := v.MapRange()
		for iter.Next() {
			sortable = innerSort(p, iter.Value())
			if !sortable {
				break
			}
		}
		return sortable
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			sf := t.Field(i)
			if sf.PkgPath != "" {
				continue
			}
			innerSort(p, v.Field(i))
		}
	default:
		return false
	}
	return true
}

// DistinctInPlace accepts a *[]T, sorts it using the rules of Sort in this
// package, and compacts it in place using the rules of Equal in this package.
//
// This is similar to the slice generic version of sorting and compacting, but
// allows for even more types to be sorted.
//
// NOTE: When generics are released, this will change to be a func(*[]T). This
// will be a non-breaking change to anything that does not use this function
// for a variable of type func(interface{}).
func DistinctInPlace(sliceptr interface{}) {
	v := reflect.ValueOf(sliceptr)
	if v.Type().Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Slice {
		panic(fmt.Sprintf("DistinctInPlace: invalid non *[]T type %v", v.Type()))
	}
	v = v.Elem()
	p := newPointers()
	innerSort(p, v)
	if v.Len() == 0 {
		return
	}
	var last int
	lastv := v.Index(last)
	for next := 1; next < v.Len(); next++ {
		nextv := v.Index(next)
		if _, eq := lteq(p, lastv, nextv); eq {
			continue
		}
		last++
		lastv = v.Index(last)
		if last != next {
			lastv.Set(nextv)
		}
	}
	v.Set(v.Slice(0, last+1))
}
