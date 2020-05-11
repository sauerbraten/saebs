package uuid

import (
	crand "crypto/rand"
	"encoding/hex"
	"math/rand"
	"time"
)

type UUID [16]byte

// NewV4 returns a UUID version 4 according to RFC 4122 (https://tools.ietf.org/html/rfc4122#section-4.4).
// Randomness is read from crypto/rand.Reader if possible. When rand.Reader errors, NewV4 falls back to
// math/rand.Rand, after seeding it with time.Now().UnixNano().
func NewV4() UUID {
	uuid := UUID{}

	// read random bytes
	_, err := crand.Read(uuid[:])
	if err != nil {
		// probably not enough entropy, use math/rand (should be good enough)
		rng := rand.New(rand.NewSource(time.Now().UnixNano()))
		rng.Read(uuid[:]) // always returns n==len(uuid), err==nil
	}

	// turn random bytes into a valid UUIDv4 (https://tools.ietf.org/html/rfc4122#section-4.4)
	setVariant(&uuid)
	setVersion4(&uuid)

	return uuid
}

// from https://github.com/google/uuid/blob/1f1ba6fb7a18af3513249fdbdeb6795a98855b68/uuid.go#L132
func (uuid UUID) String() string {
	var buf [36]byte

	hex.Encode(buf[:], uuid[:4])
	buf[8] = '-'
	hex.Encode(buf[9:13], uuid[4:6])
	buf[13] = '-'
	hex.Encode(buf[14:18], uuid[6:8])
	buf[18] = '-'
	hex.Encode(buf[19:23], uuid[8:10])
	buf[23] = '-'
	hex.Encode(buf[24:], uuid[10:])

	return string(buf[:])
}

// setVariant sets the UUID variant (RFC 4122) in the two high bits of uuid[8], leave lower bits untouched.
func setVariant(uuid *UUID) {
	// (uuid[8] & 0x3F) clears the top 2 bits of uuid[8] (example: 01010101 & 00111111 = 00010101)
	// (0x02 << 6) sets 10 (RFC 4122) into the 2 high bits of a byte (00000010 << 6 = 10000000)
	// OR uuid[8] with the created variant byte to keep the relevant parts (00010101 | 10000000 = 10010101)
	uuid[8] = (uuid[8] & 0x3F) | (0x02 << 6)
}

// setVersion sets the version number 4 in the four high bits of the 7th bit (index 6), leaving lower bits untouched.
func setVersion4(uuid *UUID) {
	// (uuid[6] & 0x0F) clears the top 4 bits of uuid[6] (example: 10101010 & 00001111 = 00001010)
	// 0x40 contains 4 (0100) in the 4 high bits (01000000) and only zeroes in the lower 4
	// OR uuid[6] with the version byte to keep the relevant halves (00001010 | 01000000 = 01001010)
	uuid[6] = (uuid[6] & 0x0F) | 0x40
}
