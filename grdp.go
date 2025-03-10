package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"github.com/robertt-smg/gordp/protocol/rfb"

	"github.com/robertt-smg/gordp/core"
	"github.com/robertt-smg/gordp/glog"
	"github.com/robertt-smg/gordp/protocol/nla"
	"github.com/robertt-smg/gordp/protocol/pdu"
	"github.com/robertt-smg/gordp/protocol/sec"
	"github.com/robertt-smg/gordp/protocol/t125"
	"github.com/robertt-smg/gordp/protocol/tpkt"
	"github.com/robertt-smg/gordp/protocol/x224"
)

type Client struct {
	Host string // ip:port
	tpkt *tpkt.TPKT
	x224 *x224.X224
	mcs  *t125.MCSClient
	sec  *sec.Client
	pdu  *pdu.Client
	vnc  *rfb.RFB
}

func NewClient(host string, logLevel glog.LEVEL) *Client {
	glog.SetLevel(logLevel)
	logger := log.New(os.Stdout, "", 0)
	glog.SetLogger(logger)
	return &Client{
		Host: host,
	}
}

func (g *Client) Login(domain, user, pwd string) error {
	conn, err := net.DialTimeout("tcp", g.Host, 3*time.Second)
	if err != nil {
		return fmt.Errorf("[dial err] %v", err)
	}
	defer conn.Close()
	glog.Info(conn.LocalAddr().String())
	//domain := strings.Split(g.Host, ":")[0]

	g.tpkt = tpkt.New(core.NewSocketLayer(conn), nla.NewNTLMv2(domain, user, pwd))
	g.x224 = x224.New(g.tpkt)
	g.mcs = t125.NewMCSClient(g.x224)
	g.sec = sec.NewClient(g.mcs)
	g.pdu = pdu.NewClient(g.sec)

	g.sec.SetUser(user)
	g.sec.SetPwd(pwd)
	g.sec.SetDomain(domain)
	//g.sec.SetClientAutoReconnect()

	g.tpkt.SetFastPathListener(g.sec)
	g.sec.SetFastPathListener(g.pdu)
	//g.x224.SetChannelSender(g.tpkt)
	//g.mcs.SetChannelSender(g.x224)
	g.sec.SetChannelSender(g.mcs)
	//g.pdu.SetFastPathSender(g.tpkt)

	//g.x224.SetRequestedProtocol(x224.PROTOCOL_SSL)
	//g.x224.SetRequestedProtocol(x224.PROTOCOL_RDP)

	err = g.x224.Connect()
	if err != nil {
		return fmt.Errorf("[x224 connect err] %v", err)
	}
	/*
		c := &cliprdr.CliprdrClient{}
		c.SetSender(g.sec)
		g.sec.On(c.GetType(), func(s []byte) {
			c.Handle(s)
		})
	*/
	glog.Info("wait connect ok")
	wg := &sync.WaitGroup{}
	wg.Add(1)

	g.pdu.On("error", func(e error) {
		err = e
		glog.Error("error", e)
		wg.Done()
	}).On("close", func() {
		err = errors.New("close")
		glog.Info("on close")
		//wg.Done()
	}).On("success", func() {
		err = nil
		glog.Info("on success")
		//wg.Done()
	}).On("ready", func() {
		glog.Info("on ready")
	}).On("update", func(rectangles []pdu.BitmapData) {
		glog.Info("on update bitmap:", len(rectangles))
	})

	wg.Wait()
	return err
}

func (g *Client) LoginVNC() error {
	conn, err := net.DialTimeout("tcp", g.Host, 3*time.Second)
	if err != nil {
		return fmt.Errorf("[dial err] %v", err)
	}
	defer conn.Close()
	glog.Info(conn.LocalAddr().String())
	//domain := strings.Split(g.Host, ":")[0]

	g.vnc = rfb.NewRFB(rfb.NewRFBConn(conn))
	wg := &sync.WaitGroup{}
	wg.Add(1)

	g.vnc.On("error", func(e error) {
		glog.Info("on error")
		err = e
		glog.Error(e)
		wg.Done()
	}).On("close", func() {
		err = errors.New("close")
		glog.Info("on close")
		//wg.Done()
	}).On("success", func() {
		err = nil
		glog.Info("on success")
		//wg.Done()
	}).On("ready", func() {
		glog.Info("on ready")
	}).On("update", func(b *rfb.BitRect) {
		glog.Info("on update:", b)
	})
	glog.Info("on Wait")
	wg.Wait()
	return err
}

var (
	ip       string
	domain   string
	user     string
	passwd   string
	loglevel int
)

func main() {
	flag.StringVar(&ip, "m", "localhost:3389", "ip:port")
	flag.StringVar(&domain, "d", "", "domain")
	flag.StringVar(&user, "u", "", "user")
	flag.StringVar(&passwd, "p", "", "passwd")
	flag.IntVar(&loglevel, "l", 1, "debug:0 info:1 warn:2 error:3")
	flag.Parse()
	if user == "" || passwd == "" {
		fmt.Println("user and passwd empty")
		os.Exit(-1)
	}
	g := NewClient(ip, glog.LEVEL(loglevel))
	err := g.Login(domain, user, passwd)
	//g := NewClient("192.168.0.132:3389", glog.LEVEL(loglevel))
	//err := g.Login("", "administrator", "Jhadmin123")
	//g := NewClient("192.168.18.100:5902", glog.DEBUG)
	//err := g.LoginVNC()

	if err != nil {
		fmt.Println("Login:", err)
	}

}
