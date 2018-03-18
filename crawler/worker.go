package crawler

// moduel deps
import "fmt"
import "sync"
import "time"
import "net/url"
import "encoding/json"
import "github.com/temoto/robotstxt-go"

// type to describe worker status
type WorkerStatus int

// worker status types
const (
	StatusInitialised WorkerStatus = iota
	StatusFetchingInProgress
	StatusFetchingComplete
	StatusFetchingError
)

// fmt.Stringer definition
func (s WorkerStatus) String() string {
	switch s {
	case StatusInitialised:
		return "initialised"
	case StatusFetchingInProgress:
		return "in-progress"
	case StatusFetchingComplete:
		return "complete"
	case StatusFetchingError:
		return "error"
	default:
		return ""
	}
}

// MarshalJSON definition for WorkerStatus
func (s WorkerStatus) MarshalJSON() ([]byte, error) {
	if r, ok := interface{}(s).(fmt.Stringer); ok {
		return json.Marshal(r.String())
	}

	return nil, fmt.Errorf("Invalid Status: %d", s)
}

// workers are crawlers specific to a domain
type Worker struct {
	// inherit wg
	sync.WaitGroup

	// mutex
	mu sync.Mutex

	// seed URL
	seed *url.URL

	// crawl depth
	crawlDepth int

	// robots agent group
	agent *robotstxt.Group

	// visited URLs
	tracker map[string]struct{}

	// fetch status
	status WorkerStatus

	// nodes tree
	Tree *Resource

	// last updated timestamp
	LastUpdated time.Time
}

// visited tracks if a URL has been crawled before
// to achieve this, we use a sync.Mutex to make it
// safe for concurrent use by multiple goroutines
func (w *Worker) visited(uri string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	_, crawled := w.tracker[uri]
	if !crawled {
		w.tracker[uri] = struct{}{}
	}

	return crawled
}

// Done describes the worker's status
func (w *Worker) Status() WorkerStatus {
	return w.status
}

// CrawlDepth returns the worker's depth
func (w *Worker) CrawlDepth() int {
	return w.crawlDepth
}
