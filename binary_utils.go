package main

import "encoding/binary"

func byteToInt32(data []byte) []int32 {
	message := make([]int32, len(data)/4)

	for i := range message {
		message[i] = int32(binary.LittleEndian.Uint32(data[i*4 : (i+1)*4]))
	}
	return message
}
