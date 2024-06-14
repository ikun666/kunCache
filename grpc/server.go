package grpcserver

import (
	"context"
	"encoding/json"
	"github.com/google/go-cmp/cmp"
	"kunCache/gcache"
	"kunCache/peer"
	"log"

	"fmt"
	"net"
	"sync"

	"google.golang.org/grpc"
	"kunCache/conf"
	"kunCache/consistentHash"
	"kunCache/etcd"
	"kunCache/grpc/pb/gcachepb"
)

// server 模块为 groupcache 之间提供了通信能力
// 这样部署在其他机器上的 groupcache 可以通过访问 server 获取缓存
// 至于找哪一个主机，由一致性 hash 负责

// server 和 Group 是解耦合的，所以 server 要自己实现并发控制
type Server[K comparable, V any] struct {
	gcachepb.UnimplementedGroupCacheServer

	Addr     string
	IP       string
	Port     string
	Protocol string
	Status   bool // true: running false: stop
	mu       sync.Mutex
	consHash *consistentHash.Map[K]
	clients  map[K]*client[K, V]
}

// NewServer 创建 cache 的 server，若 addr 为空，则使用 defaultAddr
func NewServer[K comparable, V any](addr, ip, port, protocol string) (*Server[K, V], error) {
	return &Server[K, V]{
		Addr:     addr,
		IP:       ip,
		Port:     port,
		Protocol: protocol,
		consHash: consistentHash.New[K](conf.GConfig.Replicas, nil),
		clients:  make(map[K]*client[K, V]),
	}, nil
}

// Get 实现了 Groupcache service 的 Get 方法
func (s *Server[K, V]) Get(ctx context.Context, req *gcachepb.Request) (*gcachepb.Response, error) {
	groupName, key := req.GetGroup(), req.GetKey()
	resp := &gcachepb.Response{}
	// logger.Logger.Infof("[groupcache server %s] Recv RPC Request - (%s)/(%s)", s.Addr, group, key)
	log.Printf("[groupcache server %s] Recv RPC Request - (%s)/(%s)", fmt.Sprintf("%v:%v", s.IP, s.Port), groupName, key)
	if key == "" || groupName == "" {
		return resp, fmt.Errorf("key and group name is reqiured")
	}

	g := gcache.GetGroup[K, V](groupName)
	if g == nil {
		return resp, fmt.Errorf("group %s not found", groupName)
	}
	view, err := g.Get((any)(key).(K))
	if err != nil {
		return resp, err
	}
	//fmt.Println("view:", view)
	data, err := json.Marshal(view)
	resp.Value = data
	return resp, err
}

// Start 启动 Cache 服务
func (s *Server[K, V]) Start() error {
	s.mu.Lock()

	if s.Status {
		s.mu.Unlock()
		return fmt.Errorf("server %s is already started", fmt.Sprintf("%v:%v", s.IP, s.Port))
	}

	// ------------启动服务----------------
	// 1. 设置 status = true 表示服务器已经在运行
	// 2. 初始化 stop channel，用于通知 registry stop keepalive
	// 3. 初始化 tcp socket 并开始监听
	// 4. 注册 rpc 服务至 grpc，这样 grpc 收到 request 可以分发给 server 处理
	// 5. 将自己的服务名/Host地址注册至 etcd，这样 client 就可以通过 etcd 获取服务 Host 地址进行通信；这样做的好处是：client 只需要知道服务名称以及 etcd 的 Host 就可以获取
	// 指定服务的 IP，无需将它们写死在 client 代码中
	s.Status = true

	lis, err := net.Listen("tcp", fmt.Sprintf("%v:%v", s.IP, s.Port))
	if err != nil {
		return fmt.Errorf("failed to listen %s, error: %v", fmt.Sprintf("%v:%v", s.IP, s.Port), err)
	}
	grpcServer := grpc.NewServer()
	gcachepb.RegisterGroupCacheServer(grpcServer, s)

	// 注册服务至 etcd
	go func() {
		// Register never return unless stop signal received (blocked)
		etcd.Register(&etcd.Service{
			Addr:     s.Addr,
			IP:       s.IP,
			Port:     s.Port,
			Protocol: s.Protocol,
		})

		// close tcp listen
		err = lis.Close()
		if err != nil {
			log.Fatalln(err)
		}
		// logger.Logger.Infof("[%s] Revoke service and close tcp socket ok.", s.Addr)
		fmt.Printf("[%s] Revoke service and close tcp socket ok.\n", fmt.Sprintf("%v:%v", s.IP, s.Port))
	}()
	go etcd.WatchPeers[K, V](s, conf.GConfig.Prefix)
	// logger.Logger.Infof("[%s] register service ok\n", s.Addr)
	s.mu.Unlock()
	// Serve接受侦听器列表上的传入连接，为每个连接创建一个新的ServerTransport和服务Goroutine。
	// 服务Goroutines读取GRPC请求，然后调用注册的处理程序来回复它们。当lis.Accept失败并出现致命错误时，Serve返回。当此方法返回时，LIS将关闭。
	// 除非调用Stop或GracefulStop，否则SERVE将返回非零错误。
	if err := grpcServer.Serve(lis); s.Status && err != nil {
		return fmt.Errorf("failed to serve %s, error: %v", fmt.Sprintf("%v:%v", s.IP, s.Port), err)
	}
	return nil
}

// AddPeers 将远端主机 IP 配置到 Server 里
// 这样 Server 就可以 Pick 它们了
func (s *Server[K, V]) AddPeers(peersAddr ...K) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.consHash.Add(peersAddr...)

	for _, peersAddr := range peersAddr {
		s.clients[peersAddr] = NewClient[K, V](peersAddr)
	}
}
func (s *Server[K, V]) DelPeers(peersAddr ...K) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.consHash.Remove(peersAddr...)

	for _, peersAddr := range peersAddr {
		delete(s.clients, peersAddr)
	}
}

// Pick 根据一致性哈希选举出 key 应该存放在的 cache
// return false 代表从本地获取 cache
func (s *Server[K, V]) Pick(key K) (peer.Fetcher[K, V], bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	peerAddr := s.consHash.Get(key)
	// Pick itself

	if cmp.Equal(peerAddr, fmt.Sprintf("%v:%v", s.IP, s.Port)) {
		// logger.Logger.Infof("oohhh! pick myself, i am %s\n", s.Addr)
		fmt.Printf("oohhh! pick myself, i am %s\n", fmt.Sprintf("%v:%v", s.IP, s.Port))
		return nil, false
	}

	// logger.Logger.Infof("[cache %s] pick remote peer: %s\n", s.Addr, peerAddr)
	fmt.Printf("[cache %s] pick remote peer: %v,client:%v\n", fmt.Sprintf("%v:%v", s.IP, s.Port), peerAddr, s.clients[peerAddr])
	return s.clients[peerAddr], true
}

// Stop 停止 server 运行，如果 server 没有运行，这将是一个 no-op
func (s *Server[K, V]) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.Status {
		return
	}
	// 发送停止 keepAlive 的信号，因为该节点要退出了，不需要再发送心跳探测了
	s.Status = false
	s.clients = nil // 清空一致性哈希信息，帮助 GC 进行垃圾回收
	s.consHash = nil
}

// 测试 Server 是否实现了 Picker 接口
//var _ peer.Picker = (*Server[string,[]byte])(nil)
