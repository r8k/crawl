package api

// module deps
import "mime"
import "net/url"
import "net/http"
import "github.com/labstack/echo"
import "github.com/r8k/crawl/crawler"

// Handler is used as api.handler
type Handler struct {
	Crawler *crawler.Crawler
}

// Domain struct for using in request & response
type Domain struct {
	Domain string               `json:"domain"`
	Depth  int                  `json:"depth,omitempty"`
	Status crawler.WorkerStatus `json:"status,omitempty"`
}

// HasContentType determines if http.Request has the content-type
func HasContentType(r *http.Request, mimetype string) (bool, error) {
	t, _, err := mime.ParseMediaType(r.Header.Get("Content-type"))
	if err != nil {
		return false, err
	}

	return (t == mimetype), nil
}

// CreateDomainHandler is the api.Handler to register domains
// for crawling. payload is expected in application/json format
// and is expected to include the domain and depth attributes
// below is a sample payload with their data types included
// { "domain": "http://cloudflare.com", "depth": 3 }
//
// domain - required, string
// depth  - int,      optional; defaults to 5
func (h *Handler) CreateDomainHandler(ctx echo.Context) error {
	var err error
	var isJSON bool
	var domain = new(Domain)

	isJSON, err = HasContentType(ctx.Request(), "application/json")
	if err != nil || !isJSON {
		return echo.NewHTTPError(http.StatusUnsupportedMediaType)
	}

	if err = ctx.Bind(domain); err != nil {
		ctx.Logger().Errorf("failed to unmarshal domain, %v\n", err)
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	err = h.Crawler.Crawl(domain.Domain, domain.Depth)
	if err != nil {
		ctx.Logger().Errorf("cannot initialise crawler; error: %v\n", err.Error())
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	domain.Status = crawler.StatusInitialised
	return ctx.JSON(http.StatusAccepted, domain)
}

// GetDomainHandler is the api.Handler to query domains crawl
// response tree and is expected to include the domain in the
// URL path parameter, such as /domains/https%3A%2F%2Fcloudflare.com
//
// as noted in the above example, the domain in the path parameter
// is expected to include protocol and be a URL-encoded string
// most libraries perform the URL encoding before sending the request
// over the wire, but in some cases the user is required to explicitly
// perform the encoding before making the request; examples for such
// utilities are cURL / libcurl
func (h *Handler) GetDomainHandler(ctx echo.Context) error {
	domain, err := url.PathUnescape(ctx.Param("domain"))
	if err != nil {
		ctx.Logger().Errorf("failed to unescape domain, %v\n", err)
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	worker := h.Crawler.Worker(domain)
	if worker == nil {
		return ctx.NoContent(http.StatusNotFound)
	}

	if worker.Status() != crawler.StatusFetchingComplete {
		return ctx.NoContent(http.StatusNoContent)
	}

	return ctx.JSON(http.StatusOK, []interface{}{worker.Tree})
}

// GetDomainStatusHandler is the api.Handler to query domains crawl
// response status and is expected to include the domain in the
// URL path parameter, e.g. /domains/https%3A%2F%2Fcloudflare.com/status
//
// as noted in the above example, the domain in the path parameter
// is expected to include protocol and be a URL-encoded string
// most libraries perform the URL encoding before sending the request
// over the wire, but in some cases the user is required to explicitly
// perform the encoding before making the request; examples for such
// utilities are cURL / libcurl
func (h *Handler) GetDomainStatusHandler(ctx echo.Context) error {
	domain, err := url.PathUnescape(ctx.Param("domain"))
	if err != nil {
		ctx.Logger().Errorf("failed to unescape domain, %v\n", err)
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	worker := h.Crawler.Worker(domain)
	if worker == nil {
		return ctx.NoContent(http.StatusNotFound)
	}

	status := &Domain{
		Domain: domain,
		Status: worker.Status(),
		Depth:  worker.CrawlDepth(),
	}

	return ctx.JSON(http.StatusOK, status)
}
