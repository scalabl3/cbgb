package main

import (
	"flag"
	"log"
	"time"

	"github.com/couchbaselabs/cbgb"
)

var mutationLogCh = make(chan interface{}, 1024)

var startTime = time.Now()

var addr = flag.String("addr", ":11211", "data protocol listen address")
var data = flag.String("data", "./tmp", "data directory")
var rest = flag.String("rest", ":DISABLED", "rest protocol listen address")
var staticPath = flag.String("static-path",
	"static", "path to static content")
var defaultBucketName = flag.String("default-bucket-name",
	cbgb.DEFAULT_BUCKET_NAME, "name of the default bucket; use \"\" for no default bucket")
var defaultNumPartitions = flag.Int("default-num-partitions",
	1, "default number of partitions for new buckets")
var defaultQuotaBytes = flag.Int64("default-quota-bytes",
	1000000, "default quota (max key+value bytes allowed) for new buckets")
var defaultMemoryOnly = flag.Int("default-memory-only",
	0, "default memory only level for new buckets"+
		" (0 = everything persisted"+
		"; 1 = item ops are not persisted"+
		"; 2 = nothing persisted)")

var buckets *cbgb.Buckets
var bucketSettings *cbgb.BucketSettings

func main() {
	flag.Parse()
	args := flag.Args()

	log.Printf("cbgb")
	flag.VisitAll(func(f *flag.Flag) {
		log.Printf("  %v=%v", f.Name, f.Value)
	})
	log.Printf("  %v", args)

	go cbgb.MutationLogger(mutationLogCh)

	var err error

	bucketSettings = &cbgb.BucketSettings{
		NumPartitions: *defaultNumPartitions,
		QuotaBytes:    *defaultQuotaBytes,
		MemoryOnly:    *defaultMemoryOnly,
	}
	buckets, err = cbgb.NewBuckets(*data, bucketSettings)
	if err != nil {
		log.Fatalf("error: could not make buckets: %v, data directory: %v", err, *data)
	}

	log.Printf("loading buckets from: %v", *data)
	err = buckets.Load()
	if err != nil {
		log.Fatalf("error: could not load buckets: %v, data directory: %v", err, *data)
	}

	if buckets.Get(*defaultBucketName) == nil &&
		*defaultBucketName != "" {
		_, err := createBucket(*defaultBucketName, bucketSettings)
		if err != nil {
			log.Fatalf("error: could not create default bucket: %s, err: %v",
				*defaultBucketName, err)
		}
	}

	if len(args) <= 0 || args[0] == "server" {
		mainServer(buckets, *defaultBucketName, *addr, *rest, *staticPath)
	}

	log.Fatalf("error: unknown command: %v", args[0])
}

func mainServer(buckets *cbgb.Buckets, defaultBucketName string,
	addr string, rest string, staticPath string) {
	log.Printf("listening data on: %v", addr)
	if _, err := cbgb.StartServer(addr, buckets, defaultBucketName); err != nil {
		log.Fatalf("error: could not start server: %s", err)
	}

	if rest != ":DISABLED" {
		restMain(rest, staticPath)
	}

	// Let goroutines do their work.
	select {}
}

func createBucket(bucketName string, bucketSettings *cbgb.BucketSettings) (
	cbgb.Bucket, error) {
	log.Printf("creating bucket: %v, numPartitions: %v",
		bucketName, bucketSettings.NumPartitions)

	bucket, err := buckets.New(bucketName, bucketSettings)
	if err != nil {
		return nil, err
	}

	bucket.Subscribe(mutationLogCh)

	for vbid := 0; vbid < bucketSettings.NumPartitions; vbid++ {
		bucket.CreateVBucket(uint16(vbid))
		bucket.SetVBState(uint16(vbid), cbgb.VBActive)
	}

	if err = bucket.Flush(); err != nil {
		return nil, err
	}

	return bucket, nil
}
