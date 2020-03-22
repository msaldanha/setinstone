package util

func LeftPadBytes(slice []byte, lenght int) []byte {
	if lenght <= len(slice) {
		return slice
	}

	padded := make([]byte, lenght)
	copy(padded[lenght-len(slice):], slice)

	return padded
}
