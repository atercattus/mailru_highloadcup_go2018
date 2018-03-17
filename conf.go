package main

// заменил json на свой парсер

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strconv"
)

const (
	hitsThresh     = 3
	browsersThresh = 3
)

type (
	IP      uint32
	IPRange struct {
		IP   IP
		Mask IP
	}

	InMy struct {
		Browsers [][]byte
		Hits     [][]byte
		Name     []byte
		Email    []byte
	}
)

func (ip IP) String() string {
	return fmt.Sprintf(`%d.%d.%d.%d`, ip>>24, (ip>>16)&0xFF, (ip>>8)&0xFF, ip&0xFF)
}

func (r IPRange) Contains(ip IP) bool {
	return r.IP&r.Mask == ip&r.Mask
}

func Fast(inRdr io.Reader, out io.Writer, networks []string) {
	var (
		userAgentRe = regexp.MustCompile(`Chrome/(60.0.3112.90|52.0.2743.116|57.0.2987.133)`)
		results     []string
	)

	var netParsed = parseNetworksMy(networks)

	inRdrBuf := bufio.NewReader(inRdr)
	for userId := 1; ; userId++ {
		if line, _, err := inRdrBuf.ReadLine(); err != nil { // можно параллелиться по строкам?
			break
		} else {
			hitsCnt := 0
			browsersCnt := 0

			ch := jsonPipe(line)

			for in := range ch {
			hitsLoop:
				for _, hit := range in.Hits {
					hitIP, _ := parseIP(string(hit))

					for _, network := range netParsed {
						if !network.Contains(hitIP) {
						} else if hitsCnt++; hitsCnt >= hitsThresh {
							break hitsLoop
						}
					}
				}

				for _, browser := range in.Browsers {
					if !userAgentRe.Match(browser) {
					} else if browsersCnt++; browsersCnt >= browsersThresh {
						break
					}
				}

				if hitsCnt < hitsThresh || browsersCnt < browsersThresh {
					continue
				}

				email := bytes.Replace(in.Email, []byte(`@`), []byte(` [at] `), 1)

				results = append(results, fmt.Sprintf("[%d] %s <%s>", userId, in.Name, email))
			}
		}
	}

	fmt.Fprintf(out, "Total: %d\n", len(results))
	for _, result := range results {
		fmt.Fprintln(out, result)
	}
}

func parseIP(s string) (ip IP, n int) {
	var oct uint32
	var ch byte
	shift := uint32(24)
	for n, ch = range []byte(s) {
		if ch == '.' {
			ip = ip + IP(oct<<shift)
			oct = 0
			shift -= 8
		} else if ch >= '0' && ch <= '9' {
			oct = (oct * 10) + uint32(ch-'0')
		} else {
			break
		}
	}

	if oct > 0 {
		ip = ip + IP(oct)
	}

	return
}

func parseNetworksMy(netRaw []string) (netParsed []IPRange) {
	netParsed = make([]IPRange, len(netRaw))

	var ipR IPRange

	for i, n := range netRaw {
		ip, l := parseIP(n)
		mask, _ := strconv.ParseUint(n[l+1:], 10, 32)

		ipR.IP = ip
		ipR.Mask = IP(0xFFFFFFFF << (32 - mask))

		netParsed[i] = ipR
	}
	return
}

// {"browsers":["foo",..],"company":"Tavu","country":"Albania","email":"tHall@Fiveclub.edu","hits":["151.62.127.96",...],"job":"Staff Scientist","name":"Billy Stephens","phone":"508-76-84"}
func jsonPipe(js []byte) (ch chan *InMy) {
	ch = make(chan *InMy, 100)

	go func() {
		var pos int

		checkCh := func(want byte) (c byte) {
			c = js[pos]
			pos++
			if c != want {
				panic(`checkCh. want:` + string(want) + ` got:` + string(c))
			}
			return
		}

		getCh := func() (c byte) {
			c = js[pos]
			pos++
			return
		}

		fetchString := func() []byte {
			checkCh('"')

			p := bytes.IndexByte(js[pos:], '"')
			s := js[pos : pos+p]
			pos += p + 1

			return s
		}

		fetchSliceOfStrings := func() (slice [][]byte) {
			checkCh('[')
			for {
				slice = append(slice, fetchString())
				c := getCh()
				if c == ']' {
					break
				} else if c == ',' {
				} else {
					panic(`fetchSliceOfStrings`)
				}
			}
			return
		}

		checkCh('{')

	loop:
		for {
			in := &InMy{}

			for {
				if pos >= len(js) {
					break loop
				}
				section := fetchString()
				checkCh(':')

				if bytes.Equal(section, []byte(`browsers`)) {
					in.Browsers = fetchSliceOfStrings()
				} else if bytes.Equal(section, []byte(`company`)) {
					fetchString()
				} else if bytes.Equal(section, []byte(`country`)) {
					fetchString()
				} else if bytes.Equal(section, []byte(`email`)) {
					in.Email = fetchString()
				} else if bytes.Equal(section, []byte(`hits`)) {
					in.Hits = fetchSliceOfStrings()
				} else if bytes.Equal(section, []byte(`job`)) {
					fetchString()
				} else if bytes.Equal(section, []byte(`name`)) {
					in.Name = fetchString()
				} else if bytes.Equal(section, []byte(`phone`)) {
					fetchString()
				} else {
					panic(`Unknown section: ` + string(section))
				}

				c := getCh()
				if c == ',' {
				} else if c == '}' {
					break
				} else {
					panic(`WTF end:` + string(c))
				}
			}

			ch <- in
		}

		close(ch)
	}()

	return
}
