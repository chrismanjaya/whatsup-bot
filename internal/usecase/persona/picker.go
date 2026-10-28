package persona

import (
	"fmt"
	"math/rand/v2"
	"sync"
	"whatsupbot/internal/constant"
)

type Picker struct {
	mu   sync.Mutex
	rng  *rand.Rand
	last map[string]int
}

func NewPicker() *Picker {
	randSource := rand.NewPCG(rand.Uint64(), rand.Uint64())
	return &Picker{
		rng:  rand.New(randSource),
		last: make(map[string]int),
	}
}

func generatePoolKey(jid string, errCode *constant.ErrCode) string {
	return fmt.Sprintf("%s:%d", jid, errCode.Code)
}

func generateAckPoolKey(jid string) string {
	return jid + ":ack"
}

func (p *Picker) Pick(jid string, errCode *constant.ErrCode) string {
	p.mu.Lock()
	defer p.mu.Unlock()

	if errCode == nil {
		return ""
	}

	messagePool, found := ErrorMessagePools[errCode.Code]
	if !found {
		return ""
	}

	idx := p.pickIndex(generatePoolKey(jid, errCode), len(messagePool.Messages))

	return fmt.Sprintf("%s %s", messagePool.Emoji, messagePool.Messages[idx])
}

// PickAck returns one of AckMessages, avoiding the variant this jid saw last.
func (p *Picker) PickAck(jid string) string {
	p.mu.Lock()
	defer p.mu.Unlock()

	return AckMessages[p.pickIndex(generateAckPoolKey(jid), len(AckMessages))]
}

// pickIndex picks a random index in [0, size), skipping the one last picked for poolKey. Callers must hold mu.
func (p *Picker) pickIndex(poolKey string, size int) int {
	idx := 0

	if size > 1 {
		if last, seen := p.last[poolKey]; seen && last < size {
			// Pick from the other n-1 variants.
			idx = p.rng.IntN(size - 1)
			if idx >= last {
				idx++
			}
		} else {
			idx = p.rng.IntN(size)
		}
	}

	p.last[poolKey] = idx

	return idx
}
