// Package types contains helper functions for arbitrary types (deep less,
// deep equal, deep sort).
package types

import (
	"fmt"
	"math"
	"reflect"
	"sort"
)

// Less returns whether l is deeply less than r. The input values must have the
// same type, or this will panic.
//
// Structs are deeply less if each field in order is either less or equal to
// the comparison field. If every field is equal, this returns false. Only
// public fields are compared.
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
	lt, _ := lteq(reflect.ValueOf(l), reflect.ValueOf(r))
	return lt
}

// Equal returns whether l is deeply equal to r.The input values must have the
// same type, or this will panic.
//
// Structs are deeply equal if each field is equal. Unlike reflect, this
// function compares only public fields.
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
	_, eq := lteq(reflect.ValueOf(l), reflect.ValueOf(r))
	return eq
}

// Compare returns whether l is less than, equal to, or larger than r,
// following the same rules as Less and Equal.
func Compare(l, r interface{}) int {
	lt, eq := lteq(reflect.ValueOf(l), reflect.ValueOf(r))
	if lt {
		return -1
	} else if eq {
		return 0
	}
	return 1
}

func lteq(lv, rv reflect.Value) (lt, eq bool) {
	t := lv.Type()
	if t != rv.Type() {
		panic("unequal types")
	}

	if k := t.Kind(); k != reflect.Struct {
		return lteqKind(k, lv, rv)
	}

	for i := 0; i < t.NumField(); i++ {
		sf := t.Field(i)
		if !sf.IsExported() {
			continue
		}

		lt, eq := lteqKind(sf.Type.Kind(), lv.Field(i), rv.Field(i))
		if !eq {
			return lt, false
		}
	}

	return false, true
}

func lteqKind(k reflect.Kind, lv, rv reflect.Value) (lt, eq bool) {
start:
	l, r := lv.Interface(), rv.Interface()
	switch k {
	case reflect.Bool:
		l, r := l.(bool), r.(bool)
		return !l && r, l == r
	case reflect.Int:
		return i64lt(int64(l.(int)), int64(r.(int)))
	case reflect.Int8:
		return i64lt(int64(l.(int8)), int64(r.(int8)))
	case reflect.Int16:
		return i64lt(int64(l.(int16)), int64(r.(int16)))
	case reflect.Int32:
		return i64lt(int64(l.(int32)), int64(r.(int32)))
	case reflect.Int64:
		return i64lt(l.(int64), r.(int64))
	case reflect.Uint:
		return u64lt(uint64(l.(uint)), uint64(r.(uint)))
	case reflect.Uint8:
		return u64lt(uint64(l.(uint8)), uint64(r.(uint8)))
	case reflect.Uint16:
		return u64lt(uint64(l.(uint16)), uint64(r.(uint16)))
	case reflect.Uint32:
		return u64lt(uint64(l.(uint32)), uint64(r.(uint32)))
	case reflect.Uint64:
		return u64lt(l.(uint64), r.(uint64))
	case reflect.Uintptr:
		return u64lt(uint64(l.(uintptr)), uint64(r.(uintptr)))
	case reflect.Float32:
		return f64lt(float64(l.(float32)), float64(r.(float32)))
	case reflect.Float64:
		return f64lt(l.(float64), r.(float64))
	case reflect.Complex64:
		return c128lt(complex128(l.(complex64)), complex128(r.(complex64)))
	case reflect.Complex128:
		return c128lt(l.(complex128), r.(complex128))
	case reflect.Array, reflect.Slice:
		ll, lr := lv.Len(), rv.Len()
		lt, eq = ll < lr, ll == lr
		if eq {
			for i := 0; i < lr; i++ {
				lt, eq = lteq(lv.Index(i), rv.Index(i))
				if !eq {
					return lt, false
				}
			}
		}
		return lt, eq
	case reflect.Chan:
		ll, lr := lv.Len(), rv.Len()
		return ll < lr, ll == lr
	case reflect.Func, reflect.Interface, reflect.UnsafePointer:
		return false, l == r
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
					lt, _ := lteq(keys[i], keys[j])
					return lt
				})
			}

			for i, lk := range lkeys {
				rk := rkeys[i]
				lt, eq = lteq(lk, rk)
				if !eq {
					return lt, false
				}
			}
			iter := lv.MapRange()
			for iter.Next() {
				lv := iter.Value()
				rv := rv.MapIndex(iter.Key())
				lt, eq = lteq(lv, rv)
				if !eq {
					return lt, false
				}
			}
		}
		return lt, eq
	case reflect.Ptr:
		lv, rv = reflect.Indirect(lv), reflect.Indirect(rv)
		k = lv.Type().Kind()
		goto start
	case reflect.String:
		l, r := l.(string), r.(string)
		return l < r, l == r
	case reflect.Struct:
		return lteq(lv, rv)
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
	innerSort(reflect.ValueOf(s))
}

func innerSort(v reflect.Value) (sortable bool) {
start:
	t := v.Type()
	switch v.Type().Kind() {
	case reflect.Ptr:
		v = reflect.Indirect(v)
		goto start
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
		slice := v.Slice(0, v.Len()).Interface()
		switch t.Elem().Kind() {
		case reflect.Bool:
			slice := slice.([]bool)
			sort.Slice(slice, func(i, j int) bool { return !slice[i] && slice[j] })
		case reflect.Int:
			slice := slice.([]int)
			sort.Slice(slice, func(i, j int) bool { return slice[i] < slice[j] })
		case reflect.Int8:
			slice := slice.([]int8)
			sort.Slice(slice, func(i, j int) bool { return slice[i] < slice[j] })
		case reflect.Int16:
			slice := slice.([]int16)
			sort.Slice(slice, func(i, j int) bool { return slice[i] < slice[j] })
		case reflect.Int32:
			slice := slice.([]int32)
			sort.Slice(slice, func(i, j int) bool { return slice[i] < slice[j] })
		case reflect.Int64:
			slice := slice.([]int64)
			sort.Slice(slice, func(i, j int) bool { return slice[i] < slice[j] })
		case reflect.Uint:
			slice := slice.([]uint)
			sort.Slice(slice, func(i, j int) bool { return slice[i] < slice[j] })
		case reflect.Uint8:
			slice := slice.([]uint8)
			sort.Slice(slice, func(i, j int) bool { return slice[i] < slice[j] })
		case reflect.Uint16:
			slice := slice.([]uint16)
			sort.Slice(slice, func(i, j int) bool { return slice[i] < slice[j] })
		case reflect.Uint32:
			slice := slice.([]uint32)
			sort.Slice(slice, func(i, j int) bool { return slice[i] < slice[j] })
		case reflect.Uint64:
			slice := slice.([]uint64)
			sort.Slice(slice, func(i, j int) bool { return slice[i] < slice[j] })
		case reflect.Uintptr:
			slice := slice.([]uintptr)
			sort.Slice(slice, func(i, j int) bool { return slice[i] < slice[j] })
		case reflect.Float32:
			slice := slice.([]float32)
			sort.Slice(slice, func(i, j int) bool { return slice[i] < slice[j] })
		case reflect.Float64:
			slice := slice.([]float64)
			sort.Slice(slice, func(i, j int) bool { return slice[i] < slice[j] })
		case reflect.String:
			slice := slice.([]string)
			sort.Slice(slice, func(i, j int) bool { return slice[i] < slice[j] })
		default:
			sort.Slice(slice, func(i, j int) bool { lt, _ := lteq(v.Index(i), v.Index(j)); return lt })
		}
	case reflect.Map:
		iter := v.MapRange()
		for iter.Next() {
			sortable = innerSort(iter.Value())
			if !sortable {
				break
			}
		}
		return sortable
	case reflect.Struct:
		for i := 0; i < t.NumField(); i++ {
			sf := t.Field(i)
			if !sf.IsExported() {
				continue
			}
			innerSort(v.Field(i))
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
	innerSort(v)
	if v.Len() == 0 {
		return
	}
	var last int
	lastv := v.Index(last)
	for next := 1; next < v.Len(); next++ {
		nextv := v.Index(next)
		if _, eq := lteq(lastv, nextv); eq {
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
