package httpserver

import (
	"encoding/json"
	"fmt"
	"github.com/google/go-cmp/cmp"
	"io"
	"kunCache/gcache"
	"kunCache/peer"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"kunCache/conf"
	"kunCache/consistentHash"
	"kunCache/etcd"
)

// HTTPPool implements PeerPicker for a pool of HTTP peers.
type HTTPPool[K comparable, V any] struct {
	// this peer's base URL, e.g. "localhostt:8000"
	addr     string
	ip       string
	port     string
	protocol string
	basePath string
	mu       sync.Mutex // guards peers and httpGetters
	peers    *consistentHash.Map[K]
	//每一个远程节点对应一个 httpGetter
	httpGetters map[K]*httpGetter[K, V] // keyed by e.g. "10.0.0.2:8008"
}

// NewHTTPPool initializes an HTTP pool of peers.
func NewHTTPPool[K comparable, V any](addr, ip, port, protocol string) *HTTPPool[K, V] {
	return &HTTPPool[K, V]{
		addr:        addr,
		ip:          ip,
		port:        port,
		protocol:    protocol,
		basePath:    conf.GConfig.HttpBasePath,
		peers:       consistentHash.New[K](conf.GConfig.Replicas, nil),
		httpGetters: make(map[K]*httpGetter[K, V]),
	}
}

// ServeHTTP handle all http requests
func (p *HTTPPool[K, V]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	slog.Info("[Server]", "addr", p.addr, "method", r.Method, "url", r.URL.Path, "basePath", p.basePath)
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		// panic("HTTPPool serving unexpected path: " + r.URL.Path)
		slog.Error("HTTPPool serving unexpected path", "url", r.URL.Path)
		return
	}

	// /<basepath>/<groupname>/<key> required
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	groupName := parts[0]
	key := parts[1]

	group := gcache.GetGroup[K, V](groupName)
	if group == nil {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
		return
	}

	view, err := group.Get(any(key).(K))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	//w.Header().Set("Content-Type", "application/octet-stream")
	//w.Write(view)
	data, err := json.Marshal(view)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Write(data)
}

// Set updates the pool's list of peers.
// 加入节点
func (p *HTTPPool[K, V]) AddPeers(peers ...K) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers.Add(peers...)
	for _, peer := range peers {
		//"10.0.0.2:8008/_gcache/"
		p.httpGetters[peer] = &httpGetter[K, V]{baseURL: fmt.Sprintf("%v%v", peer, p.basePath)}
	}
}
func (p *HTTPPool[K, V]) DelPeers(peers ...K) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers.Remove(peers...)
	for _, peer := range peers {
		delete(p.httpGetters, peer)
	}
}

// PickPeer picks a peer according to key
func (p *HTTPPool[K, V]) Pick(key K) (peer.Fetcher[K, V], bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	addr := p.peers.Get(key)
	slog.Info("[Pick]", "addr", addr, "p.addr", p.addr, "p.httpGetters[peer]", p.httpGetters[addr])
	//fmt.Println(cmp.Equal(addr, p.addr))
	//选择的节点不能是空和自身 选自己会一直调用自己
	if !cmp.Equal(addr, p.addr) {
		getter, ok := p.httpGetters[addr]
		//fmt.Println(getter, ok)
		return getter, ok
	}
	return nil, false
	//if peer != "" && peer != p.addr {
	//	return p.httpGetters[peer], true
	//}
	//return nil, false
}

// HTTP 客户端类
type httpGetter[K comparable, V any] struct {
	baseURL string
}

// HTTP 客户端类 httpGetter，实现 Fetch 接口。
func (h *httpGetter[K, V]) Fetch(group string, key K) (value V, err error) {
	u := fmt.Sprintf(
		"http://%v%v/%v",
		h.baseURL,
		url.QueryEscape(group),
		url.QueryEscape(fmt.Sprint(key)),
	)
	//fmt.Println(u)
	res, err := http.Get(u)
	if err != nil {
		return
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return value, fmt.Errorf("server returned: %v", res.Status)
	}

	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return value, fmt.Errorf("reading response body: %v", err)
	}
	json.Unmarshal(bytes, &value)
	return
}

// Start 启动 Cache 服务
func (p *HTTPPool[K, V]) Start() error {
	// 注册服务至 etcd
	go func() {
		// Register never return unless stop signal received (blocked)
		etcd.Register(&etcd.Service{
			Addr:     p.addr,
			IP:       p.ip,
			Port:     p.port,
			Protocol: p.protocol,
		})
	}()
	go etcd.WatchPeers[K, V](p, conf.GConfig.Prefix)
	//TODO

	return http.ListenAndServe(p.addr, p)
}
