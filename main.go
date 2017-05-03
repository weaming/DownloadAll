package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/http"
	urllib "net/url"
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
var countIgnore = 0
var outdir = "./Downloads"
var fullName = false
var outPrefix = ""
var outSuffix = ""
var checkExistDirs arrayFlags

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

// array flags: {{

type arrayFlags []string

func (i *arrayFlags) String() string {
	return "hello"
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

// }}

func main() {
	flag.StringVar(&outdir, "o", outdir, "Directory to save files")
	flag.BoolVar(&fullName, "full", fullName, "Whether to use URL path replaced slash(/) by - as the saved file name")
	flag.StringVar(&outPrefix, "p", outPrefix, "Add prefix to saved file name")
	flag.StringVar(&outSuffix, "s", outSuffix, "Add suffix to saved file name")
	flag.Var(&checkExistDirs, "d", "Optional extra directories to check whether file existed")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s FILE\n\nThe FILE is the text files contains URLs line by line.\n\n", os.Args[0])
		flag.PrintDefaults()
	}
	// call Parse() first!
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
		log.Printf("Downloaded %v files took %s, ignored %v", count, elapsed, countIgnore)
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
	countScanned := 0
	for scanner.Scan() {
		url := strings.Trim(scanner.Text(), " \r\n\t")
		if strings.HasPrefix(url, "#") || url == "" {
			continue
		} else {
			countScanned += 1
			if countScanned%500 == 0 {
				log.Printf("Scanned count: %v", countScanned)
			}
		}

		// add to wait group and HTTP pool
		wg.Add(1)
		pool <- 1

		// file name
		outName := ""
		if fullName {
			u, err := urllib.Parse(url)
			if err != nil {
				log.Printf("Parse URL failed: %v", url)
				continue
			}
			outName = strings.Replace(u.Path, "/", "-", -1)
			outName = strings.Trim(outName, "-")
		} else {
			outName = fp.Base(url)
		}
		if !strings.HasPrefix(outName, outPrefix) {
			outName = outPrefix + outName
		}
		if !strings.HasSuffix(outName, outSuffix) {
			outName += outSuffix
		}

		// go to download it!
		go downloadImage(url, outdir, outName)
	}

	// wait all goroutine to finish
	wg.Wait()
	fatal(scanner.Err())
}

func downloadImage(url, outDir, outName string) {
	defer func() {
		<-pool
		wg.Done()
	}()

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in downloadImage(): ", r)
		}
	}()

	out := fp.Join(outDir, outName)
	var _, err = os.Stat(out)
	if err == nil {
		log.Printf("Ignore existed: %v => %v\n", url, out)
		countIgnore += 1
		return
	} else {
		// check multiple directories: {{
		for _, dir := range checkExistDirs {
			tmp := fp.Join(dir, outName)
			var _, err = os.Stat(tmp)
			if err == nil {
				log.Printf("Ignore existed: %v => %v\n", url, tmp)
				countIgnore += 1
				return
			}
		}
		// }}
		defer log.Printf("%v => %v\n", url, out)
	}

	multiRangeDownload(url, out)
}

func multiRangeDownload(url, out string) {
	outfile, err := os.Create(out)
	if err != nil {
		log.Println(err)
		return
	}
	defer outfile.Close()

	FileDownloader, err := NewFileDownloader(url, outfile, -1)
	if err != nil {
		log.Println(err)
		return
	}

	var _wg sync.WaitGroup
	var exit = make(chan bool)
	FileDownloader.OnFinish(func() {
		exit <- true
		count += 1
	})

	FileDownloader.OnError(func(errCode int, err error) {
		log.Println(errCode, err)
	})

	FileDownloader.OnStart(func() {
		for {
			//status := FileDownloader.GetStatus()
			//var i = float64(status.Downloaded) / float64(FileDownloader.Size) * 50
			//h := strings.Repeat("=", int(i)) + strings.Repeat(" ", 50-int(i))

			select {
			case <-exit:
				//format := "%v/%v [%s] %v byte/s %v\n"
				//fmt.Printf(format, status.Downloaded, FileDownloader.Size, h, 0, "[FINISH]")
				_wg.Done()
			default:
				time.Sleep(time.Second * 1)
				os.Stdout.Sync()
			}
		}
	})

	_wg.Add(1)
	FileDownloader.Start()
	_wg.Wait()
}
