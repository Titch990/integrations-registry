// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"flag"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"

	"github.com/stretchr/testify/assert"
)

var (
	generateFlag  = flag.Bool("generate", false, "Write golden files")
	testCacheTime = "1"
)

func TestEndpoints(t *testing.T) {

	packagesBasePath := "./testdata/package"

	tests := []struct {
		endpoint string
		path     string
		file     string
		handler  func(w http.ResponseWriter, r *http.Request)
	}{
		{"/", "", "info.json", catchAll("./testdata", testCacheTime)},
		{"/search", "/search", "search.json", searchHandler(packagesBasePath, testCacheTime)},
		{"/categories", "/categories", "categories.json", categoriesHandler(packagesBasePath, testCacheTime)},
		{"/search?kibana=6.5.2", "/search", "search-kibana652.json", searchHandler(packagesBasePath, testCacheTime)},
		{"/search?kibana=7.2.1", "/search", "search-kibana721.json", searchHandler(packagesBasePath, testCacheTime)},
		{"/search?category=metrics", "/search", "search-category-metrics.json", searchHandler(packagesBasePath, testCacheTime)},
		{"/search?category=logs", "/search", "search-category-logs.json", searchHandler(packagesBasePath, testCacheTime)},
		{"/search?package=example", "/search", "search-package-example.json", searchHandler(packagesBasePath, testCacheTime)},
		{"/search?internal=true", "/search", "search-package-internal.json", searchHandler(packagesBasePath, testCacheTime)},
		{"/package/example-1.0.0", "", "package.json", catchAll("./testdata", testCacheTime)},
	}

	for _, test := range tests {
		t.Run(test.endpoint, func(t *testing.T) {
			runEndpoint(t, test.endpoint, test.path, test.file, test.handler)
		})
	}
}

func runEndpoint(t *testing.T, endpoint, path, file string, handler func(w http.ResponseWriter, r *http.Request)) {
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	router := mux.NewRouter()
	if path == "" {
		router.PathPrefix("/").HandlerFunc(handler)
	} else {
		router.HandleFunc(path, handler)
	}
	req.RequestURI = endpoint
	router.ServeHTTP(recorder, req)

	fullPath := "./docs/api/" + file

	if *generateFlag {
		err = ioutil.WriteFile(fullPath, recorder.Body.Bytes(), 0644)
		if err != nil {
			t.Fatal(err)
		}
	}

	data, err := ioutil.ReadFile(fullPath)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, string(data), recorder.Body.String())
	assert.Equal(t, recorder.Header()["Cache-Control"], []string{"max-age=" + testCacheTime, "public"})
}
