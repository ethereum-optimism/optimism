package tcpproxy

import (
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/log"
)

type Proxy struct {
	mu           sync.Mutex
	conns        map[net.Conn]struct{}
	lis          net.Listener
	wg           sync.WaitGroup
	lgr          log.Logger
	upstreamAddr string
	stopped      atomic.Bool
}

func New(lgr log.Logger) *Proxy {
	return &Proxy{
		conns: make(map[net.Conn]struct{}),
		lgr:   lgr,
	}
}

func (p *Proxy) Addr() string {
	return p.lis.Addr().String()
}

func (p *Proxy) SetUpstream(addr string) {
	p.mu.Lock()
	p.upstreamAddr = addr
	p.mu.Unlock()
}

func (p *Proxy) Start() error {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("could not listen: %w", err)
	}
	p.lis = lis

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()

		for {
			downConn, err := p.lis.Accept()
			if err != nil {
				if p.stopped.Load() {
					return
				}
				p.lgr.Error("failed to accept downstream", "err", err)
				continue
			}

			p.wg.Add(1)
			go func() {
				defer p.wg.Done()
				p.handleConn(downConn)
			}()
		}
	}()

	return nil
}

func (p *Proxy) handleConn(downConn net.Conn) {
	defer downConn.Close()

	p.mu.Lock()
	addr := p.upstreamAddr
	if addr == "" {
		p.mu.Unlock()
		p.lgr.Error("upstream not set")
		return
	}

	upConn, err := net.Dial("tcp", addr)
	if err != nil {
		p.mu.Unlock()
		p.lgr.Error("failed to dial upstream", "err", err)
		return
	}
	defer upConn.Close()
	p.conns[downConn] = struct{}{}
	p.conns[upConn] = struct{}{}
	p.mu.Unlock()

	var wg sync.WaitGroup
	wg.Add(2)
	pump := func(dst io.Writer, src io.Reader, direction string) {
		defer wg.Done()
		if _, err := io.Copy(dst, src); err != nil {
			p.lgr.Error("failed to proxy", "direction", direction, "err", err)
		}
	}
	go pump(downConn, upConn, "downstream")
	go pump(upConn, downConn, "upstream")
	wg.Wait()

	p.mu.Lock()
	delete(p.conns, downConn)
	delete(p.conns, upConn)
	p.mu.Unlock()
}

func (p *Proxy) Close() error {
	p.stopped.Store(true)
	p.lis.Close()
	p.mu.Lock()
	for conn := range p.conns {
		conn.Close()
	}
	p.mu.Unlock()
	p.wg.Wait()
	return nil
}
