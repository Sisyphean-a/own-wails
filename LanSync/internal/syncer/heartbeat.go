package syncer

import "time"

func (e *Engine) heartbeat() {
	defer e.wg.Done()
	for {
		e.mu.RLock()
		peers := make([]Peer, 0, len(e.peers))
		for _, p := range e.peers {
			peers = append(peers, p)
		}
		e.mu.RUnlock()
		for _, p := range peers {
			var hello struct {
				ID     string `json:"id"`
				Name   string `json:"name"`
				Port   int    `json:"port"`
				Paired bool   `json:"paired"`
			}
			if err := e.rpcAddress(p.Address, p.ID, "/hello", struct{}{}, &hello); err == nil && hello.ID == p.ID {
				if hello.Name != p.Name {
					e.mu.Lock()
					current := e.peers[p.ID]
					current.Name = hello.Name
					if er := e.store.put("peer/"+p.ID, current); er == nil {
						e.peers[p.ID] = current
					} else {
						e.networkError = er.Error()
					}
					e.mu.Unlock()
				}
			}
		}
		select {
		case <-e.ctx.Done():
			return
		case <-time.After(10 * time.Second):
		}
	}
}
