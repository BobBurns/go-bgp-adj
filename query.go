package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type NodeQuery struct {
	Router       string
	RemoteAS     string
	RemoteASName string
	RemoteHost   string
	RemoteID     string
	LocalAddr    string
	RemoteAddr   string
	State        string
	Alert        string
}

var results *[]NodeQuery

// set true to do dns lookups
var lookup = true
var t *template.Template
var alertq int

func (n *NodeQuery) Match(line string) bool {
	found := false
	retVals := strings.Split(line, "=")
	// santity check
	if len(retVals) < 2 {
		n.Router = "Bad query"
		return found
	}

	ids := strings.Split(retVals[0], " ")
	if len(ids) < 3 {
		n.Router = "Bad query"
		return found
	}

	checks := []string{"RemoteAs$", "RemoteAsName", "RemoteIdentifier", "LocalAddr", "RemoteAddr", "State"}
	for _, exp := range checks {
		//fmt.Println(ids[2])
		// must compile
		matched, _ := regexp.Match(exp, []byte(ids[2]))
		if matched {

			found = true
			retVals[1] = strings.TrimSpace(retVals[1])
			switch exp {
			case "RemoteAs$":
				n.RemoteAS = retVals[1]
			case "RemoteAsName":
				n.RemoteASName = retVals[1]
			case "RemoteIdentifier":
				addr := retVals[1]
				// handle hex
				if strings.Contains(addr, ":") {
					addr = convertHexAddr(addr)
				}
				n.RemoteID = addr
				if lookup {

					names, err := net.LookupAddr(addr)
					if err == nil && len(names) > 0 {
						n.RemoteHost = names[0]
					} else {
						n.RemoteHost = "N/A"
					}
				}
			case "LocalAddr":
				n.LocalAddr = retVals[1]
			case "RemoteAddr":
				if n.LocalAddr != "" {
					n.RemoteAddr = retVals[1]
				} else {
					found = false
				}

			case "State":
				vals := strings.Split(retVals[1], ",")
				n.State = vals[1]
				v, err := strconv.Atoi(strings.TrimSpace(vals[0]))
				if err != nil {
					fmt.Println(err)
				}

				//fmt.Println("alert: ", v)
				// see https://tools.ietf.org/html/rfc4273
				if v < 4 {
					n.Alert = "danger"
					alertq++
				} else if v == 6 {
					n.Alert = "success"

				} else if v < 6 {
					n.Alert = "warning"
					alertq++
				} else {
					n.Alert = "info"
				}

			}

		}
	}
	// if no remote id use remote addr to lookup interface name
	if n.RemoteID == "" && lookup {
		names, err := net.LookupAddr(n.RemoteAddr)
		if err == nil && len(names) > 0 {
			n.RemoteHost = names[0]
		} else {
			n.RemoteHost = "N/A"
		}
	}
	return found
}

func getPass() string {
	apipath := "apass"
	apiFile, err := os.Open(apipath)
	if err != nil {
		log.Fatalf("Problem reading file %s\n%s\n", apipath, err)
	}
	var apikeys []string
	scanner := bufio.NewScanner(apiFile)
	for scanner.Scan() {
		apikeys = append(apikeys, scanner.Text())
	}
	if len(apikeys) != 1 {
		log.Fatal("apikey file is not formatted correctly")
	}
	return apikeys[0]
}

func Query() *[]NodeQuery {
	pass := getPass()

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	//filters border routers
	mget := "mget * /border/ /bgp4/ *"
	postValues := url.Values{"cmds": {mget}}

	akipsURL := "https://put-your-url-here/api-db?password=" + pass
	resp, err := client.PostForm(akipsURL, postValues)
	// TODO fix this
	if err != nil {
		log.Fatal("do something", err)
	}
	defer resp.Body.Close()
	//	body, err := ioutil.ReadAll(resp.Body)
	lines := bufio.NewScanner(resp.Body)
	alertq = 0
	var nqs []NodeQuery
	for lines.Scan() {
		line := lines.Text()
		retVals := strings.Split(line, "=")
		ids := strings.Split(retVals[0], " ")
		id := ids[1]
		match := id
		nq := NodeQuery{
			Router: ids[0],
		}
		// this will always skip the first line
		found := false
		for id == match && lines.Scan() {
			line = lines.Text()
			//fmt.Println(line)
			retVals = strings.Split(line, "=")
			ids = strings.Split(retVals[0], " ")
			id = ids[1]
			// do matching

			if nq.Match(line) {
				found = true
			}

		}
		// optionally filter for eBGP
		//if found && nq.RemoteAS != <local AS> {
		if found {
			nqs = append(nqs, nq)
			//fmt.Println(nq)
		}
	}
	return &nqs

}
func convertHexAddr(addr string) string {
	vals := strings.Split(addr, ":")
	if len(vals) != 4 {
		return addr
	}
	var conv [4]uint64
	for i := 0; i < 4; i++ {
		v, err := strconv.ParseUint(vals[i], 16, 8)
		if err != nil {
			return addr
		}
		conv[i] = v
	}
	return fmt.Sprintf("%v.%v.%v.%v", conv[0], conv[1], conv[2], conv[3])
}
