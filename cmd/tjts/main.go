package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"time"

	"github.com/lstoll/tjts"
)

// TripleJURL is HARDCODE
const TripleJURL = "http://live-radio01.mediahubaustralia.com/2TJW/aac/"
const DoubleJURL = "http://live-radio02.mediahubaustralia.com/DJDW/aac/"

func main() {
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, os.Interrupt)
	signal.Notify(sc, os.Kill)

	cachePath := os.Getenv("CACHE_PATH")
	strCacheInterval := os.Getenv("CACHE_INTERVAL")
	cacheInterval := 10 * time.Minute
	if strCacheInterval != "" {
		parsed, err := time.ParseDuration(strCacheInterval)
		if err != nil {
			log.Fatalf("Error parsing CACHE_INTERVAL=%s: %q", strCacheInterval, err)
		}
		cacheInterval = parsed
	}
	host := os.Getenv("HOST")
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	cht := 2 * time.Second

	tc := tjts.NewClient(TripleJURL, cht)
	tchd := make(chan []byte, 512)
	go func() {
		tc.Start(tchd)
	}()

	dc := tjts.NewClient(DoubleJURL, cht)
	dchd := make(chan []byte, 512)
	go func() {
		dc.Start(dchd)
	}()

	var tjCache, djCache string
	if cachePath != "" {
		tjCache = cachePath + "/triplej.stream.cache"
		djCache = cachePath + "/doublej.stream.cache"
	}

	tsh := tjts.NewMemShifter(tchd, cht, 20*time.Hour, tjCache, cacheInterval)
	dsh := tjts.NewMemShifter(dchd, cht, 20*time.Hour, djCache, cacheInterval)
	s := tjts.NewServer()
	s.AddEndpoint("doublej", dsh)
	s.AddEndpoint("triplej", tsh)
	go func() {
		for range sc {
			log.Print("Shutdown requested")
			tsh.Shutdown()
			dsh.Shutdown()
			os.Exit(0)
		}
	}()

	s.ListenAndServe(net.JoinHostPort(host, port))
}
