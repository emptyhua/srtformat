package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/dimchansky/utfbom"
	"github.com/saintfish/chardet"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/transform"
)

func decodeToUTF8(bs []byte, enc encoding.Encoding) ([]byte, error) {
	buf := bytes.NewBuffer(bs)
	r := transform.NewReader(buf, enc.NewDecoder())
	return ioutil.ReadAll(r)
}

const (
	srtExpectNum = iota
	srtExpectTime
	srtExpectText
)

type SubLine struct {
	Index int
	Start string
	End   string
	Text  string
}

func formatSrt(bs []byte) ([]byte, error) {
	indexNum := 0
	lineNum := 0
	stage := srtExpectNum
	// convert 0: 1: 2,342 -->  0: 1: 5,334 to 00:01:02,342 --> 00:01:05,334
	timeReg := regexp.MustCompile(`^(\d+):\s*([\d]+):\s*([\d]+),\s*(\d+)\s+-->\s+\s*([\d]+):\s*([\d]+):\s*([\d]+),\s*(\d+)$`)

	subLines := []*SubLine{}
	var subLine *SubLine

	r := bytes.NewBuffer(bs)
	scanner := bufio.NewScanner(utfbom.SkipOnly(r))
	for scanner.Scan() {
		lineNum += 1
		line := strings.TrimSpace(scanner.Text())
		switch stage {
		case srtExpectNum:
			{
				_, err := strconv.Atoi(line)
				if err != nil {
					log.Fatal("line:", lineNum, " expect number: ", err)
				}
				indexNum += 1
				subLine = &SubLine{Index: indexNum}
				stage = srtExpectTime
			}
		case srtExpectTime:
			{
				m := timeReg.FindStringSubmatch(line)
				if len(m) != 9 {
					log.Fatal("line:", lineNum, " invalid time format: ", line)
				}
				subLine.Start = fmt.Sprintf("%02s:%02s:%02s,%03s", m[1], m[2], m[3], m[4])
				subLine.End = fmt.Sprintf("%02s:%02s:%02s,%03s", m[5], m[6], m[7], m[8])
				stage = srtExpectText
			}
		case srtExpectText:
			{
				if line == "" {
					stage = srtExpectNum
					duplicated := false
					if len(subLines) > 0 {
						lastLine := subLines[len(subLines)-1]
						if lastLine.End == subLine.Start &&
							lastLine.Text == subLine.Text {
							lastLine.End = subLine.End
							indexNum -= 1
							duplicated = true
						}
					}

					if !duplicated {
						subLines = append(subLines, subLine)
					}
				} else {
					if subLine.Text != "" {
						subLine.Text += "\n"
					}
					subLine.Text += line
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	w := bytes.NewBuffer(nil)
	for _, sub := range subLines {
		w.WriteString(fmt.Sprintf("%d\n", sub.Index))
		w.WriteString(fmt.Sprintf("%s --> %s\n", sub.Start, sub.End))
		w.WriteString(fmt.Sprintf("%s\n\n", sub.Text))
	}

	return w.Bytes(), nil
}

func main() {
	saveSrt := false
	flag.BoolVar(&saveSrt, "save", false, "Save formated srt instead of printing out.")
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Usage: srtformat [options] <input.srt>")
		fmt.Fprintln(os.Stderr, "options:")
		flag.PrintDefaults()
		os.Exit(1)
	}

	srtPath := args[0]
	srtFile, err := os.Open(srtPath)
	if err != nil {
		log.Fatal(err)
	}

	srtBytes, err := ioutil.ReadAll(srtFile)
	srtFile.Close()
	if err != nil {
		log.Fatal(err)
	}

	detector := chardet.NewTextDetector()
	charset, err := detector.DetectBest(srtBytes)
	if err != nil {
		log.Fatal("detect charset failed", err)
	}
	fmt.Fprintln(os.Stderr, "Input chart set:", charset.Charset, " lanuage: ", charset.Language)

	switch charset.Charset {
	case "GB-18030":
		srtBytes, err = decodeToUTF8(srtBytes, simplifiedchinese.GB18030)
	case "Big5":
		srtBytes, err = decodeToUTF8(srtBytes, traditionalchinese.Big5)
	}

	if err != nil {
		log.Fatal("decode error: ", err)
	}

	formatedBytes, err := formatSrt(srtBytes)
	if err != nil {
		log.Fatal(err)
	}

	if saveSrt {
		srtFile, err := os.OpenFile(srtPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			log.Fatal(err)
		}
		srtFile.Write(formatedBytes)
		srtFile.Close()
	} else {
		fmt.Print(string(formatedBytes))
	}
}
