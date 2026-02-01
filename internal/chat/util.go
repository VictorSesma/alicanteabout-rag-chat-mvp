package chat

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"content-rag-chat/internal/rag"
)

func topScore(results []rag.ScoredChunk) float32 {
	if len(results) == 0 {
		return 0
	}
	return results[0].Score
}

func fmtDuration(d time.Duration) string {
	return fmt.Sprintf("%.2fs (%dms)", d.Seconds(), d.Milliseconds())
}

func newReqID() string {
	var b [16]byte
	if _, err := io.ReadFull(rand.Reader, b[:]); err != nil {
		return fmt.Sprintf("req-%d", time.Now().UnixNano())
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		binary.BigEndian.Uint32(b[0:4]),
		binary.BigEndian.Uint16(b[4:6]),
		binary.BigEndian.Uint16(b[6:8]),
		binary.BigEndian.Uint16(b[8:10]),
		b[10:16],
	)
}
