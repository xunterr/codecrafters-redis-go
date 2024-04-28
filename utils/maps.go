package utils

func ArrayToMap[T comparable](array []T) map[T]struct{} {
	res := make(map[T]struct{})
	for _, k := range array {
		res[k] = struct{}{}
	}
	return res
}
