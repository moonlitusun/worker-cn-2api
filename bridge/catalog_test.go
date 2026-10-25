package bridge

import (
	"testing"

	"qoder2api/models"
)

func TestNextCatalogKeepsLastFetchedOnFailure(t *testing.T) {
	previous := &models.ModelCatalog{ModelMap: map[string]string{"DeepSeek-Flash": "k"}}

	cat, fetched, ttl := nextCatalog(nil, previous, true)

	if cat != previous || !fetched {
		t.Fatalf("expected the last fetched catalog to keep serving, got %v", cat.Keys())
	}
	if ttl != catalogRetryTTL {
		t.Fatalf("expected a quick retry, got ttl %v", ttl)
	}
}

func TestNextCatalogUsesDefaultWhenNeverFetched(t *testing.T) {
	cat, fetched, ttl := nextCatalog(nil, models.DefaultCatalog(), false)

	if fetched || len(cat.Keys()) == 0 || ttl != catalogRetryTTL {
		t.Fatalf("expected the built-in catalog with a quick retry, got fetched=%v ttl=%v", fetched, ttl)
	}
}

func TestNextCatalogPrefersFreshFetch(t *testing.T) {
	previous := &models.ModelCatalog{ModelMap: map[string]string{"Old": "k"}}
	fresh := &models.ModelCatalog{ModelMap: map[string]string{"New": "k"}}

	cat, fetched, ttl := nextCatalog(fresh, previous, true)

	if cat != fresh || !fetched || ttl != catalogTTL {
		t.Fatalf("expected the fresh catalog for a full TTL, got %v fetched=%v ttl=%v", cat.Keys(), fetched, ttl)
	}
}
