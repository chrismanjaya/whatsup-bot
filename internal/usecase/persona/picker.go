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

	poolKey := generatePoolKey(jid, errCode)
	idx := 0

	if len(messagePool.Messages) > 1 {
		if last, seen := p.last[poolKey]; seen && last < len(messagePool.Messages) {
			// Pick from the other n-1 variants.
			idx = p.rng.IntN(len(messagePool.Messages) - 1)
			if idx >= last {
				idx++
			}
		} else {
			idx = p.rng.IntN(len(messagePool.Messages))
		}
	}

	p.last[poolKey] = idx

	return fmt.Sprintf("%s %s", messagePool.Emoji, messagePool.Messages[idx])
}
