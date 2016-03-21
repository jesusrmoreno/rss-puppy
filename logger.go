package main

import "log"

// Logger Serves as an example for implementing handlers on the exposed events.
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
