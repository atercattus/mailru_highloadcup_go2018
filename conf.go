package main

// немного срезал аллокаций

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"regexp"
	"strconv"
	"strings"
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

	var netParsed = parseNetworks(networks)

	inRdrBuf := bufio.NewReader(inRdr)
	for userId := 1; ; userId++ {
		if line, _, err := inRdrBuf.ReadLine(); err != nil {
			break
		} else {
			json.Unmarshal(line, &in)

			hitsCnt := 0

		hitsLoop:
			for _, hit := range in.Hits {
				hitIP := net.ParseIP(string(hit[1 : len(hit)-1]))

				for _, network := range netParsed {
					if !network.Contains(hitIP) {
					} else if hitsCnt++; hitsCnt >= hitsThresh {
						break hitsLoop
					}
				}
			}

			browsersCnt := 0

			for _, browser := range in.Browsers {
				if !userAgentRe.Match(browser) {
				} else if browsersCnt++; browsersCnt >= browsersThresh {
					break
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

func parseCIDR(s string) (n net.IPNet) {
	i := strings.IndexByte(s, '/')
	addr, maskStr := s[:i], s[i+1:]
	iplen := net.IPv4len
	ip := parseIPv4(addr)

	mask, _ := strconv.Atoi(maskStr)
	m := net.CIDRMask(mask, 8*iplen)

	n.IP = ip.Mask(m)
	n.Mask = m
	return
}

func parseIPv4(s string) net.IP {
	var p [net.IPv4len]byte
	for i := 0; i < net.IPv4len; i++ {
		if len(s) == 0 {
			// Missing octets.
			return nil
		}
		if i > 0 {
			if s[0] != '.' {
				return nil
			}
			s = s[1:]
		}
		n, c, ok := dtoi(s)
		if !ok || n > 0xFF {
			return nil
		}
		s = s[c:]
		p[i] = byte(n)
	}
	if len(s) != 0 {
		return nil
	}
	return net.IPv4(p[0], p[1], p[2], p[3])
}

func dtoi(s string) (n int, i int, ok bool) {
	const big = 0xFFFFFF

	n = 0
	for i = 0; i < len(s) && '0' <= s[i] && s[i] <= '9'; i++ {
		n = n*10 + int(s[i]-'0')
		if n >= big {
			return big, i, false
		}
	}
	if i == 0 {
		return 0, 0, false
	}
	return n, i, true
}

func parseNetworks(netRaw []string) (netParsed []net.IPNet) {
	netParsed = make([]net.IPNet, len(netRaw))
	for i, n := range netRaw {
		netParsed[i] = parseCIDR(n)
	}
	return
}
