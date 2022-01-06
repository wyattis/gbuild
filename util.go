package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type StringSlice []string

func (s *StringSlice) Set(val string) error {
	vals := strings.Split(val, ",")
	if len(vals) == 0 {
		return errors.New("must supply a value")
	}
	*s = append(*s, vals...)
	return nil
}

func (s *StringSlice) String() string {
	return strings.Join(*s, ",")
}

type SMap map[string][]string

func (m SMap) OnlyKeys(keys ...string) (res SMap) {
	res = make(SMap, len(keys))
	for _, key := range keys {
		if vals, ok := m[key]; ok {
			res[key] = make([]string, len(vals))
			copy(res[key], vals)
		}
	}
	return
}

func (m SMap) WithoutKeys(keys ...string) (res SMap) {
	res = make(SMap, len(m)-len(keys))
	for key, vals := range m {
		if !StringSliceContains(keys, key) {
			res[key] = make([]string, len(vals))
			copy(res[key], vals)
		}
	}
	return
}

func (m SMap) Copy() (res SMap) {
	res = make(SMap)
	for key, vals := range m {
		res[key] = make([]string, len(vals))
		copy(res[key], vals)
	}
	return
}

func (m SMap) WithoutVals(vals ...string) (res SMap) {
	res = make(SMap)
	for key, v := range m {
		res[key] = StringSliceExclude(v, vals)
	}
	return
}

func (m SMap) Union(other SMap) (res SMap) {
	res = m.Copy()
	for key, vals := range other {
		if existing, ok := res[key]; ok {
			res[key] = StringSliceMerge(existing, vals)
		} else {
			res[key] = make([]string, len(vals))
			copy(res[key], vals)
		}
	}
	return
}

func StringCut(val string, sep string) (left, right string, found bool) {
	if i := strings.Index(val, sep); i >= 0 {
		return val[:i], val[i+len(sep):], true
	}
	return val, "", false
}

func StringCutAny(val string, seperators ...string) (left, right string, found bool) {
	for _, sep := range seperators {
		left, right, found = StringCut(val, sep)
		if found {
			return
		}
	}
	return
}

func StringSplitMany(val string, seperators ...string) (parts []string) {
	parts = append(parts, val)
	for _, sep := range seperators {
		for i, part := range parts {
			vals := strings.Split(part, sep)
			if len(vals) > 1 {
				// Replace existing element with all vals
				parts = append(parts[:i], append(vals, parts[i+1:]...)...)
			}
		}
	}
	return
}

func StringSliceRemove(haystack []string, needle string) (res []string) {
	for i := range haystack {
		if haystack[i] != needle {
			res = append(res, haystack[i])
		}
	}
	return
}

func StringSliceMerge(a []string, b []string) (res []string) {
	res = make([]string, len(a))
	copy(res, a)
	for _, val := range b {
		if !StringSliceContains(a, val) {
			res = append(res, val)
		}
	}
	return
}

func StringSliceCut(items []string, search string) (left, right []string, found bool) {
	var i = 0
	for i = range items {
		if items[i] != search {
			left = append(left, items[i])
		} else {
			break
		}
	}
	if len(items) > i {
		right = items[i+1:]
		found = true
	}
	return
}

func StringSliceExclude(s []string, exclude []string) (res []string) {
	for _, v := range exclude {
		res = StringSliceRemove(s, v)
	}
	return
}

func StringSliceContainsAny(s []string, keys ...string) bool {
	for _, key := range keys {
		if StringSliceContains(s, key) {
			return true
		}
	}
	return false
}

func StringSliceContains(s []string, key string) bool {
	for _, v := range s {
		if v == key {
			return true
		}
	}
	return false
}

func renderString(tmpl *template.Template, data interface{}) (string, error) {
	buf := bytes.NewBufferString("")
	if err := tmpl.Execute(buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func cleanDirGlob(dir, pattern string) error {
	names, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return err
	}
	if len(names) != 0 {
		fmt.Printf("cleaning %d files from %s\n", len(names), dir)
	}
	for _, p := range names {
		if err = os.Remove(p); err != nil {
			return err
		}
	}
	return nil
}
