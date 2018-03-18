package crawler

// module deps
import "io"
import "os"
import "log"
import "mime"
import "sync"
import "time"
import "errors"
import "net/url"
import "net/http"
import "golang.org/x/net/html"
import "github.com/temoto/robotstxt-go"
import "github.com/jackdanger/collectlinks"

// constants
const (
	// throttling rate
	DefaultThrottlingRate = 20

	// max crawl depth
	DefaultMaxCrawlDepth = 5

	// default compliance level with robots.txt policy
	// @see https://moz.com/learn/seo/robotstxt
	DefaultComplyWithRobotPolicy = true

	// DefaultUserAgent is the default user agent string in HTTPRequest
	DefaultUserAgent = "GoCrawler/v0.1 (+https://github.com/q/gocrawler)"
)

// relative pathof robots.txt at the domain level
var robotsTxtParsedPath, _ = url.Parse("/robots.txt")

// error definitions
var ErrDomainAlreadyRegistered = errors.New("domain is already registered/crawled")

// normalises relative URLs to absolute URLs
// checks that the link belongs to the parent domain
func normaliseURL(href string, base *url.URL) *url.URL {
	uri, err := url.Parse(href)
	if err != nil {
		return nil
	}

	// if the link belongs to a different domain
	// we do not want to normalise / crawl the link
	if uri.Host != "" && uri.Host != base.Host {
		return nil
	}

	uri = base.ResolveReference(uri)

	if uri.Scheme != "http" && uri.Scheme != "https" {
		return nil
	}

	return uri
}

// getTitleForPage parses the HTML and retrieves the `title` tag
func getTitleForPage(doc io.ReadCloser) (string, error) {
	titleTag := "title"

	page := html.NewTokenizer(doc)
	for {
		tt := page.Next()
		switch tt {
		case html.ErrorToken:
			return "", page.Err()
		}

		token := page.Token()
		switch token.Data {
		case titleTag:
			tt = page.Next()
			switch tt {
			case html.ErrorToken:
				return "", page.Err()
			case html.TextToken:
				token = page.Token()
				return token.Data, nil
			}
		}
	}

	return "", nil
}

// resource describes a web page and it's nodes
type Resource struct {
	// mutex
	sync.Mutex

	// resource URL
	URL *url.URL `json:"_"`

	// string version
	URLString string `json:"url"`

	// from meta
	Title string `json:"title"`

	// HTTP StatusCode
	HTTPStatusCode int `json:"status"`

	// root node
	Root *url.URL `json:"_"`

	// parent node ancestry
	Parent []string `json:"_"`

	// current depth
	Depth int `json:"depth"`

	// child nodes
	Nodes []*Resource `json:"nodes"`

	// last fetched timestamp
	LastFetched time.Time `json:"_"`
}

// queue is a task queue for crawlers
type Queue struct {
	// track the state of queue, so workers
	// need not try to receive tasks from a
	// closed channel, thus avoiding panics
	closed bool

	// work channel
	ch chan *Resource
}

// Logger defines the logging interface
type Logger interface {
	SetOutput(w io.Writer)
	SetPrefix(prefix string)
	Fatal(v ...interface{})
	Fatalf(format string, v ...interface{})
	Panic(v ...interface{})
	Panicf(format string, v ...interface{})
	Print(v ...interface{})
	Printf(format string, v ...interface{})
}

// crawler is a collection of workers
// that crawl their respective domains
type Crawler struct {
	// mutex
	sync.Mutex

	// user agent to send
	UserAgent string

	// http client
	HTTPClient *http.Client

	// logger interface
	Logger Logger

	// registered workers
	workers map[string]*Worker

	// work Queue
	q *Queue

	// throttle channel
	throttle chan bool

	// channel to listen for close event
	stop chan chan error
}

// New returns a new crawler
func New() *Crawler {
	c := &Crawler{
		UserAgent:  DefaultUserAgent,
		HTTPClient: http.DefaultClient,
		Logger:     log.New(os.Stderr, "gocrawler", log.LstdFlags),
		stop:       make(chan chan error),
		workers:    make(map[string]*Worker),
		q:          &Queue{ch: make(chan *Resource, 100)},
		throttle:   make(chan bool, DefaultThrottlingRate),
	}

	go c.loop()
	return c
}

// loop listens for channel events and
// sends them to get enqueued and processed
func (c *Crawler) loop() {
	for {
		select {
		case resource := <-c.q.ch:
			c.enqueue(resource)
		case errc := <-c.stop:
			close(c.stop)
			c.q.closed = true
			close(c.q.ch)
			errc <- nil
			return // we're done
		}
	}
}

// Close cancels the subscriptions in flight,
// closes the Updates channel, and returns the
// last fetch error, if any that is captured
func (c *Crawler) Close() error {
	log.Println("[WARN] received close event, waiting for listeners to shut down")

	// make a err channel
	errc := make(chan error)

	// send a stop signal
	c.stop <- errc

	// wait for close to complete
	log.Println("[WARN] listeners shut down, waiting for crawlers to drain")
	for _, worker := range c.workers {
		worker.Wait()
	}

	log.Println("[WARN] shut down complete, exiting")
	return <-errc
}

// Worker returns worker for a given domain
func (c *Crawler) Worker(domain string) *Worker {
	worker, _ := c.workers[domain]
	return worker
}

// recursively finds the correct leaf for
// the node to be added under the root node
func addNode(parent, child *Resource) error {
	if child.Parent[len(child.Parent)-1] == parent.URL.String() {
		parent.Nodes = append(parent.Nodes, child)
		return nil
	}

	for _, p := range child.Parent[1:] {
		for _, node := range parent.Nodes {
			if node.URL.String() == p {
				return addNode(node, child)
			}
		}
	}

	return nil
}

// append adds a node to the list at the correct
// leaf in the tree belonging to the root node
func (c *Crawler) append(resource *Resource) {
	worker, _ := c.workers[resource.Root.String()]
	if worker.Tree == nil {
		worker.Tree = resource
		return
	}

	worker.Tree.Lock()
	defer worker.Tree.Unlock()
	worker.LastUpdated = time.Now()

	// insert children at depth > 1
	if len(resource.Parent) == 1 && resource.Parent[0] == worker.Tree.URL.String() {
		worker.Tree.Nodes = append(worker.Tree.Nodes, resource)
		return
	}

	addNode(worker.Tree, resource)
}

// Crawl initialises crawler by looking up robots.txt
// and then seeds the queue with a initial resource
func (c *Crawler) Crawl(rawurl string, depth int) error {
	c.Lock()
	defer c.Unlock()

	u, err := url.Parse(rawurl)
	if err != nil {
		return err
	}

	if _, exists := c.workers[u.String()]; exists {
		return ErrDomainAlreadyRegistered
	}

	res, err := c.HTTPClient.Get(u.ResolveReference(robotsTxtParsedPath).String())
	if err != nil {
		return err
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	robData, err := robotstxt.FromResponse(res)
	if err != nil {
		return err
	}

	agent := robData.FindGroup(c.UserAgent)
	if agent == nil {
		return err
	}

	if depth == 0 {
		depth = DefaultMaxCrawlDepth
	}

	c.workers[u.String()] = &Worker{
		seed:       u,
		agent:      agent,
		crawlDepth: depth,
		status:     StatusInitialised,
		tracker:    make(map[string]struct{}),
	}

	go func(u string) {
		worker, _ := c.workers[u]
		delay := 15 * time.Second
		ticker := time.NewTicker(delay)
		for {
			select {
			case <-ticker.C:
				if worker.LastUpdated.Add(delay).After(time.Now()) {
					// there is no activity in the last 15 seconds
					// so assume the fetch is complete; there is no
					// other "better" way to determine this, because
					// the race condition between adding a work to the
					// WaitGroup and wg.Done is a lot flakier than this
					worker.status = StatusFetchingComplete
				}
			}
		}
	}(u.String())

	// seed the crawler
	c.q.ch <- &Resource{URL: u, URLString: u.String(), Depth: 1, Root: u}
	return nil
}

// enqueue adds work request to the queue after
// validating that the resource is a valid URL &
// that the robots.txt policy allows crawling it
func (c *Crawler) enqueue(resource *Resource) {
	// if queue is closed dont start new work
	if c.q.closed {
		return
	}

	if resource.URL == nil {
		return
	}

	worker, created := c.workers[resource.Root.String()]
	if !created {
		return
	}

	if worker.visited(resource.URL.String()) {
		return
	}

	if resource.Depth > worker.CrawlDepth() {
		return
	}

	if !worker.agent.Test(resource.URL.Path) {
		log.Printf("[ERROR] robots.txt policy does not allow path to be crawled: %v\n", resource.URL.String())
		return
	}

	req, err := http.NewRequest(http.MethodGet, resource.URL.String(), nil)
	if err != nil {
		return
	}

	req.Header.Add("User-Agent", c.UserAgent)
	worker.Add(1)

	// fetch resource
	go func(req *http.Request, resource *Resource) { c.fetch(req, resource) }(req, resource)
}

// isMIMETypeHTML makes an attempt to determine if the resource
// has a mime-type ~ text/html. when crawling web resources, not
// always you will encounter html mime-type content, but also other
// mime-types such as js, json, jpg, css, svg, mp{3,4} etc, which
// are not html documents and therefore these resouces cannot contain
// child resources defined by html tags such as <a href=... />
func (c *Crawler) isMIMETypeHTML(resource *Resource) (bool, error) {
	texthtml := "text/html"
	req, err := http.NewRequest(http.MethodHead, resource.URL.String(), nil)
	if err != nil {
		return false, err
	}

	req.Header.Add("User-Agent", c.UserAgent)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return false, err
	}

	defer resp.Body.Close()
	t, _, err := mime.ParseMediaType(resp.Header.Get("Content-type"))
	if err != nil {
		return false, err
	}

	return (t == texthtml), nil
}

// fetch makes a HTTPRequest using the provided HTTPRequest
// in addition to that, it will also enqueue child links as
// it finds them from the parent page; only links from the
// same domain are allowed to be enqueued to prevent going
// into an infinite loop with websites which cross-reference
// large media content sites such as youtube.com / reddit.com
func (c *Crawler) fetch(req *http.Request, resource *Resource) {
	worker, _ := c.workers[resource.Root.String()]
	defer worker.Done()
	defer func() { <-c.throttle }()
	c.throttle <- true

	// if queue is closed
	// dont start new work
	if c.q.closed {
		return
	}

	isHTML, err := c.isMIMETypeHTML(resource)
	if err != nil || !isHTML {
		return
	}

	if worker.status != StatusFetchingInProgress {
		worker.status = StatusFetchingInProgress
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return
	}

	defer resp.Body.Close()

	// add node to the leaf
	resource.HTTPStatusCode = resp.StatusCode
	resource.Title, _ = getTitleForPage(resp.Body)
	go func(resource *Resource) { c.append(resource) }(resource)

	links := collectlinks.All(resp.Body)

	if len(links) == 0 {
		return
	}

	for _, link := range links {
		absolute := normaliseURL(link, resource.URL)
		if absolute != nil {
			go func(absolute *url.URL, resource *Resource) {
				if c.q.closed {
					return
				}

				c.q.ch <- &Resource{
					URL:         absolute,
					Root:        resource.Root,
					URLString:   absolute.String(),
					Nodes:       make([]*Resource, 0),
					Parent:      append(resource.Parent, resource.URL.String()),
					Depth:       resource.Depth + 1,
					LastFetched: time.Now(),
				}
			}(absolute, resource)
		}
	}
}
