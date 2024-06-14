package grpcserver

import (
	"context"
	"encoding/json"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"kunCache/grpc/pb/gcachepb"
)

// client 模块实现了 groupcache 访问其他远程节点从而获取缓存的能力
type client[K comparable, V any] struct {
	name K // 服务名称 ip:port
}

// Fetch 从 remote peer 获取对应的缓存值
func (c *client[K, V]) Fetch(group string, key K) (value V, err error) {
	conn, err := grpc.Dial(fmt.Sprintf("%v", c.name), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return
	}

	grpcClient := gcachepb.NewGroupCacheClient(conn)
	resp, err := grpcClient.Get(context.Background(), &gcachepb.Request{
		Group: group,
		Key:   fmt.Sprintf("%v", key),
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	err = json.Unmarshal(resp.Value, &value)
	if err != nil {
		fmt.Println(err)
		return
	}
	//fmt.Println("value:", value)
	return
}

func NewClient[K comparable, V any](service K) *client[K, V] {
	return &client[K, V]{name: service}
}

// 测试 client 是否实现了 Fetcher 接口
//var _ peer.Fetcher = (*client)(nil)
