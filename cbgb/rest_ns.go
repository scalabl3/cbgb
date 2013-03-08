package main

import (
	"log"
	"net/http"

	"github.com/couchbaselabs/go-couchbase"
	"github.com/gorilla/mux"
)

var toplevelPool = couchbase.Pools{
	ImplementationVersion: "1.0-cbgb",
	IsAdmin:               false,
	UUID:                  "abc",
	Pools: []couchbase.RestPool{
		{
			Name:         "default",
			StreamingURI: "/poolsStreaming/default",
			URI:          "/pools/default",
		},
	}}

func notImplemented(w http.ResponseWriter, r *http.Request) {
	log.Printf("Request for %v:%v", r.Method, r.URL.Path)
	http.Error(w, "Not implemented", 501)
}

func restNSPools(w http.ResponseWriter, r *http.Request) {
	jsonEncode(w, &toplevelPool)
}

func getNSNodeList() []couchbase.Node {
	return []couchbase.Node{
		couchbase.Node{
			ClusterCompatibility: 131072,
			ClusterMembership:    "active",
			CouchAPIBase:         "http://localhost:8077/", // XXX: FIXTERMINATE
			Hostname:             "127.0.0.1:8091",         // XXX: FIXTERMINATE
			Ports:                map[string]int{"direct": 11211},
			Status:               "healthy",
			Version:              "1.0.0-cbgb",
		},
	}
}

func restNSPoolsDefault(w http.ResponseWriter, r *http.Request) {
	jsonEncode(w, map[string]interface{}{
		"buckets": map[string]interface{}{"uri": "/pools/default/buckets"},
		"name":    "default",
		"nodes":   getNSNodeList(),
		"stats":   map[string]interface{}{"uri": "/pools/default/stats"},
	})
}

func restNSBucket(w http.ResponseWriter, r *http.Request) {
	bucketName := mux.Vars(r)["bucketname"]
	rv := couchbase.Bucket{
		AuthType:     "sasl",
		Capabilities: []string{"couchapi"},
		Type:         "membase",
		Name:         bucketName,
		NodeLocator:  "vbucket",
		Nodes:        getNSNodeList(),
		Replicas:     1,
		URI:          "/pools/default/buckets/" + bucketName,
	}
	rv.VBucketServerMap.HashAlgorithm = "CRC"
	rv.VBucketServerMap.NumReplicas = 1
	rv.VBucketServerMap.ServerList = []string{"127.0.0.1:11211"} // XXX: me
	rv.VBucketServerMap.VBucketMap = [][]int{{0}}

	jsonEncode(w, &rv)
}

func restNSAPI(r *mux.Router) {

	ns_server_paths := []string{
		"/pools/default/buckets/{bucketname}/statsDirectory",
		"/pools/default/buckets/{bucketname}/stats",
		"/pools/default/buckets/{bucketname}/nodes",
		"/pools/default/buckets/{bucketname}/nodes/{node}/stats",
		"/pools/default/buckets/{bucketname}/ddocs",
		"/pools/default/buckets/{bucketname}/localRandomKey",
		"/pools/default/bucketsStreaming/{bucketname}",
		"/pools/default/stats",
		"/pools/default/buckets",
		"/poolsStreaming",
	}

	// Init the 501s from above
	for _, p := range ns_server_paths {
		r.HandleFunc(p, notImplemented).Methods("GET")
	}

	r.HandleFunc("/pools", restNSPools)
	r.HandleFunc("/pools/default", restNSPoolsDefault)
	r.HandleFunc("/pools/default/buckets/{bucketname}", restNSBucket)
}
