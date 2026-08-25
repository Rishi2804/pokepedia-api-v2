package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/elastic/go-elasticsearch/v9"
	"github.com/elastic/go-elasticsearch/v9/esutil"

	"github.com/Rishi2804/pokepedia-api-v2/internal/search"
)

// bulkIndex creates a new versioned index, loads docs into it, verifies the
// load, and only on success atomically swaps the alias onto it. It never
// touches the currently-live index until the new one is proven good — a
// failed run leaves the target index in place, unaliased, for inspection.
func bulkIndex(ctx context.Context, es *elasticsearch.TypedClient, alias string, docs []indexDoc, keep int) error {
	target := fmt.Sprintf("%s-%s", alias, time.Now().UTC().Format("20060102-150405"))

	log.Printf("creating index %s", target)
	if _, err := es.Indices.Create(target).Raw(bytes.NewReader(search.Mapping)).Do(ctx); err != nil {
		return fmt.Errorf("create index: %w", err)
	}

	// Refreshing per-flush is waste for a write-once index nobody is
	// reading yet; restored once the bulk load is done.
	if err := setRefreshInterval(ctx, es, target, "-1"); err != nil {
		return fmt.Errorf("disable refresh: %w", err)
	}

	var failed atomic.Int64
	bi, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		Client:     es,
		Index:      target,
		NumWorkers: 2,
		FlushBytes: 1 << 20,
		OnError: func(_ context.Context, err error) {
			log.Printf("bulk indexer error: %v", err)
		},
	})
	if err != nil {
		return fmt.Errorf("new bulk indexer: %w", err)
	}

	for _, d := range docs {
		body, err := json.Marshal(d)
		if err != nil {
			return fmt.Errorf("marshal %s:%d: %w", d.Type, d.EntityID, err)
		}
		id := d.Type + ":" + strconv.Itoa(int(d.EntityID))
		err = bi.Add(ctx, esutil.BulkIndexerItem{
			Action:     "index",
			DocumentID: id,
			Body:       bytes.NewReader(body),
			OnFailure: func(_ context.Context, item esutil.BulkIndexerItem, res esutil.BulkIndexerResponseItem, err error) {
				failed.Add(1)
				if err != nil {
					log.Printf("indexer: %s failed: %v", item.DocumentID, err)
				} else {
					log.Printf("indexer: %s failed: %s %s", item.DocumentID, res.Error.Type, res.Error.Reason)
				}
			},
		})
		if err != nil {
			return fmt.Errorf("add %s: %w", id, err)
		}
	}

	if err := bi.Close(ctx); err != nil {
		return fmt.Errorf("close bulk indexer: %w", err)
	}
	stats := bi.Stats()
	log.Printf("bulk indexer stats: indexed=%d failed=%d", stats.NumIndexed, stats.NumFailed)

	if err := setRefreshInterval(ctx, es, target, "1s"); err != nil {
		return fmt.Errorf("restore refresh: %w", err)
	}
	if _, err := es.Indices.Refresh().Index(target).Do(ctx); err != nil {
		return fmt.Errorf("refresh: %w", err)
	}
	// The data is a static snapshot, so one segment is strictly optimal:
	// best compression, best term-dictionary locality, no background
	// merges ever.
	if _, err := es.Indices.Forcemerge().Index(target).MaxNumSegments("1").Do(ctx); err != nil {
		return fmt.Errorf("forcemerge: %w", err)
	}

	countResp, err := es.Count().Index(target).Do(ctx)
	if err != nil {
		return fmt.Errorf("count: %w", err)
	}

	if failed.Load() > 0 || stats.NumFailed > 0 {
		return fmt.Errorf("refusing to swap alias: %d documents failed to index (index %s left in place for inspection)", failed.Load(), target)
	}
	if countResp.Count != int64(len(docs)) {
		return fmt.Errorf("refusing to swap alias: indexed count %d != expected %d (index %s left in place for inspection)", countResp.Count, len(docs), target)
	}

	if err := swapAlias(ctx, es, alias, target); err != nil {
		return fmt.Errorf("swap alias: %w", err)
	}
	log.Printf("alias %s now points at %s", alias, target)

	return prune(ctx, es, alias, keep)
}

func setRefreshInterval(ctx context.Context, es *elasticsearch.TypedClient, index, interval string) error {
	body := fmt.Sprintf(`{"index":{"refresh_interval":%q}}`, interval)
	_, err := es.Indices.PutSettings().Indices(index).Raw(strings.NewReader(body)).Do(ctx)
	return err
}

// swapAlias removes alias from any index it currently points at (must_exist:
// false, so the first-ever run doesn't fail on a nonexistent removal) and
// adds it to target, as a single atomic action.
func swapAlias(ctx context.Context, es *elasticsearch.TypedClient, alias, target string) error {
	body := fmt.Sprintf(`{
		"actions": [
			{"remove": {"index": %q, "alias": %q, "must_exist": false}},
			{"add":    {"index": %q, "alias": %q}}
		]
	}`, alias+"-*", alias, target, alias)
	_, err := es.Indices.UpdateAliases().Raw(strings.NewReader(body)).Do(ctx)
	return err
}

// prune deletes all but the newest keep versioned indices behind alias, so
// reruns don't accumulate one index per run forever.
func prune(ctx context.Context, es *elasticsearch.TypedClient, alias string, keep int) error {
	resp, err := es.Indices.Get(alias + "-*").Do(ctx)
	if err != nil {
		return err
	}

	var names []string
	for name := range resp {
		names = append(names, name)
	}
	// Timestamp suffix is fixed-width (20060102-150405), so lexicographic
	// order is chronological order.
	sort.Sort(sort.Reverse(sort.StringSlice(names)))

	if len(names) <= keep {
		return nil
	}
	for _, name := range names[keep:] {
		log.Printf("pruning old index %s", name)
		if _, err := es.Indices.Delete(name).Do(ctx); err != nil {
			return fmt.Errorf("delete %s: %w", name, err)
		}
	}
	return nil
}
