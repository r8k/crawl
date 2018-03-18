package crawler

// module deps
import "strings"
import "testing"
import "net/url"
import "io/ioutil"

// test NormaliseURL
func TestNormaliseURL(t *testing.T) {
	// execute test in parallel
	t.Parallel()

	relative := "/resource"
	baseURL, _ := url.Parse("http://example.com/sub")
	expected, _ := url.Parse("http://example.com/resource")
	absolute := normaliseURL(relative, baseURL)

	if absolute.String() != expected.String() {
		t.Fatalf("expected %v, got: %v\n", expected.String(), absolute.String())
	}
}

// test GetTitleForPage
func TestGetTitleForPage(t *testing.T) {
	// execute test in parallel
	t.Parallel()

	doc := ioutil.NopCloser(strings.NewReader(`<meta charset="UTF-8"><title>Example Title</title>`))
	title, err := getTitleForPage(doc)

	if err != nil || title != "Example Title" {
		t.Fatalf("expected Example Title, got: %v\n", title)
	}
}

// test NewCrawler
func TestNewCrawler(t *testing.T) {
	// execute test in parallel
	t.Parallel()

	c := New()
	defer c.Close()

	if c == nil {
		t.Fatalf("expected new crawler, got nil\n")
	}
}
