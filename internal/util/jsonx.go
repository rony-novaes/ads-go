package util

import "strings"

func Str(v any) string {
	if v == nil { return "" }
	if s, ok := v.(string); ok { return s }
	return ""
}

func FirstOrEmpty(a []string) string {
	if len(a) > 0 { return a[0] }
	return ""
}

func NestedStr(m map[string]any, path string) string {
	cur := any(m)
	for _, p := range strings.Split(path, ".") {
		mm, ok := cur.(map[string]any)
		if !ok { return "" }
		cur = mm[p]
	}
	if s, ok := cur.(string); ok { return s }
	return ""
}

func NestedBool(m map[string]any, path string) bool {
	cur := any(m)
	for _, p := range strings.Split(path, ".") {
		mm, ok := cur.(map[string]any)
		if !ok { return false }
		cur = mm[p]
	}
	if b, ok := cur.(bool); ok { return b }
	return false
}

func NestedSliceStr(m map[string]any, path string) []string {
	parts := strings.Split(path, ".")
	if len(parts) != 2 { return nil }
	arrRaw, ok := m[parts[0]]
	if !ok { return nil }
	arr, ok := arrRaw.([]any)
	if !ok { return nil }
	out := make([]string, 0, len(arr))
	for _, el := range arr {
		if obj, ok := el.(map[string]any); ok {
			if v, ok := obj[parts[1]]; ok {
				if s, ok := v.(string); ok && s != "" {
					out = append(out, s)
				}
			}
		}
	}
	return out
}
