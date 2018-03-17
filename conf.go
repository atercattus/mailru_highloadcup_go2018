package main

// наивная реализация. проверка корректности

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"regexp"
)

const (
	hitsThresh     = 3
	browsersThresh = 3
)

type In struct {
	Browsers []json.RawMessage
	//Country  json.RawMessage
	Email json.RawMessage
	Hits  []json.RawMessage
	//Job      json.RawMessage
	Name json.RawMessage
	//Phone    json.RawMessage
}

func Fast(inRdr io.Reader, out io.Writer, networks []string) {
	var (
		userAgentRe = regexp.MustCompile(`Chrome/(60.0.3112.90|52.0.2743.116|57.0.2987.133)`)
		results     []string
		in          In
	)

	inRdrBuf := bufio.NewReader(inRdr)
	for userId := 1; ; userId++ {
		if line, _, err := inRdrBuf.ReadLine(); err != nil {
			break
		} else {
			json.Unmarshal(line, &in)

			hitsCnt := 0

			for _, hit := range in.Hits {
				hitIP := net.ParseIP(string(hit[1 : len(hit)-1]))

				for _, network := range networks {
					_, networkNet, _ := net.ParseCIDR(network)

					if networkNet.Contains(hitIP) {
						hitsCnt++
					}
				}
			}

			browsersCnt := 0

			for _, browser := range in.Browsers {
				if userAgentRe.Match(browser) {
					browsersCnt++
				}
			}

			if hitsCnt < hitsThresh || browsersCnt < browsersThresh {
				continue
			}

			name := in.Name[1 : len(in.Name)-1]

			email := in.Email[1 : len(in.Email)-1]
			email = bytes.Replace(email, []byte(`@`), []byte(` [at] `), 1)

			results = append(results, fmt.Sprintf("[%d] %s <%s>", userId, name, email))
		}
	}

	fmt.Fprintf(out, "Total: %d\n", len(results))
	for _, result := range results {
		fmt.Fprintln(out, result)
	}
}
