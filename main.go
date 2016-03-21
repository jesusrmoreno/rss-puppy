package main

import (
	"bufio"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/SlyMarbo/rss"
	"github.com/chuckpreslar/emission"
	"github.com/codegangsta/cli"
	lediscfg "github.com/siddontang/ledisdb/config"
	"github.com/siddontang/ledisdb/ledis"
)

// Event name constants
const (
	ErrorEvent            = "error"
	EntryEvent            = "entry"
	FeedParsedEvent       = "feed-parsed"
	CheckingOldFeedsEvent = "checking-old-feeds"
	OldFeedEvent          = "old-feed"
	NewEntryEvent         = "new-entry"
)

// Emitter serves as the global emitter that all outputs will listen to..
var Emitter *emission.Emitter

type config struct {
	ExitOnError bool
	DBPath      string
	Feeds       []string
	Throttling  struct {
		ConcurrentInterval int64
		MaxConcurrent      int64
		MonitorFrequency   time.Duration
		OldFeedThreshold   int64
	}
}

// Context ...
type Context struct {
	Config     *config
	DB         *ledis.DB
	emitter    *emission.Emitter
	QueryQueue chan string
}

// Entry ...
type Entry struct {
	ID    string
	Feed  string
	Title string
	Date  time.Time
	Link  string
}

func (e *Entry) asByteSlice() ([]byte, error) {
	val, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}
	return val, nil
}

func readConfig(f string) *config {
	if _, err := os.Stat(f); err != nil {
		log.Fatal("Missing Config File")
	}
	conf := config{}
	if _, err := toml.DecodeFile(f, &conf); err != nil {
		log.Fatal(err)
	}
	return &conf
}

func (ctx *Context) oldFeeds() {
	ctx.emitter.Emit(CheckingOldFeedsEvent)
	for _, feed := range ctx.Config.Feeds {
		value, err := ctx.DB.Get([]byte(feed))
		if err != nil {
			ctx.emitter.Emit(ErrorEvent, err)
			if ctx.Config.ExitOnError {
				log.Fatal(err)
			}
		}
		if value != nil {
			continue
		}
		ctx.emitter.Emit(OldFeedEvent, feed)
	}
}

func (ctx *Context) updateTimestamp(feed string) {
	ctx.DB.Set([]byte(feed), []byte(feed))
	ctx.DB.Expire([]byte(feed), ctx.Config.Throttling.OldFeedThreshold)
}

func (ctx *Context) parseFeed(feedURL string) {
	rss.CacheParsedItemIDs(false)
	feed, err := rss.Fetch(feedURL)
	if err != nil {
		ctx.emitter.Emit(ErrorEvent, err)
		if ctx.Config.ExitOnError {
			log.Fatal(err)
		}
	}
	for _, item := range feed.Items {
		e := Entry{
			Title: item.Title,
			ID:    item.ID,
			Date:  item.Date,
			Link:  item.Link,
			Feed:  feedURL,
		}
		dbVal, err := ctx.DB.Get([]byte(e.ID))
		if err != nil {
			ctx.emitter.Emit(ErrorEvent, err)
			if ctx.Config.ExitOnError {
				log.Fatal(err)
			}
		}
		if dbVal != nil {
			continue
		} else {
			ctx.emitter.Emit(EntryEvent, e)
		}
	}
	ctx.emitter.Emit(FeedParsedEvent, feedURL)
}

func (ctx *Context) persistEntry(e Entry) {
	val, err := e.asByteSlice()
	if err != nil {
		ctx.emitter.Emit(ErrorEvent, err)
		if ctx.Config.ExitOnError {
			log.Fatal(err)
		}
	}
	ctx.DB.Set([]byte(e.ID), val)
	ctx.emitter.Emit(NewEntryEvent, e)
}

// Setup ...
func (ctx *Context) Setup() {
	for i := 0; i < cap(ctx.QueryQueue); i++ {
		ctx.QueryQueue <- "null"
	}
	ctx.emitter.On(OldFeedEvent, func(feed string) {
		ctx.QueryQueue <- feed
	})
	ctx.emitter.On(FeedParsedEvent, ctx.updateTimestamp)
	ctx.emitter.On(EntryEvent, ctx.persistEntry)
}

func (ctx *Context) run() {
	for {
		feedURL := <-ctx.QueryQueue
		if feedURL != "null" {
			go ctx.parseFeed(feedURL)
		}
	}
}

// Logger ...
func Logger() {
	Emitter.On(CheckingOldFeedsEvent, func() {
		log.Println("checking for out of date feeds...")
	})
	Emitter.On(OldFeedEvent, func(feed string) {
		log.Println("needs checking:", feed)
	})
	Emitter.On(NewEntryEvent, func(e Entry) {
		log.Println(" > new entry on feed", e.Feed, "with id", e.ID)
	})
}

func runApp(c *cli.Context) {
	conf := readConfig("config.toml")
	cfg := lediscfg.NewConfigDefault()
	dbPath := c.String("dbpath")
	cfg.DBPath = dbPath
	l, err := ledis.Open(cfg)
	if err != nil {
		log.Fatal(err)
	}
	db, err := l.Select(0)
	if err != nil {
		log.Fatal(err)
	}
	if c.Bool("destroy-db") {
		scanner := bufio.NewScanner(os.Stdin)
		log.Println("Warning! This will clear the entire database and cannot be undone")
		log.Println("Are you sure you want to continue? y/N")
		for scanner.Scan() {
			answer := scanner.Text()
			if answer == "y" {
				log.Println("Deleting all data")
				db.FlushAll()
				log.Println("Deleted all data")
				os.Exit(0)
			} else {
				log.Println("Did not delete any data")
				os.Exit(0)
			}
		}

	}
	Emitter = emission.NewEmitter()
	Emitter.SetMaxListeners(-1)
	ctx := Context{
		Config:     conf,
		DB:         db,
		emitter:    Emitter,
		QueryQueue: make(chan string, conf.Throttling.MaxConcurrent),
	}
	ctx.Setup()
	go ctx.run()
	Logger()
	for {
		ctx.oldFeeds()
		time.Sleep(time.Millisecond * ctx.Config.Throttling.MonitorFrequency)
	}
}

func main() {
	app := cli.NewApp()
	app.Name = "rss-puppy"
	app.Usage = "A watchdog tool for monitoring RSS feeds"
	app.Action = runApp
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "dbpath",
			Value: "",
			Usage: "Where to store the ledisdb database",
		},
		cli.BoolFlag{
			Name:  "destroy-db",
			Usage: "Will wipe the ledis database",
		},
	}
	app.Run(os.Args)
}
