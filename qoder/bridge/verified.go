package bridge

import (
	"context"
	"fmt"
	"worker-cn-2api/qoder/models"
)

// VerifiedCatalog prevents a failed account from being advertised using defaults.
func (b *OpenAiBridge) VerifiedCatalog(ctx context.Context) (*models.ModelCatalog, error) {
	if err := b.EnsureFreshSession(ctx); err != nil {
		return nil, err
	}
	catalog := b.GetCatalog(ctx)
	b.catalogMu.Lock()
	valid := b.catalogFetch
	b.catalogMu.Unlock()
	if !valid || catalog == nil {
		return nil, fmt.Errorf("Qoder model catalog could not be verified")
	}
	return catalog, nil
}
