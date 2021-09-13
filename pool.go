package main

var (
	wsips []uint32
)

func hasExist(wsip uint32) bool {
	for _, ip := range wsips {
		if ip == wsip {
			return true
		}
	}
	return false
}
