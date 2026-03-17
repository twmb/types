// Package types contains helper functions for arbitrary types (deep less,
// deep equal, deep sort).
package types

import (
	"cmp"
	"reflect"
	"slices"
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
func Less(l, r any) bool {
	lt, _ := lteq(new(pointers), reflect.ValueOf(l), reflect.ValueOf(r))
	return lt
}

// Equal returns whether l is deeply equal to r. The input values must have the
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
func Equal(l, r any) bool {
	_, eq := lteq(new(pointers), reflect.ValueOf(l), reflect.ValueOf(r))
	return eq
}

// Compare returns whether l is less than, equal to, or larger than r,
// following the same rules as Less and Equal.
func Compare(l, r any) int {
	lt, eq := lteq(new(pointers), reflect.ValueOf(l), reflect.ValueOf(r))
	if lt {
		return -1
	} else if eq {
		return 0
	}
	return 1
}

type pointers map[unsafe.Pointer]struct{}

func (p *pointers) hasOrAdd(ptr unsafe.Pointer) bool {
	if *p == nil {
		*p = pointers{ptr: {}}
		return false
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

func (p *pointers) compareValues(a, b reflect.Value) int {
	lt, eq := lteq(p, a, b)
	if lt {
		return -1
	}
	if eq {
		return 0
	}
	return 1
}

func lteq(p *pointers, lv, rv reflect.Value) (lt, eq bool) {
	t := lv.Type()
	if t != rv.Type() {
		panic("unequal types")
	}

	if k := t.Kind(); k != reflect.Struct {
		return lteqKind(p, k, lv, rv)
	}

	for i := range t.NumField() {
		sf := t.Field(i)
		if !sf.IsExported() {
			continue
		}

		lt, eq := lteqKind(p, sf.Type.Kind(), lv.Field(i), rv.Field(i))
		if !eq {
			return lt, false
		}
	}

	return false, true
}

func orderedLtEq[T cmp.Ordered](l, r T) (lt, eq bool) {
	if l == r {
		return false, true
	}
	return l < r, false
}

func floatLtEq(l, r float64) (lt, eq bool) {
	c := cmp.Compare(l, r)
	return c < 0, c == 0
}

func c128lt(l, r complex128) (lt, eq bool) {
	lt, eq = floatLtEq(real(l), real(r))
	if eq {
		lt, eq = floatLtEq(imag(l), imag(r))
	}
	return lt, eq
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
		return orderedLtEq(lv.Int(), rv.Int())
	case reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64,
		reflect.Uintptr:
		return orderedLtEq(lv.Uint(), rv.Uint())
	case reflect.Float32,
		reflect.Float64:
		return floatLtEq(lv.Float(), rv.Float())
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
		return orderedLtEq(lv.String(), rv.String())
	case reflect.Struct:
		return lteq(p, lv, rv)

	case reflect.Array,
		reflect.Slice:
		ll, lr := lv.Len(), rv.Len()
		lt, eq = ll < lr, ll == lr
		if eq {
			for i := range lr {
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
			slices.SortFunc(lkeys, p.compareValues)
			slices.SortFunc(rkeys, p.compareValues)

			for i, lk := range lkeys {
				rk := rkeys[i]
				lt, eq = lteq(p, lk, rk)
				if !eq {
					return lt, false
				}
			}
			for _, lk := range lkeys {
				lval := lv.MapIndex(lk)
				rval := rv.MapIndex(lk)
				lt, eq = lteq(p, lval, rval)
				if !eq {
					return lt, false
				}
			}
		}
		return lt, eq

	case reflect.Pointer:
		if lv.IsNil() {
			return !rv.IsNil(), rv.IsNil()
		} else if rv.IsNil() {
			return false, false
		}

		lptr, rptr := unsafe.Pointer(lv.Pointer()), unsafe.Pointer(rv.Pointer())
		if lptr == rptr {
			return false, true
		}
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

// Sort deeply sorts any slice anywhere within s, traversing into maps, slices,
// and exported struct fields. Any non-primitive type is less than the other
// following the rules of Less.
//
// Note that this function performs value copies. This must not be used to sort
// types that are not safe to copy. For example, this must not sort
// []struct{sync.Mutex}, but it can sort []*struct{sync.Mutex}.
//
// If a slice contains a type that has a Less method that accepts itself and
// returns a bool, Sort uses that type's Less method to sort the slice.
func Sort(s any) {
	innerSort(new(pointers), reflect.ValueOf(s))
}

func innerSort(p *pointers, v reflect.Value) (sortable bool) {
	t := v.Type()
	switch t.Kind() {
	case reflect.Pointer:
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
		if v.Len() == 0 {
			return true
		}

		switch t.Elem().Kind() {
		case reflect.Bool:
			slice := unsafe.Slice((*bool)(unsafe.Pointer(v.Pointer())), v.Len())
			slices.SortFunc(slice, func(a, b bool) int {
				if a == b {
					return 0
				}
				if !a {
					return -1
				}
				return 1
			})
		case reflect.Int:
			slices.Sort(unsafe.Slice((*int)(unsafe.Pointer(v.Pointer())), v.Len()))
		case reflect.Int8:
			slices.Sort(unsafe.Slice((*int8)(unsafe.Pointer(v.Pointer())), v.Len()))
		case reflect.Int16:
			slices.Sort(unsafe.Slice((*int16)(unsafe.Pointer(v.Pointer())), v.Len()))
		case reflect.Int32:
			slices.Sort(unsafe.Slice((*int32)(unsafe.Pointer(v.Pointer())), v.Len()))
		case reflect.Int64:
			slices.Sort(unsafe.Slice((*int64)(unsafe.Pointer(v.Pointer())), v.Len()))
		case reflect.Uint:
			slices.Sort(unsafe.Slice((*uint)(unsafe.Pointer(v.Pointer())), v.Len()))
		case reflect.Uint8:
			slices.Sort(unsafe.Slice((*uint8)(unsafe.Pointer(v.Pointer())), v.Len()))
		case reflect.Uint16:
			slices.Sort(unsafe.Slice((*uint16)(unsafe.Pointer(v.Pointer())), v.Len()))
		case reflect.Uint32:
			slices.Sort(unsafe.Slice((*uint32)(unsafe.Pointer(v.Pointer())), v.Len()))
		case reflect.Uint64:
			slices.Sort(unsafe.Slice((*uint64)(unsafe.Pointer(v.Pointer())), v.Len()))
		case reflect.Uintptr:
			slices.Sort(unsafe.Slice((*uintptr)(unsafe.Pointer(v.Pointer())), v.Len()))
		case reflect.Float32:
			slices.Sort(unsafe.Slice((*float32)(unsafe.Pointer(v.Pointer())), v.Len()))
		case reflect.Float64:
			slices.Sort(unsafe.Slice((*float64)(unsafe.Pointer(v.Pointer())), v.Len()))
		case reflect.String:
			slices.Sort(unsafe.Slice((*string)(unsafe.Pointer(v.Pointer())), v.Len()))

		case reflect.Struct, reflect.Interface:
			v0 := v.Index(0)
			t0 := v0.Type()
			if meth, ok := t0.MethodByName("Less"); ok {
				less := v0.Method(meth.Index)
				tless := less.Type()
				if tless.NumIn() == 1 && tless.NumOut() == 1 && tless.In(0) == t0 && tless.Out(0) == reflect.TypeFor[bool]() {
					vslice := make([]reflect.Value, 1)
					sort.Slice(v.Interface(), func(i, j int) bool {
						vslice[0] = v.Index(j)
						return v.Index(i).Method(meth.Index).Call(vslice)[0].Bool()
					})
					return true
				}
			}
			fallthrough

		default:
			// Each element of this **top** level slice is sorted,
			// but now we have to sort each element's innards. We
			// do this before sorting the type itself, because
			// sorting innards may change the outer comparison.
			for i := range v.Len() {
				if !innerSort(p, v.Index(i)) {
					break
				}
			}

			sort.Slice(v.Interface(), func(i, j int) bool { lt, _ := lteq(p, v.Index(i), v.Index(j)); return lt })
		}

	case reflect.Map:
		iter := v.MapRange()
		for iter.Next() {
			if !innerSort(p, iter.Value()) {
				return false
			}
		}
		return true
	case reflect.Struct:
		for i := range t.NumField() {
			sf := t.Field(i)
			if !sf.IsExported() {
				continue
			}
			innerSort(p, v.Field(i))
		}
	default:
		return false
	}
	return true
}

// DistinctInPlace sorts *s using the rules of Sort in this package, and
// compacts it in place using the rules of Equal in this package.
//
// This is similar to the slice generic version of sorting and compacting, but
// allows for even more types to be sorted.
func DistinctInPlace[S ~[]E, E any](s *S) {
	v := reflect.ValueOf(s).Elem()
	p := new(pointers)
	innerSort(p, v)
	*s = slices.CompactFunc(*s, func(a, b E) bool {
		_, eq := lteq(p, reflect.ValueOf(a), reflect.ValueOf(b))
		return eq
	})
}
