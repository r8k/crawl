## gocrawler [1]

`gocrawler` is an API layer, to crawl domains. The Crawler adds a domain into a `worker` queue configured with a given `depth`, so that the crawling is stopped after the `depth`. Crawling is restricted to the same domain, since crawling external domains (in addition to the requested domain) can go into a ~infinite loop, for e.g. when the crawling request is received for `https://google.com`, any child links outside of `google.com` are not added back to the task queue.

 * `gocrawler` is concurrent safe, utilises [goroutines](https://gobyexample.com/goroutines) to achieve concurrency
 * `gocrawler` uses [Channels](https://gobyexample.com/channels) to pass references to data between goroutines
 * `gocrawler` uses [Channels](https://gobyexample.com/channels) to achieve [throttled](https://github.com/golang/go/wiki/RateLimiting) concurrency
 * uses `robots.txt` & adheres to the policies of robots.txt exclusion [standard](https://en.wikipedia.org/wiki/Robots_exclusion_standard)

## Getting started

* [Pre-requisites](#pre-requisites)
* [Quick Start](#quick-start)

## Pre-requisites

This readme is prepared for OSX & it works mostly for Linux as well. However, if you are on Windows OS these instructions might vary significantly, the extent of which I'm not sure because I do not have a Windows machine to test these instructions.

 * [Golang](https://golang.org/doc/install) is needed to build `gocrawler`. Steps to install Go can be found [here](https://golang.org/doc/install).
 * [GNU Make](https://www.gnu.org/software/make/). If you are on OSX, it comes with `make`, but if you are on a different OS, please consult this [link](https://www.gnu.org/software/make/) for installation.

## Quick Start

Quickstart to build and run `gocrawler`. All instructions below assume that you are in the directory of `gocrawler` and have the [Pre-requisites](#pre-requisites) installed.

```shell
# run tests and make binary
make
```

Now let's start `gocrawler`

```shell
./gocrawler -a 127.0.0.1 -p 8080
```

Accessing `help` is just an argument away

```shell
./gocrawler -h
```

## API Docs

API Docs are available at `http://127.0.0.1:8080/docs`, assuming that you have started `gocrawler` with flags `-a 127.0.0.1 -p 8080`

## Testing

To run tests

```shell
make test
```
