package stats

// Failure contains metadata only: no request/response bodies or credentials.
type Failure struct {
	At         string `json:"at"`
	Provider   string `json:"provider"`
	Model      string `json:"model"`
	DurationMS int64  `json:"duration_ms"`
	Status     int    `json:"status"`
	Kind       string `json:"kind"`
	Reason     string `json:"reason"`
}

const MaxFailures = 200

// RecordFailure immediately saves the bounded recent failure list. Flush is
// serialized with the background saver, so an older snapshot cannot overwrite it.
func (r *Recorder) RecordFailure(f Failure) error {
	r.mu.Lock()
	r.data.Failures = append(r.data.Failures, f)
	if len(r.data.Failures) > MaxFailures {
		r.data.Failures = append([]Failure(nil), r.data.Failures[len(r.data.Failures)-MaxFailures:]...)
	}
	r.dirty = true
	r.mu.Unlock()
	return r.Flush()
}
