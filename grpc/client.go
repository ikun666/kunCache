package grpcserver

//
//import (
//	"context"
//	"kunCache/peer"
//
//	"google.golang.org/grpc"
//	"google.golang.org/grpc/credentials/insecure"
//	"kunCache/grpc/pb/gcachepb"
//)
//
//// client 模块实现了 groupcache 访问其他远程节点从而获取缓存的能力
//type client struct {
//	name string // 服务名称 ip:port
//}
//
//// Fetch 从 remote peer 获取对应的缓存值
//func (c *client) Fetch(group string, key string) ([]byte, error) {
//	conn, err := grpc.Dial(c.name, grpc.WithTransportCredentials(insecure.NewCredentials()))
//	if err != nil {
//		return nil, err
//	}
//
//	grpcClient := gcachepb.NewGroupCacheClient(conn)
//	resp, err := grpcClient.Get(context.Background(), &gcachepb.Request{
//		Group: group,
//		Key:   key,
//	})
//	if err != nil {
//		return nil, err
//	}
//	return resp.Value, err
//}
//
//func NewClient(service string) *client {
//	return &client{name: service}
//}
//
//// 测试 client 是否实现了 Fetcher 接口
//var _ peer.Fetcher = (*client)(nil)
