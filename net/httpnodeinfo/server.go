package httpnodeinfo

import (
	"UNetwork/common/config"
	"UNetwork/core/ledger"
	. "UNetwork/net/protocol"
	"fmt"
	"html/template"
	"net/http"
	"sort"
	"strconv"
)

type Info struct {
	NodeVersion  string
	BlockHeight  uint32
	NeighborCnt  int
	Neighbors    []NgbNodeInfo
	HttpRestPort int
	HttpWsPort   int
	HttpJsonPort int
	NodePort     int
	NodeId       string
	NodeType     string
}

const (
	verifyNode  = "Verify Node"
	serviceNode = "Service Node"
)

var node UNode

var templates = template.Must(template.New("info").Parse(page))

func newNgbNodeInfo(ngbId string, ngbType string, ngbAddr string, httpInfoAddr string, httpInfoPort int, httpInfoStart bool) *NgbNodeInfo {
	return &NgbNodeInfo{NgbId: ngbId, NgbType: ngbType, NgbAddr: ngbAddr, HttpInfoAddr: httpInfoAddr,
		HttpInfoPort: httpInfoPort, HttpInfoStart: httpInfoStart}
}

func initPageInfo(blockHeight uint32, curNodeType string, ngbrCnt int, ngbrsInfo []NgbNodeInfo) (*Info, error) {
	id := fmt.Sprintf("0x%x", node.GetID())
	return &Info{NodeVersion: config.Version, BlockHeight: blockHeight,
		NeighborCnt: ngbrCnt, Neighbors: ngbrsInfo,
		HttpRestPort: config.Parameters.HttpRestPort,
		HttpWsPort:   config.Parameters.HttpWsPort,
		HttpJsonPort: config.Parameters.HttpJsonPort,
		NodePort:     config.Parameters.NodePort,
		NodeId:       id, NodeType: curNodeType}, nil
}

func viewHandler(w http.ResponseWriter, r *http.Request) {
	var ngbrUNodesInfo []NgbNodeInfo
	var ngbId string
	var ngbAddr string
	var ngbType string
	var ngbInfoPort int
	var ngbInfoState bool
	var ngbHttpInfoAddr string

	curNodeType := serviceNode
	bookKeepers, _, _ := ledger.DefaultLedger.Store.GetBookKeeperList()
	bookKeeperLen := len(bookKeepers)
	for i := 0; i < bookKeeperLen; i++ {
		if node.GetPubKey().X.Cmp(bookKeepers[i].X) == 0 {
			curNodeType = verifyNode
			break
		}
	}

	ngbrUNodes := node.GetNeighborUNode()
	ngbrsLen := len(ngbrUNodes)
	for i := 0; i < ngbrsLen; i++ {
		ngbType = serviceNode
		for j := 0; j < bookKeeperLen; j++ {
			if ngbrUNodes[i].GetPubKey().X.Cmp(bookKeepers[j].X) == 0 {
				ngbType = verifyNode
				break
			}
		}

		ngbAddr = ngbrUNodes[i].GetAddr()
		ngbInfoPort = ngbrUNodes[i].GetHttpInfoPort()
		ngbInfoState = ngbrUNodes[i].GetHttpInfoState()
		ngbHttpInfoAddr = ngbAddr + ":" + strconv.Itoa(ngbInfoPort)
		ngbId = fmt.Sprintf("0x%x", ngbrUNodes[i].GetID())

		ngbrInfo := newNgbNodeInfo(ngbId, ngbType, ngbAddr, ngbHttpInfoAddr, ngbInfoPort, ngbInfoState)
		ngbrUNodesInfo = append(ngbrUNodesInfo, *ngbrInfo)
	}
	sort.Sort(NgbNodeInfoSlice(ngbrUNodesInfo))

	blockHeight := ledger.DefaultLedger.Blockchain.BlockHeight
	pageInfo, err := initPageInfo(blockHeight, curNodeType, ngbrsLen, ngbrUNodesInfo)
	if err != nil {
		http.Redirect(w, r, "/info", http.StatusFound)
		return
	}

	err = templates.ExecuteTemplate(w, "info", pageInfo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func StartServer(n UNode) {
	node = n
	port := int(config.Parameters.HttpInfoPort)
	http.HandleFunc("/info", viewHandler)
	http.ListenAndServe(":"+strconv.Itoa(port), nil)
}
