package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	//"strings"
	fp "path/filepath"
)

func fatal(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
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
		url := scanner.Text()
		downloadImage(url, fp.Join(outdir, fp.Base(url)))
	}

	fatal(scanner.Err())
}

func downloadImage(url, out string) {
	fmt.Printf("%v => %v\n", url, out)

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
	}
}
