package util

type number interface {
	int | int8 | int16 | int32 | int64 | uint | uint8 | uint16 | uint32 | uint64 | float32 | float64
}

func Max[T number](elements ...T) T {
	var largest T

	for _, value := range elements {
		if value > largest {
			largest = value
		}
	}

	return largest
}
