package main

import (
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"syscall"
	"time"

	"github.com/IPoWS/node-core/data/nodes"
	"github.com/IPoWS/node-core/link"
	log "github.com/sirupsen/logrus"
)

var (
	nodesfile string
	newnodes  = new(nodes.Nodes)
)

func serveFile(w http.ResponseWriter, r *http.Request) {
	link.NodesList.FileMu.RLock()
	if nodesfile == "" {
		nodesfile = "./nodes"
	}
	http.ServeFile(w, r, nodesfile)
	link.NodesList.FileMu.RUnlock()
}

// websocket实现
func nps(w http.ResponseWriter, r *http.Request) {
	// 检查是否GET请求
	if methodIs("GET", w, r) {
		// 检查uid
		q := r.URL.Query()
		ent := getFirst("ent", &q)
		name := getFirst("name", &q)
		if len(ent) == 6 {
			host := getIPPortStr(r)
			_, ok := link.NodesList.Nodes[host]
			if ok {
				serveFile(w, r)
			} else {
				newip := uint32(time.Now().UnixNano()) ^ rand.Uint32()
				for hasExist(newip) {
					newip = uint32(time.Now().UnixNano()) ^ rand.Uint32()
				}
				wsips = append(wsips, newip)
				nip64 := uint64(newip) << 32
				wsip, delay, err := link.UpgradeLink(w, r, nip64|1)
				log.Infof("[/nps] get peer wsip: %x.", wsip)
				if err == nil {
					// link.NodesList.AddNode(host, ent, wsip, name, uint64(delay))
					newnodes.AddNode(host, ent, wsip, name, uint64(delay))
					err = link.SaveNodesBack()
					if err == nil {
						serveFile(w, r)
						go link.SendNewNodes(newnodes)
					} else {
						http.Error(w, "Save node file error.", http.StatusInternalServerError)
					}
				} else {
					http.Error(w, "Init link error.", http.StatusBadRequest)
					log.Errorf("[/nps] wsip: %x, nip64: %x, err: %v.", wsip, nip64, err)
				}
			}
		} else {
			http.Error(w, "Invalid entry length.", http.StatusBadRequest)
			log.Errorln("[/nps] invalid entry length.")
		}
	}
}

func init() {
	log.SetLevel(log.DebugLevel)
	rand.Seed(time.Now().UnixNano())
	newnodes.Clear()
	t := time.NewTicker(time.Second)
	go func() {
		for range t.C {
			link.SaveNodesBack()
		}
	}()
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
				nodesfile = os.Args[2]
			}
			err = link.LoadNodes(nodesfile)
			if err != nil {
				log.Infof("[loadnodes] %v.", err)
			}
			http.HandleFunc("/nps", nps)
			link.InitEntry("ws://"+os.Args[1]+"/nps", "npsent", "saki.fumiama", 0xffff_ffff_0000_0000)
			go func() {
				time.Sleep(time.Second)
				link.Register()
			}()
			log.Fatal(http.Serve(listener, nil))
		}
	} else {
		fmt.Println("Usage: host:port (nodesfile) (uid)")
	}
}
