package idgen

import (
	"context"
	"encoding/binary"
	"math/rand/v2"
	"time"

	"github.com/google/uuid"
)

type Generator struct {
	pool chan uuid.UUID
}

func NewGenerator(ctx context.Context, bufferSize int) *Generator {
	g := &Generator{
		pool: make(chan uuid.UUID, bufferSize),
	}
	go g.fillPool(ctx)
	return g
}

func (g *Generator) Next() uuid.UUID {
	select {
	case id := <-g.pool:
		return id
	default:
		return g.generateV7()
	}
}

func (g *Generator) generateV7() uuid.UUID {
	var res [16]byte

	// Timestamp (48 bit)
	now := uint64(time.Now().UnixMilli())
	res[0] = byte(now >> 40)
	res[1] = byte(now >> 32)
	res[2] = byte(now >> 24)
	res[3] = byte(now >> 16)
	res[4] = byte(now >> 8)
	res[5] = byte(now)

	// We need 12 bits of randomness + 4 bits for the version
	v7randA := uint16(rand.Uint32()&0x0fff) | 0x7000
	binary.BigEndian.PutUint16(res[6:8], v7randA)

	binary.BigEndian.PutUint64(res[8:], rand.Uint64())

	// Then we force the top 2 bits of Byte 8 to be 10
	// 0x3f is 00111111
	// 0x80 is 10000000
	res[8] = (res[8] & 0x3f) | 0x80

	return res
}

func (g *Generator) fillPool(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		id := g.generateV7()
		select {
		case <-ctx.Done():
			return

		case g.pool <- id:

		default:
			select {
			case <-ctx.Done():
				return
			case <-time.After(100 * time.Microsecond):
			}
		}
	}
}
