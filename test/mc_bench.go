package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
)

var host, method, topicName, lineName string
var port, testCount, concurrency int

func init() {
	flag.StringVar(&host, "h", "127.0.0.1", "hostname")
	flag.IntVar(&port, "p", 11211, "port")
	flag.IntVar(&concurrency, "c", 10, "concurrency level")
	flag.IntVar(&testCount, "n", 10000, "test count")
	flag.StringVar(&method, "m", "push", "test method")
	flag.StringVar(&topicName, "t", "StressTestTool", "topic to test")
	flag.StringVar(&lineName, "l", "Line", "line to test")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s -h host -p port -c concurrency -n count -m push -t topic -l line\n", os.Args[0])
		flag.PrintDefaults()
	}
}

func initQueue() {
	var mc *memcache.Client
	conn := fmt.Sprintf("%s:%d", host, port)
	mc = memcache.New(conn)

	err := mc.Add(&memcache.Item{Key: topicName, Value: []byte{}})
	if err != nil {
		log.Printf("add error: %v", err)
	}

	fullLineName := fmt.Sprintf("%s/%s", topicName, lineName)
	err = mc.Add(&memcache.Item{Key: fullLineName, Value: []byte{}})
	if err != nil {
		log.Printf("add error: %v", err)
	}
}

func setTestSingle(ch chan bool, cn, n int) {
	var err error
	var mc *memcache.Client
	conn := fmt.Sprintf("%s:%d", host, port)
	mc = memcache.New(conn)
	for i := 0; i < n; i++ {
		v := fmt.Sprintf("Value-c%d:%d", cn, i)
		start := time.Now()
		err = mc.Set(&memcache.Item{Key: topicName, Value: []byte(v)})
		if err != nil {
			log.Printf("set error: c%d %v", cn, err)
		} else {
			end := time.Now()
			duration := end.Sub(start).Seconds()
			log.Printf("set succ: %s - %s spend: %.3fms", topicName, v, duration*1000)
		}
	}
	ch <- true
}

func setTest(c, n int) {
	ch := make(chan bool)
	singleCount := n / c
	for i := 0; i < c; i++ {
		go setTestSingle(ch, i, singleCount)
	}
	for i := 0; i < c; i++ {
		select {
		case <-ch:
			log.Printf("set single succ: %s - c%d", topicName, i)
		}
	}
}

func getTestSingle(ch chan bool, cn, n int) {
	var mc *memcache.Client
	conn := fmt.Sprintf("%s:%d", host, port)
	key := fmt.Sprintf("%s/%s", topicName, lineName)
	mc = memcache.New(conn)
	for i := 0; i < n; i++ {
		start := time.Now()
		item, err := mc.Get(key)
		if err != nil {
			log.Printf("get error: c%d %v", cn, err)
		} else {
			end := time.Now()
			duration := end.Sub(start).Seconds()
			log.Printf("get succ: %s - %s spend: %.3fms", item.Key, string(item.Value), duration*1000)
		}
	}
	ch <- true
}

func getTest(c, n int) {
	ch := make(chan bool)
	singleCount := n / c
	for i := 0; i < c; i++ {
		go getTestSingle(ch, i, singleCount)
	}
	for i := 0; i < c; i++ {
		select {
		case <-ch:
			log.Printf("get single succ: %s/%s - c%d", topicName, lineName, i)
		}
	}
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()

	if method != "push" && method != "pop" {
		fmt.Printf("test method not supported!\n")
		return
	}

	now := time.Now()
	year := now.Year()
	month := now.Month()
	day := now.Day()
	hour := now.Hour()
	minute := now.Minute()
	second := now.Second()
	logName := fmt.Sprintf("uq_%s_%d-%d-%d_%d:%d:%d_c%d_n%d.log", method, year, month, day, hour, minute, second, concurrency, testCount)
	logfile, err := os.OpenFile(logName, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		fmt.Printf("%s\r\n", err.Error())
		os.Exit(-1)
	}
	defer logfile.Close()

	log.SetFlags(log.Lshortfile | log.LstdFlags | log.Lmicroseconds)
	log.SetOutput(logfile)

	initQueue()

	start := time.Now()
	if method == "push" {
		setTest(concurrency, testCount)
	} else if method == "pop" {
		getTest(concurrency, testCount)
	}
	end := time.Now()

	duration := end.Sub(start)
	dSecond := duration.Seconds()

	fmt.Printf("StressTest Done!")
	fmt.Printf("Spend: %.3fs Speed: %.3f msg/s", dSecond, float64(testCount)/dSecond)

	log.Printf("StressTest Done!")
	log.Printf("Spend: %.3fs Speed: %.3f msg/s", dSecond, float64(testCount)/dSecond)
	return
}