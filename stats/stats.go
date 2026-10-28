// Package stats provides thread-safe request statistics aggregation
// (totals, per-model counters, hourly buckets) with periodic persistence.
package stats

import (
	"sort"
	"strings"
	"sync"
	"time"
)

// HourKeyFormat is the layout used for hourly bucket keys.
const HourKeyFormat = "2006-01-02T15"

// BucketRetention is how long hourly buckets are kept before pruning.
const BucketRetention = 96 * time.Hour

// ModelStat aggregates request counters for a single model.
type ModelStat struct {
	Model   string `json:"model"`
	Total   int64  `json:"total"`
	Success int64  `json:"success"`
	Failed  int64  `json:"failed"`
}

// HourStat aggregates request counters for a single hour bucket.
type HourStat struct {
	Hour    string `json:"hour"`
	Total   int64  `json:"total"`
	Success int64  `json:"success"`
	Failed  int64  `json:"failed"`
}

// Data is the persisted stats snapshot.
type Data struct {
	Failures []Failure        `json:"failures,omitempty"`
	Channels map[string]*Data `json:"channels,omitempty"`
	Total    int64            `json:"total"`
	Success  int64            `json:"success"`
	Failed   int64            `json:"failed"`
	// Token / billing totals aggregated from the gateway usage frames.
	PromptTokens     int64                 `json:"prompt_tokens"`
	CompletionTokens int64                 `json:"completion_tokens"`
	CachedTokens     int64                 `json:"cached_tokens"`
	Credits          float64               `json:"credits"`
	ByModel          map[string]*ModelStat `json:"by_model,omitempty"`
	Hourly           map[string]*HourStat  `json:"hourly,omitempty"`
}

// ModelRow is a per-model report row served to the admin UI.
type ModelRow struct {
	Model       string  `json:"model"`
	Total       int64   `json:"total"`
	Success     int64   `json:"success"`
	Failed      int64   `json:"failed"`
	SuccessRate float64 `json:"success_rate"`
}

// HourRow is a chart bucket report row served to the admin UI.
type HourRow struct {
	Hour    string `json:"hour"`
	Label   string `json:"label"`
	Total   int64  `json:"total"`
	Success int64  `json:"success"`
	Failed  int64  `json:"failed"`
}

// Report is the aggregated snapshot served to the admin UI.
type Report struct {
	Failures    []Failure `json:"failures"`
	Total       int64     `json:"total"`
	Success     int64     `json:"success"`
	Failed      int64     `json:"failed"`
	SuccessRate float64   `json:"success_rate"`
	// Token / billing totals.
	PromptTokens     int64      `json:"prompt_tokens"`
	CompletionTokens int64      `json:"completion_tokens"`
	CachedTokens     int64      `json:"cached_tokens"`
	Credits          float64    `json:"credits"`
	ByModel          []ModelRow `json:"by_model"`
	Hourly           []HourRow  `json:"hourly"`
}

// Persister persists the stats data (implemented by store.Store).
type Persister interface {
	LoadStats() *Data
	SaveStats(*Data) error
}

// Recorder aggregates request statistics with thread-safe access.
// Data is persisted lazily via Flush so hot request paths never touch disk.
type Recorder struct {
	mu      sync.Mutex
	flushMu sync.Mutex
	data    *Data
	persist Persister
	dirty   bool
}

// NewRecorder creates a recorder, restoring any previously persisted data.
// The loaded data is cloned so later mutations never race with other
// store.save() calls that marshal the persisted snapshot.
func NewRecorder(p Persister) *Recorder {
	r := &Recorder{persist: p}
	if p != nil {
		if d := p.LoadStats(); d != nil {
			r.data = normalize(d).Clone()
		}
	}
	if r.data == nil {
		r.data = newData()
	}

	if r.data.Channels == nil {
		r.data.Channels = map[string]*Data{}
		// Historical model counts identify channels; historical usage/hourly totals do not.
		for _, m := range r.data.ByModel {
			d := r.channelLocked(channelForModel(m.Model))
			d.Total += m.Total
			d.Success += m.Success
			d.Failed += m.Failed
			cp := *m
			d.ByModel[m.Model] = &cp
		}
	}
	return r
}

func newData() *Data {
	return &Data{
		ByModel: map[string]*ModelStat{},
		Hourly:  map[string]*HourStat{},
	}
}

func normalize(d *Data) *Data {
	if d.ByModel == nil {
		d.ByModel = map[string]*ModelStat{}
	}
	if d.Hourly == nil {
		d.Hourly = map[string]*HourStat{}
	}
	return d
}

// Clone returns a deep copy of d. Callers must not hold the recorder mutex
// when the copy is needed for concurrent use.
func (d *Data) Clone() *Data {
	cp := &Data{
		Channels:         make(map[string]*Data, len(d.Channels)),
		Failures:         append([]Failure(nil), d.Failures...),
		Total:            d.Total,
		Success:          d.Success,
		Failed:           d.Failed,
		PromptTokens:     d.PromptTokens,
		CompletionTokens: d.CompletionTokens,
		CachedTokens:     d.CachedTokens,
		Credits:          d.Credits,
		ByModel:          make(map[string]*ModelStat, len(d.ByModel)),
		Hourly:           make(map[string]*HourStat, len(d.Hourly)),
	}
	if d.Channels == nil {
		cp.Channels = nil
	}
	for k, v := range d.Channels {
		if v != nil {
			cp.Channels[k] = v.Clone()
		}
	}
	for k, v := range d.ByModel {
		cp.ByModel[k] = &ModelStat{Model: v.Model, Total: v.Total, Success: v.Success, Failed: v.Failed}
	}
	for k, v := range d.Hourly {
		cp.Hourly[k] = &HourStat{Hour: v.Hour, Total: v.Total, Success: v.Success, Failed: v.Failed}
	}
	return cp
}

// Record counts one completed request.
func (r *Recorder) Record(model string, ok bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.dirty = true
	r.data.record(model, ok)
	r.channelLocked(channelForModel(model)).record(model, ok)
}

func channelForModel(model string) string {
	if strings.HasPrefix(model, "qoder/") {
		return "qoder"
	}
	if strings.HasPrefix(model, "trae/") {
		return "trae"
	}
	return "workbuddy"
}
func (r *Recorder) channelLocked(provider string) *Data {
	if r.data.Channels == nil {
		r.data.Channels = map[string]*Data{}
	}
	d := r.data.Channels[provider]
	if d == nil {
		d = newData()
		r.data.Channels[provider] = d
	}
	return normalize(d)
}
func (d *Data) record(model string, ok bool) {
	d.Total++
	if ok {
		d.Success++
	} else {
		d.Failed++
	}

	m := d.ByModel[model]
	if m == nil {
		m = &ModelStat{Model: model}
		d.ByModel[model] = m
	}
	m.Total++
	if ok {
		m.Success++
	} else {
		m.Failed++
	}

	key := time.Now().Format(HourKeyFormat)
	h := d.Hourly[key]
	if h == nil {
		h = &HourStat{Hour: key}
		d.Hourly[key] = h
	}
	h.Total++
	if ok {
		h.Success++
	} else {
		h.Failed++
	}
}

// RecordUsage adds token/billing totals extracted from a gateway usage frame.
// Called once per successful request when the upstream supplies usage data;
// missing frames (error-interrupted streams) simply skip this call.
func (r *Recorder) RecordUsage(u *Usage) { r.RecordUsageFor("", u) }

func (r *Recorder) RecordUsageFor(provider string, u *Usage) {
	if u == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.dirty = true
	r.data.PromptTokens += int64(u.PromptTokens)
	r.data.CompletionTokens += int64(u.CompletionTokens)
	r.data.CachedTokens += int64(u.CachedTokens)
	r.data.Credits += u.Credits
	if provider != "" {
		d := r.channelLocked(provider)
		d.PromptTokens += int64(u.PromptTokens)
		d.CompletionTokens += int64(u.CompletionTokens)
		d.CachedTokens += int64(u.CachedTokens)
		d.Credits += u.Credits
	}

}

// Usage is the token accounting payload produced by transform.Usage.
// Declared locally to avoid an import cycle (transform is a lower layer).
type Usage struct {
	PromptTokens     int
	CompletionTokens int
	CachedTokens     int
	Credits          float64
}

// Flush persists pending data if anything changed since the last flush.
// A deep copy is saved so concurrent Record calls never race with the
// persister's serialization of the snapshot.
func (r *Recorder) Flush() error {
	r.flushMu.Lock()
	defer r.flushMu.Unlock()
	r.mu.Lock()
	if !r.dirty {
		r.mu.Unlock()
		return nil
	}
	r.pruneLocked()
	cp := r.data.Clone()
	r.dirty = false
	r.mu.Unlock()

	if r.persist == nil {
		return nil
	}
	if err := r.persist.SaveStats(cp); err != nil {
		r.mu.Lock()
		r.dirty = true
		r.mu.Unlock()
		return err
	}
	return nil
}

// pruneLocked drops hourly buckets older than BucketRetention.
// Caller must hold the mutex.
func (r *Recorder) pruneLocked() {
	cutoff := time.Now().Add(-BucketRetention).Format(HourKeyFormat)
	for _, d := range r.data.Channels {
		if d != nil {
			for k := range d.Hourly {
				if k < cutoff {
					delete(d.Hourly, k)
				}
			}
		}
	}
	for k := range r.data.Hourly {
		if k < cutoff {
			delete(r.data.Hourly, k)
		}
	}
}

// Report returns a snapshot for the admin UI: totals, per-model rows sorted by
// total (descending), and the last 24 hourly buckets in ascending order.
func (r *Recorder) Report(provider ...string) *Report {
	r.mu.Lock()
	defer r.mu.Unlock()

	d := r.data
	if len(provider) > 0 && provider[0] != "" {
		d = r.data.Channels[provider[0]]
		if d == nil {
			d = newData()
		}
	}
	rep := &Report{
		Total:            d.Total,
		Success:          d.Success,
		Failed:           d.Failed,
		PromptTokens:     d.PromptTokens,
		CompletionTokens: d.CompletionTokens,
		CachedTokens:     d.CachedTokens,
		Credits:          d.Credits,
		ByModel:          []ModelRow{},
		Hourly:           []HourRow{},
	}
	rep.Failures = []Failure{}
	for i := len(r.data.Failures) - 1; i >= 0; i-- {
		f := r.data.Failures[i]
		if len(provider) == 0 || provider[0] == "" || f.Provider == provider[0] {
			rep.Failures = append(rep.Failures, f)
		}
	}
	if d.Total > 0 {
		rep.SuccessRate = float64(d.Success) / float64(d.Total)
	}

	for _, m := range d.ByModel {
		rate := 0.0
		if m.Total > 0 {
			rate = float64(m.Success) / float64(m.Total)
		}
		rep.ByModel = append(rep.ByModel, ModelRow{
			Model:       m.Model,
			Total:       m.Total,
			Success:     m.Success,
			Failed:      m.Failed,
			SuccessRate: rate,
		})
	}
	sort.Slice(rep.ByModel, func(i, j int) bool {
		if rep.ByModel[i].Total != rep.ByModel[j].Total {
			return rep.ByModel[i].Total > rep.ByModel[j].Total
		}
		return rep.ByModel[i].Model < rep.ByModel[j].Model
	})

	now := time.Now()
	start := now.Add(-23 * time.Hour).Truncate(time.Hour)
	for i := 0; i < 24; i++ {
		ts := start.Add(time.Duration(i) * time.Hour)
		key := ts.Format(HourKeyFormat)
		row := HourRow{Hour: key, Label: ts.Format("15:04")}
		if h := d.Hourly[key]; h != nil {
			row.Total = h.Total
			row.Success = h.Success
			row.Failed = h.Failed
		}
		rep.Hourly = append(rep.Hourly, row)
	}
	return rep
}
