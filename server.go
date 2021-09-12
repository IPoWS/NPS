package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"syscall"

	. "github.com/IPoWS/node-core/data/nodes"
	"github.com/IPoWS/node-core/link"
	log "github.com/sirupsen/logrus"
)

var (
	nodes Nodes
)

func serveFile(w http.ResponseWriter, r *http.Request) {
	Filemu.RLock()
	http.ServeFile(w, r, Nodesfile)
	Filemu.RUnlock()
}

// websocket实现
func nps(w http.ResponseWriter, r *http.Request) {
	// 检查是否GET请求
	if methodIs("GET", w, r) {
		// 检查uid
		q := r.URL.Query()
		ent := getFirst("ent", &q)
		if len(ent) == 6 {
			host := getIPPortStr(r)
			_, ok := nodes.Nodes[host]
			if ok {
				serveFile(w, r)
			} else {
				conn, _, _, err := link.InitLink(host + "/" + ent)
				if err == nil {
					conn.Close()
					node := new(Node)
					ip := net.ParseIP(host)
					if ip != nil {
						node.Ip |= uint32(ip[12]) << 24
						node.Ip |= uint32(ip[13]) << 16
						node.Ip |= uint32(ip[14]) << 8
						node.Ip |= uint32(ip[15])
						_, p, err := net.SplitHostPort(host)
						if err == nil {
							port, err := strconv.Atoi(p)
							if err == nil && port > 0 && port < 65536 {
								node.Port = uint32(port)
								node.Entry = ent
								node.Wsnetaddr = uint32(len(nodes.Nodes)) + 1
								if node.Wsnetaddr == 0 {
									http.Error(w, "500 Internal Server Error\nMax ip addr amount exceeded.", http.StatusInternalServerError)
									log.Errorln("[/nps] max ip addr amount exceeded.")
								}
								Memmu.Lock()
								nodes.Nodes[host] = node
								Memmu.Unlock()
								err = nodes.Save()
								if err == nil {
									serveFile(w, r)
								} else {
									http.Error(w, "500 Internal Server Error\nNode mashal error.", http.StatusInternalServerError)
									log.Errorln("[/nps] node mashal error.")
								}
							}
						}
					} else {
						http.Error(w, "500 Internal Server Error\nSplit host port error.", http.StatusInternalServerError)
						log.Errorln("[/nps] split host port error.")
					}
				} else {
					http.Error(w, "400 BAD REQUEST\nInit link error.", http.StatusBadRequest)
				}
			}
		} else {
			http.Error(w, "400 BAD REQUEST\nInvalid entry length.", http.StatusBadRequest)
			log.Errorln("[/nps] invalid entry length.")
		}
	}
}

func main() {
	l := len(os.Args)
	if l >= 2 && l <= 4 {
		listener, err := net.Listen("tcp", os.Args[1])
		if err != nil {
			panic(err)
		} else {
			if l == 4 {
				uid, err := strconv.Atoi(os.Args[3])
				if err == nil {
					syscall.Setuid(uid)
					syscall.Setgid(uid)
				} else {
					panic(err)
				}
			}
			if l >= 3 && os.Args[2] != "" {
				Nodesfile = os.Args[2]
			}
			nodes.Load()
			http.HandleFunc("/nps", nps)
			link.InitEntry("/npsent")
			log.Fatal(http.Serve(listener, nil))
		}
	} else {
		fmt.Println("Usage: host:port (nodesfile) (uid)")
	}
}
