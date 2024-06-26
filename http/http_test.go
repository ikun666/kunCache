package httpserver

import (
	"fmt"
	"kunCache/conf"
	"kunCache/etcd"
	"log"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"kunCache/gcache"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
	"IKUN": "250",
	"CXK":  "2.5",
}

func createGroup(name string) *gcache.Group[string, []byte] {
	return gcache.NewGroup[string, []byte]("scores", 2<<10, gcache.GetterFunc[string, []byte](
		func(key string) ([]byte, error) {
			slog.Info("[SlowDB] search key", "key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			slog.Info("[not exist]", "key", key)
			return nil, fmt.Errorf("%s not exist", key)
		}))
}

// 启动缓存服务器：创建 HTTPPool，添加节点信息，注册到 g 中，启动 HTTP 服务

func startCacheHTTPServer(addr, ip, port, protocol string, g *gcache.Group[string, []byte]) {
	server := NewHTTPPool[string, []byte](addr, ip, port, protocol)
	addrs, err := etcd.DiscoverPeers(conf.GConfig.Prefix)
	if err != nil {
		log.Println(err)
		return
	}
	// 将节点打到哈希环上
	server.AddPeers(addrs...)
	// 为 Group 注册服务 Picker
	g.RegisterServer(server)
	slog.Info("gcache is running at", "addr", addr)
	// 启动服务
	err = server.Start()
	if err != nil {
		log.Fatal(err)
	}
}

// 启动一个 API 服务
func startAPIServer(apiAddr string, g *gcache.Group[string, []byte]) {
	http.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		t1 := time.Now()
		key := r.URL.Query().Get("key")
		view, err := g.Get(key)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// fmt.Println("api", view.String(), view.ByteSlice())
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(view)
		fmt.Printf("use time:%v um\n", time.Since(t1).Milliseconds())

	})
	slog.Info("server is running at", "apiAddr", apiAddr)
	log.Fatal(http.ListenAndServe(apiAddr[7:], nil))

}
func TestHttp(t *testing.T) {

	//conf.Init("../conf/conf.json")
	//go func() {
	//	g := createGroup("ikun666")
	//	go startAPIServer(conf.GConfig.ApiAddr, g)
	//	startCacheHTTPServer("localhost:8000", "localhost", "8000", "HTTP", g)
	//}()
	//go func() {
	//	g := createGroup("ikun666")
	//	startCacheHTTPServer("localhost:8100", "localhost", "8100", "HTTP", g)
	//}()
	//go func() {
	//	g := createGroup("ikun666")
	//	startCacheHTTPServer("localhost:8200", "localhost", "8200", "HTTP", g)
	//}()
	//
	//select {}

}
