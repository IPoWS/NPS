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
	"github.com/IPoWS/node-core/router"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

var (
	nodesfile string
)

func serveFile(w http.ResponseWriter, r *http.Request) {
	nodes.Filemu.RLock()
	if nodesfile == "" {
		nodesfile = "./nodes"
	}
	http.ServeFile(w, r, nodesfile)
	nodes.Filemu.RUnlock()
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
			_, ok := router.Allnodes.Nodes[host]
			if ok {
				serveFile(w, r)
			} else {
				newip := uint32(time.Now().UnixNano()) ^ rand.Uint32()
				for hasExist(newip) {
					newip = uint32(time.Now().UnixNano()) ^ rand.Uint32()
				}
				wsips = append(wsips, newip)
				nip64 := uint64(newip) << 32
				wsip, _, err := link.UpgradeLink(w, r, nip64|1)
				logrus.Infof("[/nps] get peer wsip: %x.", wsip)
				if err == nil && wsip&0xffff_ffff_0000_0000 == nip64 {
					router.AddNode(host, ent, wsip, uint64(time.Now().UnixNano()))
					err = router.SaveNodesBack()
					if err == nil {
						serveFile(w, r)
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
	rand.Seed(time.Now().UnixNano())
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
			err = router.LoadNodes(nodesfile)
			if err != nil {
				logrus.Infof("[loadnodes] %v.", err)
			}
			http.HandleFunc("/nps", nps)
			link.SetNPSUrl("ws://" + os.Args[1] + "/nps")
			link.InitEntry("npsent")
			go func() {
				time.Sleep(time.Second * 5)
				link.Register("npsent")
			}()
			log.Fatal(http.Serve(listener, nil))
		}
	} else {
		fmt.Println("Usage: host:port (nodesfile) (uid)")
	}
}
