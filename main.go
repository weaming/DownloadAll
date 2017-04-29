package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	fp "path/filepath"
	"strings"
	"sync"
	"time"
)

// HTTP GET timeout
const TIMEOUT = 20

// HTTP concurrence pool size
const CLIENT_POOL = 20

var pool = make(chan int, CLIENT_POOL)
var wg sync.WaitGroup
var count = 0
var outdir = "./Downloads"
var outPrefix = ""
var outSuffix = ""

var client = &http.Client{
	Transport: &http.Transport{
		MaxIdleConnsPerHost: 30,
	},
	Timeout: TIMEOUT * time.Second,
}

func fatal(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	flag.StringVar(&outdir, "o", outdir, "Directory to save files")
	flag.StringVar(&outPrefix, "p", outPrefix, "Add prefix to saved file name")
	flag.StringVar(&outSuffix, "s", outSuffix, "Add suffix to saved file name")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s FILE\n\nThe FILE is the text files contains URLs line by line.\n\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	infile := flag.Arg(0)
	if infile == "" {
		fmt.Fprintf(os.Stderr, "Please give the pictures url file(one url each line)\n")
		os.Exit(1)
	}
	if flag.Arg(1) != "" {
		outdir = flag.Arg(1)
	}

	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		log.Printf("Downloaded %v files took %s", count, elapsed)
	}()

	// create file if not exists
	var _, err = os.Stat(outdir)
	if os.IsNotExist(err) {
		err := os.MkdirAll(outdir, 0755)
		if err != nil {
			panic(err)
		}
	}

	infd, err := os.Open(infile)
	fatal(err)
	defer infd.Close()

	scanner := bufio.NewScanner(infd)
	for scanner.Scan() {
		url := strings.Trim(scanner.Text(), " \r\n\t")
		if strings.HasPrefix(url, "#") || url == "" {
			continue
		}

		// add to wait group and HTTP pool
		wg.Add(1)
		pool <- 1

		// file name
		outName := fp.Base(url)
		if !strings.HasPrefix(outName, outPrefix) {
			outName = outPrefix + outName
		}
		if !strings.HasSuffix(outName, outSuffix) {
			outName += outSuffix
		}

		// go to download it!
		go downloadImage(url, fp.Join(outdir, outName))
	}

	// wait all goroutine to finish
	wg.Wait()
	fatal(scanner.Err())
}

func downloadImage(url, out string) {
	defer func() {
		<-pool
		wg.Done()
	}()

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in downloadImage(): ", r)
		}
	}()

	var _, err = os.Stat(out)
	if err == nil {
		log.Printf("Ignore existed: %v => %v\n", url, out)
		return
	} else {
		defer log.Printf("%v => %v\n", url, out)
	}

	resp, err := client.Get(url)

	if resp.Body != nil {
		defer resp.Body.Close()
	}

	if err != nil {
		log.Println("Trouble making GET photo request!")
		return
	}

	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Trouble reading reesponse body!")
		return
	}

	err = ioutil.WriteFile(out, contents, 0644)
	if err != nil {
		log.Println("Trouble creating file!")
		return
	}
	count += 1
}
