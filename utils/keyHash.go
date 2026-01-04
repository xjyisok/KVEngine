package utils

func HashKey32(key []byte) uint32 {
	const (
		offset32 = 2166136261
		prime32  = 16777619
	)

	var hash uint32 = offset32
	for _, b := range key {
		hash ^= uint32(b)
		hash *= prime32
	}
	return hash
}

func HashKey32Range(key []byte, max uint32) uint32 {
	if max == 0 {
		panic("max must > 0")
	}
	return HashKey32(key) % max
}
