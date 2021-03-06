package pg

import (
	"reflect"
	"strings"
	"sync"
)

var (
	structs = newStructCache()
)

func isUpper(c byte) bool {
	return c >= 'A' && c <= 'Z'
}

func isLower(c byte) bool {
	return !isUpper(c)
}

func toLower(c byte) byte {
	return c + 32
}

func formatColumnName(s string) string {
	b := []byte(s)
	r := make([]byte, 0, len(b))
	for i := 0; i < len(b); i++ {
		c := b[i]
		if isUpper(c) {
			if i-1 > 0 && i+1 < len(b) && (isLower(b[i-1]) || isLower(b[i+1])) {
				r = append(r, '_', toLower(c))
			} else {
				r = append(r, toLower(c))
			}
		} else {
			r = append(r, c)
		}
	}
	return string(r)
}

type structCache struct {
	fields  map[reflect.Type]map[string][]int
	fieldsl sync.RWMutex

	methods  map[reflect.Type]map[string]int
	methodsl sync.RWMutex
}

func newStructCache() *structCache {
	return &structCache{
		fields:  make(map[reflect.Type]map[string][]int),
		methods: make(map[reflect.Type]map[string]int),
	}
}

func (c *structCache) Fields(typ reflect.Type) map[string][]int {
	c.fieldsl.RLock()
	indxs, ok := c.fields[typ]
	c.fieldsl.RUnlock()
	if ok {
		return indxs
	}

	indxs = fields(typ)

	c.fieldsl.Lock()
	c.fields[typ] = indxs
	c.fieldsl.Unlock()

	return indxs
}

func (c *structCache) Methods(typ reflect.Type) map[string]int {
	c.methodsl.RLock()
	indxs, ok := c.methods[typ]
	c.methodsl.RUnlock()
	if ok {
		return indxs
	}

	num := typ.NumMethod()
	indxs = make(map[string]int, num)
	for i := 0; i < num; i++ {
		m := typ.Method(i)
		if m.Type.NumIn() > 1 {
			continue
		}
		if m.Type.NumOut() != 1 {
			continue
		}
		indxs[m.Name] = m.Index
	}

	c.methodsl.Lock()
	c.methods[typ] = indxs
	c.methodsl.Unlock()

	return indxs
}

func fields(typ reflect.Type) map[string][]int {
	num := typ.NumField()
	dst := make(map[string][]int, num)
	for i := 0; i < num; i++ {
		f := typ.Field(i)

		if f.Anonymous {
			typ := f.Type
			if typ.Kind() == reflect.Ptr {
				typ = typ.Elem()
			}
			for name, indx := range fields(typ) {
				dst[name] = append(f.Index, indx...)
			}
			continue
		}

		if f.PkgPath != "" {
			continue
		}

		tokens := strings.Split(f.Tag.Get("pg"), ",")
		name := tokens[0]
		if name == "-" {
			continue
		}
		if name == "" {
			name = formatColumnName(f.Name)
		}
		dst[name] = f.Index
	}
	return dst
}
