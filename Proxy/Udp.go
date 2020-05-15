package Proxy

import (
	"XPortForward/TypeDefine"
	"XPortForward/XConfig"
	"github.com/go-redis/redis/v7"
	"log"
	"net"
	"strconv"
	"sync"
	"time"
)

type UdpProxy struct{
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
	//Redis
	Redis *redis.Client
	//ProxyData
	ProxyData *TypeDefine.ProxyData
	//net.Listener
	l *net.UDPConn
	//Close Flag
	CloseFlag bool
	//Main WaitGroup
	wg sync.WaitGroup
	//Close Flag Lock
	CloseFlagLock sync.Mutex
}

type UdpNetConn struct {
	n net.Conn
	r net.Conn
}
func NewUdpProxy(ForwardName string,ProxyIP string,ProxyPort int,ListenIP string,ListenPort int,Redis *redis.Client) *UdpProxy{
	UdpProxy := UdpProxy{ForwardName:ForwardName,ProxyIP:ProxyIP,ProxyPort:ProxyPort,ListenIP:ListenIP,ListenPort:ListenPort,Redis:Redis,ProxyData:&TypeDefine.ProxyData{},CloseFlag:false}
	return &UdpProxy
}

func (t *UdpProxy) Start(){
	//遇到错误直接panic,无须担心,会自动处理
	var Listenerr error
	addr := &net.UDPAddr{IP: net.ParseIP(t.ListenIP), Port: t.ListenPort}
	t.l, Listenerr = net.ListenUDP("udp", addr)
	if Listenerr != nil{
		panic("Start Forward Error: " + Listenerr.Error())
	}
	RedisConn := t.Redis
	//defer func(){
		// _ = RedisConn.Close()
	//}()
	_,RedisSetUploadTrafficErr := RedisConn.Do("Set", t.ForwardName + "_UploadTraffic",0).Result()
	if RedisSetUploadTrafficErr != nil{
		panic("Set Redis Upload Traffic Error: " + RedisSetUploadTrafficErr.Error())
	}
	_,RedisSetDownloadTrafficErr := RedisConn.Do("Set",t.ForwardName + "_DownloadTraffic", 0).Result()
	if RedisSetDownloadTrafficErr != nil{
		panic("Set Redis Download Traffic Error: " + RedisSetDownloadTrafficErr.Error())
	}
	t.wg.Add(1)
	go func() {
		defer func() {
			if e := recover(); e != nil {
				if XConfig.Debug{
					(*t).ProxyData.Status = "stopped"
					log.Println("Forward Listen Error:" + e.(string))
				}
			}
			t.wg.Done()
		}()
		for {
			t.CloseFlagLock.Lock()
			if (*t).CloseFlag {
				t.CloseFlagLock.Unlock()
				break
			}
			t.CloseFlagLock.Unlock()
			var buf = make([]byte, 2048)
			n, srcAddr, err := t.l.ReadFromUDP(buf)
			if err == nil {
				packet := buf[0:n]
				t.wg.Add(1)
				go func() {
					defer func() {
						if e := recover(); e != nil {
							if XConfig.Debug{
								log.Println("Udp Handler Error: " + e.(string))
							}
						}
						t.wg.Done()
					}()
					Udp_Handle(packet, srcAddr,t.Redis,t.ProxyData.DbData.ForwardName,t.l,t.ProxyIP,t.ProxyPort)
				}()
			} else {
				t.CloseFlagLock.Lock()
				if (*t).CloseFlag {
					t.CloseFlagLock.Unlock()
					break
				} else {
					t.CloseFlagLock.Unlock()
					if XConfig.Debug{
						log.Println("Read From Udp Error: " + err.Error())
					}
					continue
				}
			}
		}
	}()
}

func (t *UdpProxy) Stop(IsForce bool,wg *sync.WaitGroup){
	t.CloseFlagLock.Lock()
	(*t).CloseFlag = true
	t.CloseFlagLock.Unlock()
	_ = t.l.Close()
	defer func(){
		// 暂只在强行关闭时触发
		if IsForce{
			wg.Done()
		}
	}()
	//等待所有UDP连接关闭
	if !IsForce{
		t.wg.Wait()
	}
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

func (t *UdpProxy) SetProxyData(data *TypeDefine.ProxyData){
	t.ProxyData = data
}

func Udp_Handle(packet []byte, srcAddr *net.UDPAddr,RedisConn *redis.Client,ForwardName string,l *net.UDPConn,ProxyIP string,ProxyPort int){
	//遇到错误直接panic,无须担心,会自动处理
	dstAddr, err := net.ResolveUDPAddr("udp", ProxyIP + ":" + strconv.Itoa(ProxyPort))
	if err != nil {
		panic("Failed to resolve udp addr: " + err.Error())
	}
	clientSrcAddr := &net.UDPAddr{IP: net.IPv4zero, Port: 0}
	conn, err := net.DialUDP("udp", clientSrcAddr, dstAddr)
	if err != nil {
		panic("Connect Error: " + err.Error())
	}
	defer func(){
		if conn != nil{
			_ = conn.Close()
		}
	}()
	//defer func(){
	//	_ = RedisConn.Close()
	//}()
	//3s没有写成功就关闭
	_ = conn.SetDeadline(time.Now().Add(time.Second*3))
	_, err = conn.Write(packet)
	if err != nil {
		panic("Send To Udp Server Error: " + err.Error())
	}
	_,RedisIncrbyUploadTrafficerr := RedisConn.Do("INCRBY",ForwardName + "_UploadTraffic",len(packet)).Result()
	if RedisIncrbyUploadTrafficerr != nil && XConfig.Debug{
		log.Println("Redis INCRBY Upload Traffic Error: " + RedisIncrbyUploadTrafficerr.Error())
	}
	buf := make([]byte, 512)
	rlen, _, err := conn.ReadFromUDP(buf)
	if err != nil {
		panic("Read Udp Error: " + err.Error())
	}
	_, err = l.WriteToUDP(buf[0:rlen], srcAddr)
	if err != nil {
		panic("Send To Client Error" + err.Error())
	}
	_,RedisIncrbyDownloadTrafficErr := RedisConn.Do("INCRBY",ForwardName + "_DownloadTraffic",rlen).Result()
	if RedisIncrbyDownloadTrafficErr != nil && XConfig.Debug{
		log.Println("Redis INCRBY Download Traffic Error: " + RedisIncrbyDownloadTrafficErr.Error())
	}
}