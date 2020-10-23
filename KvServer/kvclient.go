package KvServer

import (
	"HummingbirdDS/Flake"
	"crypto/rand"
	"log"
	"math/big"
	mrand "math/rand"
	"net/rpc"
	"sync/atomic"
	"time"
)

var clients = make(map[int64]bool)

type Clerk struct {
	servers []*rpc.Client

	leader 		int   		// 记录哪一个是leader
	// 为了保证操作的一致性
	seq    		int   		// 当前的操作数
	ClientID    uint64 		// 记录当前客户端的序号
	serversIsOk *[]int32	// 用于记录那一个服务器当前可以连接，是一个bool位
}

func nrand() int64 {
	max := big.NewInt(int64(1) << 62)
	bigx, _ := rand.Int(rand.Reader, max)
	x := bigx.Int64()
	return x
}

// 在创建的时候已经知道了如何于服务端交互
func MakeClerk(servers []*rpc.Client, IsOk *[]int32) *Clerk {
	ck := new(Clerk)
	ck.servers = servers
	ck.serversIsOk = IsOk

	ck.leader = mrand.Intn(len(servers))	// 随机选择一个起始值 生成(0,len(server)-1)的随机数
	ck.seq = 1
	ck.ClientID = Flake.GenSonyflake()

	DPrintf("Clerk: %d\n", ck.ClientID)

	return ck
}

/*
 * @brief: 因为为了保证强一致性，一个客户端一次只会跑一个操作
 */
func (ck *Clerk) Get(key string) string {
	DPrintf("Clerk: Get: %q\n", key)
	serverLength := len(ck.servers)
	for {
		args := &GetArgs{Key: key, ClientID: ck.ClientID, SeqNo: ck.seq}
		reply := new(GetReply)

		ck.leader %= serverLength
		// go中*和[]优先级不一样，要加个括号，挺扯的
		if atomic.LoadInt32(&((*ck.serversIsOk)[ck.leader])) == 0{
			ck.leader++
			continue	// 不能连接就切换
		}

		replyArrival := make(chan bool, 1)
		go func() {
			err := ck.servers[ck.leader].Call("RaftKV.Get", args, reply)
			flag := true
			if err != nil{
				log.Fatal(err.Error())
				flag = false
			}
			replyArrival <- flag
		}()
		select {
		case ok := <-replyArrival:
			if ok {
				if reply.Err == OK || reply.Err == ErrNoKey || reply.Err == Duplicate {
					ck.seq++
					return reply.Value
				} else if reply.Err == ReElection || reply.Err == NoLeader{ // 这两种情况我们需要重新发送请求 即重新选择主
					ck.leader++
				}
			} else {
				ck.leader++
			}
		case <-time.After(200 * time.Millisecond): // RPC超过200ms以后直接切服务器 一般来说信道没有问题200ms绝对够用
			ck.leader++
		}
	}
	return ""
}

func (ck *Clerk) PutAppend(key string, value string, op string) {
	DPrintf("Clerk: PutAppend: %q => (%q,%q) from: %d\n", op, key, value, ck.ClientID)

	cnt := len(ck.servers)
	for {
		args := &PutAppendArgs{Key: key, Value: value, Op: op, ClientID: ck.ClientID, SeqNo: ck.seq}
		reply := new(PutAppendReply)

		ck.leader %= cnt

		if atomic.LoadInt32(&((*ck.serversIsOk)[ck.leader])) == 0{
			ck.leader++
			continue	// 不能连接就切换
		}

		replyArrival := make(chan bool, 1)
		go func() {
			err := ck.servers[ck.leader].Call("RaftKV.PutAppend", args, reply)
			flag := true
			if err != nil{
				log.Fatal(err.Error())
				flag = false
			}
			replyArrival <- flag
		}()
		select {
		case <-time.After(200 * time.Millisecond): // rpc timeout: 200ms
			ck.leader++
			continue
		case ok := <-replyArrival:
			if ok && (reply.Err == OK || reply.Err == Duplicate) {
				ck.seq++
				return
			}
			ck.leader++
		}
	}
}

func (ck *Clerk) Put(key string, value string) {
	ck.PutAppend(key, value, "Put")
}
func (ck *Clerk) Append(key string, value string) {
	ck.PutAppend(key, value, "Append")
}
