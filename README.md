# RSS Puppy
# This is a clone of the buzzfeed rss-puppy tool written in Go and requireing less heavy dependencies.
[Buzzfeed rss-puppy](https://github.com/buzzfeed-openlab/rss-puppy)
##### A watchdog tool for monitoring RSS feeds

This tool is designed to monitor RSS feeds in bulk, and to generate machine friendly notifications when new entries appear. While there exists no shortage of RSS readers and web-based notification services, nothing we found combines easy managment of hundreds of RSS feeds with the flexibility to direct output to a variety of data stores or over disparate protocols.

This monitor can be run on any cloud service provider, and only requires Go. Also, it is trivial to add output handlers which can pipe feed entry data to any service you use.

Read more about the [motivation](http://www.buzzfeed.com/westleyargentum/automated-journalism-that-works-with-journalists) and [design](http://www.buzzfeed.com/westleyargentum/your-very-own-rss-watchdog).

## How to run

### Get the code

- `git clone https://github.com/jesusrmoreno/rss-puppy/rss-puppy.git`
- `cd rss-puppy; go build *.go;

### Set up a database
This version of the monitor uses the ledisdb key value store to keep track of feeds and entries and does not
need to have a database set up. By default it stores this database in a folder called var.
You can pass the flag --dbpath to tell it where to store the database.

### Configure your feeds
In the config file there will be a section called `"feeds"` and a section called `"throttling"`.

```toml
exitOnError = true
feeds = [
  "https://www.sec.gov/cgi-bin/browse-edgar?action=getcompany&CIK=0001440512&type=&dateb=&owner=exclude&start=0&count=40&output=atom",
  "https://www.sec.gov/cgi-bin/browse-edgar?action=getcompany&CIK=0001326801&type=&dateb=&owner=exclude&start=0&count=40&output=atom",
  "https://www.sec.gov/cgi-bin/browse-edgar?action=getcompany&CIK=0001652044&type=&dateb=&owner=exclude&start=0&count=40&output=atom",
  "https://www.sec.gov/cgi-bin/browse-edgar?action=getcompany&CIK=0000320193&type=&dateb=&owner=exclude&start=0&count=40&output=atom",
  "https://www.sec.gov/cgi-bin/browse-edgar?action=getcompany&CIK=0001018724&type=&dateb=&owner=exclude&start=0&count=40&output=atom",
  "https://www.sec.gov/cgi-bin/browse-edgar?action=getcompany&CIK=0000936468&type=&dateb=&owner=exclude&start=0&count=40&output=atom",
]
[throttling]
  concurrentInterval = 1000
  maxConcurrent = 10
  monitorFrequency = 8000
  oldFeedThreshold = 30

```

- `"feeds"` is an array of RSS feed urls that will be monitored.
- `"throttling"` is broken into several parts
	- `"monitorFrequency"`: How often the monitor will check to see if it needs to query any "old" RSS feeds (ie: ones that haven't been queried in awhile).
	- `"maxConcurrent"`: The maximum number of concurrent queries the monitor will make (excess queries will be queued).
	- `"concurrentInterval"`: The interval to wait between making `"maxConcurrent"` queries (ie: X queries per 10 seconds, or X queries per 60 seconds).

concurrentInterval is still being implemented

### Configure your outputs
Outputs are modules of code that listen for events that the monitor emits and do something useful with the resulting data.

There are several different kinds of events:

- `"new-entry"`: Emitted when the monitor encounters an entry that it has not seen before. Handlers will be invoked with the `entry` as a json object, and the `feed` url as a string.
- `"checking-old-feeds"`: Emitted whenever the monitor wakes up to look for feeds to query (approx every `"monitorFrequency"` seconds).
- `"old-feed"`: Emitted whenever the monitor finds a feed that hasn't been queried in awhile and needs to be checked. Handlers will be invoked with the `feed` url as a string.
- `"entry"`: Emitted whenever an entry is parsed from a feed. Note that feeds will be queried and parsed over and over again, so this will be emitted for the same entry many times. Handlers will be called with the `entry` as a json object and the `feed` url as a string.

rss-puppy must be rebuilt every time a new output is added.
The logger.go file shows a very very simple example of an output.


### Run the monitor!

```bash
./rss-puppy --dbpath examplepath
```
