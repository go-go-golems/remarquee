package compile

import "github.com/google/uuid"

func uuidToBytesLE(u uuid.UUID) []byte {
	b := u[:]
	out := make([]byte, 16)
	out[0] = b[3]
	out[1] = b[2]
	out[2] = b[1]
	out[3] = b[0]
	out[4] = b[5]
	out[5] = b[4]
	out[6] = b[7]
	out[7] = b[6]
	copy(out[8:], b[8:])
	return out
}
