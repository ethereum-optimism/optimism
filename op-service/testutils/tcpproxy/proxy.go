package tcpproxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/retry"

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
	p.lgr.Info("set upstream", "addr", addr)
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
			if p.stopped.Load() {
				return
			}
			if err != nil {
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	upConn, err := retry.Do(ctx, 3, retry.Exponential(), func() (net.Conn, error) {
		return net.Dial("tcp", addr)
	})
	cancel()
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

	closeBoth := func() {
		downConn.Close()
		upConn.Close()
		wg.Done()
	}

	pump := func(dst io.Writer, src io.Reader, direction string) {
		defer closeBoth()
		if _, err := io.Copy(dst, src); err != nil {
			// ignore net.ErrClosed since it creates a huge amount of log spam
			if !errors.Is(err, net.ErrClosed) {
				p.lgr.Error("failed to proxy", "direction", direction, "err", err)
			}
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
