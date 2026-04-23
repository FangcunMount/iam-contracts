package jwks

import (
	"context"
	"time"
)

func (m *JWKSManager) shouldStartRefreshLoop() bool {
	return m.config.RefreshInterval > 0 && m.cache != nil
}

func (m *JWKSManager) refreshLoop() {
	ticker := time.NewTicker(m.config.RefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			_ = m.ForceRefresh(context.Background())
		case <-m.stopCh:
			return
		}
	}
}
