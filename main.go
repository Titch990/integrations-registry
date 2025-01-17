// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/elastic/package-registry/util"

	ucfgYAML "github.com/elastic/go-ucfg/yaml"

	"github.com/gorilla/mux"
)

const (
	packageDir = "package"
)

var (
	packagesBasePath string
	address          string
	configPath       = "config.yml"

	// Cache times for the different endpoints
	searchCacheTime     = strconv.Itoa(60 * 60)      // 1 hour
	categoriesCacheTime = strconv.Itoa(60 * 60)      // 1 hour
	catchAllCacheTime   = strconv.Itoa(24 * 60 * 60) // 24 hour
)

func init() {
	flag.StringVar(&address, "address", "localhost:8080", "Address of the package-registry service.")
}

type Config struct {
	PublicDir string `config:"public_dir"`
}

func main() {
	flag.Parse()
	log.Println("Package registry started.")
	defer log.Println("Package registry stopped.")

	config, err := getConfig()
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}

	packagesBasePath := config.PublicDir + "/" + packageDir

	// Prefill the package cache
	packages, err := util.GetPackages(packagesBasePath)
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}
	log.Printf("%v package manifests loaded into memory.\n", len(packages))

	server := &http.Server{Addr: address, Handler: getRouter(*config, packagesBasePath)}

	go func() {
		err := server.ListenAndServe()
		if err != nil {
			log.Printf("Error serving: %s", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx := context.TODO()
	if err := server.Shutdown(ctx); err != nil {
		log.Print(err)
	}
}

func getConfig() (*Config, error) {
	cfg, err := ucfgYAML.NewConfigWithFile(configPath)
	if err != nil {
		return nil, err
	}

	config := &Config{}
	err = cfg.Unpack(config)
	if err != nil {
		return nil, err
	}

	return config, nil
}

func getRouter(config Config, packagesBasePath string) *mux.Router {
	router := mux.NewRouter().StrictSlash(true)

	router.HandleFunc("/search", searchHandler(packagesBasePath, searchCacheTime))
	router.HandleFunc("/categories", categoriesHandler(packagesBasePath, categoriesCacheTime))
	router.PathPrefix("/").HandlerFunc(catchAll(config.PublicDir, catchAllCacheTime))

	return router
}
