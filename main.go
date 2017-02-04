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
	"time"
)

var count = 0

func fatal(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		log.Printf("Downloaded %v files took %s", count, elapsed)
	}()

	flag.Parse()
	infile := flag.Arg(0)
	outdir := "./Downloads"

	if infile == "" {
		fmt.Fprintf(os.Stderr, "Please give the pictures url file(one url each line)")
	}
	if flag.Arg(1) != "" {
		outdir = flag.Arg(1)
	}

	// create file if not exists
	var _, err = os.Stat(outdir)
	if os.IsNotExist(err) {
		err := os.MkdirAll(outdir, 0755)
		if err != nil {
			panic(err)
		}
	}

	file, err := os.Open(infile)
	fatal(err)
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		url := strings.Trim(scanner.Text(), " \r\n\t")
		if strings.HasPrefix(url, "#") || url == "" {
			continue
		}
		downloadImage(url, fp.Join(outdir, fp.Base(url)))
	}

	fatal(scanner.Err())
}

func downloadImage(url, out string) {

	var _, err = os.Stat(out)
	if err == nil {
		log.Printf("Ignore existed: %v => %v\n", url, out)
		return
	} else {
		log.Printf("%v => %v\n", url, out)
	}

	resp, err := http.Get(url)
	defer resp.Body.Close()

	if err != nil {
		log.Fatal("Trouble making GET photo request!")
	}

	contents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("Trouble reading reesponse body!")
	}

	err = ioutil.WriteFile(out, contents, 0644)
	if err != nil {
		log.Fatal("Trouble creating file: ", err)
	} else {
		count += 1
	}
}
