package main

// module deps
import "os"
import "fmt"
import "flag"
import "time"
import "context"
import "syscall"
import "os/signal"
import "html/template"
import "github.com/labstack/echo"
import "github.com/r8k/crawl/api"
import "github.com/r8k/crawl/crawler"
import "github.com/labstack/echo/middleware"

// module constants
const version = "0.1.1"
const usage = `gocrawler v%s
Usage:
  gocrawler -p 8080 -a 127.0.0.1
  gocrawler -h | -help
  gocrawler -v | -version
`

// flag variables
var bindAddress = flag.String("a", "127.0.0.1", "server bind address")
var bindPort = flag.String("p", "8080", "server bind port to listen")
var fHelp = flag.Bool("h", false, "show help")
var fVers = flag.Bool("v", false, "show version")

func showUsage() {
	flag.Usage()
	os.Exit(1)
}

func showVersion() {
	fmt.Println(version)
	os.Exit(1)
}

// main entry point
func main() {
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, fmt.Sprintf(usage, version))
	}
	flag.Parse()

	if *fHelp {
		showUsage()
	}

	if *fVers {
		showVersion()
	}

	// start crawler
	srvaddr := fmt.Sprintf("%s:%s", *bindAddress, *bindPort)

	// create api handler
	handler := &api.Handler{
		Crawler: crawler.New(),
	}

	// swagger template
	t := &Template{
		templates: template.Must(
			template.ParseGlob("swagger/swagger.yaml"),
		),
	}

	// create api router
	e := echo.New()
	e.Renderer = t
	e.HideBanner = true

	// add middleware to router
	e.Use(middleware.Logger())
	e.Use(middleware.RequestID())
	e.Use(middleware.Recover())
	e.Logger.SetPrefix("gocrawler")

	// CORS middleware
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{echo.GET, echo.POST},
	}))

	// register api handlers
	e.GET("/docs", swagger)
	e.GET("/swagger.yaml", renderSwagger)
	e.POST("/api/domains", handler.CreateDomainHandler)
	e.GET("/api/domains/:domain", handler.GetDomainHandler)
	e.GET("/api/domains/:domain/status", handler.GetDomainStatusHandler)

	// start api server
	go func() {
		if err := e.Start(srvaddr); err != nil {
			e.Logger.Info("shutting down the crawler http server")
		}
	}()

	// wait for interrupt signal to shutdown the server with a timeout
	interrupt := make(chan os.Signal)
	signal.Notify(interrupt, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)

	<-interrupt
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}

	handler.Crawler.Close()
}
