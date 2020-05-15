package Proxy

import (
	"XPortForward/TypeDefine"
	"XPortForward/XConfig"
	"fmt"
	"github.com/go-redis/redis/v7"
	"log"
	"net"
	"sync"
	"sync/atomic"
)

type TcpProxy struct{
	//转发唯一名称
	ForwardName string
	//转发到的服务器IP
	ProxyIP string
	//转发到的服务器端口
	ProxyPort int
	//转发服务器监听IP地址
	ListenIP string
	//转发服务器监听的端口
	ListenPort int
	//并发数
	MaxConnNum int64
	//速度(kb/s)
	RateLimit int64
	//Redis
	Redis *redis.Client
	//ProxyData
	ProxyData *TypeDefine.ProxyData
	//当前连接数
	NowConnNum int64
	//net.Listener
	l net.Listener
	//Close Flag
	CloseFlag bool
	//Main WaitGroup
	wg sync.WaitGroup
	//Client Conn
	ClientConn []*TcpNetConn
	//Close Flag Lock
	CloseFlagLock sync.Mutex
	//Client Conn Lock
	ClientConnFlag sync.Mutex
}

type TcpNetConn struct {
	n net.Conn
	r net.Conn
}
func NewTcpProxy(ForwardName string,ProxyIP string,ProxyPort int,ListenIP string,ListenPort int,MaxConnNum int64,RateLimit int64,Redis *redis.Client) *TcpProxy{
	TcpProxy := TcpProxy{ForwardName:ForwardName,ProxyIP:ProxyIP,ProxyPort:ProxyPort,ListenIP:ListenIP,ListenPort:ListenPort,MaxConnNum:MaxConnNum,RateLimit:RateLimit,Redis:Redis,ProxyData:&TypeDefine.ProxyData{},CloseFlag:false}
	return &TcpProxy
}

func (t *TcpProxy) Start(){
	//遇到错误直接panic,无须担心,会自动处理
	var Listenerr error
	t.l, Listenerr = net.Listen("tcp", fmt.Sprintf("%s:%d", t.ListenIP, t.ListenPort))
	if Listenerr != nil{
		panic("Start Forward Error: " + Listenerr.Error())
	}
	RedisConn := t.Redis
	//defer func(){
	//	_ = RedisConn.Close()
	//}()
	_,RedisSetUploadTrafficErr := RedisConn.Do("Set", t.ForwardName + "_UploadTraffic",0).Result()
	if RedisSetUploadTrafficErr != nil{
		panic("Set Redis Upload Traffic Error: " + RedisSetUploadTrafficErr.Error())
	}
	_,RedisSetDownloadTrafficErr := RedisConn.Do("Set",t.ForwardName + "_DownloadTraffic", 0).Result()
	if RedisSetDownloadTrafficErr != nil{
		panic("Set Redis Download Traffic Error: " + RedisSetDownloadTrafficErr.Error())
	}
	(*t).wg.Add(1)
	go func() {
		defer func() {
			if e := recover(); e != nil {
				if XConfig.Debug{
					(*t).ProxyData.Status = "stopped"
					log.Println("Forward Listen Error:" + e.(string))
				}
			}
			(*t).wg.Done()
		}()
		for {
			var conn net.Conn
			var err error
			conn, err = (*t).l.Accept()
			if err == nil {
				t.CloseFlagLock.Lock()
				if (*t).CloseFlag{
					t.CloseFlagLock.Unlock()
					//Close
					_ = conn.Close()
					break
				}else{
					t.CloseFlagLock.Unlock()
				}
				if atomic.LoadInt64(&t.NowConnNum)+1 > int64(t.MaxConnNum){
					//并发数超过
					_ = conn.Close()
					continue
				}
				atomic.AddInt64(&t.NowConnNum,1)
				Tcpconn := &TcpNetConn{n:conn}
				t.wg.Add(1)
				go func() {
					defer func() {
						t.CloseFlagLock.Lock()
						if !(*t).CloseFlag{
							//不是因为Close所以退出的
							if e := recover(); e != nil {
								if XConfig.Debug{
									log.Println("Connection Handler Error: " + e.(string))
								}
							}
							t.CloseFlagLock.Unlock()
						}else{
							t.CloseFlagLock.Unlock()
						}
						atomic.AddInt64(&t.NowConnNum,-1)
						if Tcpconn.n != nil{
							_ = Tcpconn.n.Close()
						}
						if Tcpconn.r != nil{
							_ = Tcpconn.r.Close()
						}
						Tcpconn = nil
						(*t).wg.Done()
					}()
					Tcp_Handle(Tcpconn,t.Redis,t.ProxyData.DbData.ForwardName,t.RateLimit,t.ProxyIP,t.ProxyPort)
				}()
				t.ClientConnFlag.Lock()
				(*t).ClientConn = append((*t).ClientConn,Tcpconn)
				t.ClientConnFlag.Unlock()
			} else {
				t.CloseFlagLock.Lock()
				if (*t).CloseFlag {
					t.CloseFlagLock.Unlock()
					break
				} else {
					t.CloseFlagLock.Unlock()
					if XConfig.Debug{
						log.Println("Accept Error: " + err.Error())
					}
					continue
				}
			}
		}
	}()
}

func (t *TcpProxy) Stop(IsForce bool,wg *sync.WaitGroup){
	t.CloseFlagLock.Lock()
	(*t).CloseFlag = true
	t.CloseFlagLock.Unlock()
	if t.l != nil{
		_ = t.l.Close()
	}
	t.ClientConnFlag.Lock()
	defer func(){
		// 暂只在强行关闭时触发
		if IsForce{
			wg.Done()
		}
		t.ClientConn = nil
	}()
	if !IsForce{
		for _,v := range (*t).ClientConn{
			//关掉已经有的链接
			if v.r != nil{
				_ = v.r.Close()
			}
			if v.n != nil{
				_ = v.n.Close()
			}
		}
	}
	t.ClientConnFlag.Unlock()
	RedisConn := t.Redis
	//defer func(){
	//	_ = RedisConn.Close()
	//}()
	_,RedisDelUploadTrafficErr := RedisConn.Do("DEL", t.ForwardName + "_UploadTraffic").Result()
	if RedisDelUploadTrafficErr != nil{
		if XConfig.Debug{
			log.Println("Del Redis Upload Traffic Error: " + RedisDelUploadTrafficErr.Error())
		}
	}
	_,RedisDelDownloadTrafficErr := RedisConn.Do("DEL",t.ForwardName + "_DownloadTraffic").Result()
	if RedisDelDownloadTrafficErr != nil{
		if XConfig.Debug{
			log.Println("Del Redis Download Traffic Error: " + RedisDelDownloadTrafficErr.Error())
		}
	}
}

func (t *TcpProxy) SetProxyData(data *TypeDefine.ProxyData){
	t.ProxyData = data
}

func Tcp_Handle(conn *TcpNetConn,RedisConn *redis.Client,ForwardName string,RateLimit int64,ProxyIP string,ProxyPort int){
	//遇到错误直接panic,无须担心,会自动处理
	var RemoteErr error
	conn.r, RemoteErr = net.Dial("tcp", fmt.Sprintf("%s:%d", ProxyIP, ProxyPort))
	if RemoteErr != nil {
		panic("Remote Connect Error: " +  RemoteErr.Error())
	}
	ProxyLock := make(chan struct{})
	var RedisLock sync.Mutex
	// defer func(){
	//	RedisLock.Lock()
	//	_ = RedisConn.Close()
	//	RedisLock.Unlock()
	//}()
	go func(){
		TrafficFunc := func(traffic int){
			RedisLock.Lock()
			_,RedisIncrbyDownloadTrafficErr := RedisConn.Do("INCRBY",ForwardName + "_DownloadTraffic",traffic).Result()
			if RedisIncrbyDownloadTrafficErr != nil && XConfig.Debug{
				log.Println("Redis INCRBY Download Traffic Error: " + RedisIncrbyDownloadTrafficErr.Error())
			}
			RedisLock.Unlock()
		}
		var err error
		if RateLimit > 0 {
			newreader := NewReader(conn.r)
			newreader.SetRateLimit(float64(RateLimit*1024))
			_,_,err = ioCopy(conn.n,newreader,TrafficFunc)
		} else {
			_,_,err = ioCopy(conn.n,conn.r,TrafficFunc)
		}
		if err != nil && XConfig.Debug{
			func(){
				defer func(){
					if e := recover(); e != nil {
						//Fix Bug
					}
				}()
				close(ProxyLock)
			}()
		}
	}()
	go func(){
		TrafficFunc := func(traffic int){
			RedisLock.Lock()
			_,RedisIncrbyUploadTrafficerr := RedisConn.Do("INCRBY",ForwardName + "_UploadTraffic",traffic).Result()
			if RedisIncrbyUploadTrafficerr != nil && XConfig.Debug{
				log.Println("Redis INCRBY Upload Traffic Error: " + RedisIncrbyUploadTrafficerr.Error())
			}
			RedisLock.Unlock()
		}
		var err error
		if RateLimit > 0 {
			newreader := NewReader(conn.n)
			newreader.SetRateLimit(float64(RateLimit*1024))
			_,_,err = ioCopy(conn.r,newreader,TrafficFunc)
		} else {
			_,_,err = ioCopy(conn.r,conn.n,TrafficFunc)
		}
		if err != nil && XConfig.Debug{
			func(){
				defer func(){
					if e := recover(); e != nil {
						//Fix Bug
					}
				}()
				close(ProxyLock)
			}()
		}
	}()
	<-ProxyLock
	ProxyLock = nil
}