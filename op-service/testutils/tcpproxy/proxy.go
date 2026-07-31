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
	ctx          context.Context // set by Start, cancelled by Close to abort in-flight upstream dials
	cancel       context.CancelFunc
	// dial is the upstream dial function; injectable for tests.
	dial func(ctx context.Context, addr string) (net.Conn, error)
}

func New(lgr log.Logger) *Proxy {
	var d net.Dialer
	return &Proxy{
		conns: make(map[net.Conn]struct{}),
		lgr:   lgr,
		dial: func(ctx context.Context, addr string) (net.Conn, error) {
			return d.DialContext(ctx, "tcp", addr)
		},
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
	p.ctx, p.cancel = context.WithCancel(context.Background())
	p.lgr.Info("proxy listening", "addr", lis.Addr().String())

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()

		for {
			downConn, err := p.lis.Accept()
			if p.stopped.Load() {
				if err == nil {
					downConn.Close()
				}
				p.lgr.Info("accept loop exiting: proxy stopped")
				return
			}
			if err != nil {
				p.lgr.Error("accept failed", "err", err, "addr", p.lis.Addr().String(), "stopped", p.stopped.Load())
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
	p.mu.Unlock()
	if addr == "" {
		p.lgr.Error("upstream not set")
		return
	}

	// Dial outside the lock: a slow or unresponsive upstream dial must not
	// stall other accepted connections, SetUpstream, or Close. Each attempt is
	// individually bounded (a single unanswered SYN must not consume the whole
	// budget) and parented on the proxy lifecycle context so Close aborts it.
	ctx, cancel := context.WithTimeout(p.ctx, 10*time.Second)
	upConn, err := retry.Do(ctx, 3, retry.Exponential(), func() (net.Conn, error) {
		attemptCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		return p.dial(attemptCtx, addr)
	})
	cancel()
	if err != nil {
		p.lgr.Error("failed to dial upstream", "err", err)
		return
	}
	defer upConn.Close()

	p.mu.Lock()
	if p.stopped.Load() {
		p.mu.Unlock()
		return
	}
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
	p.lgr.Info("closing proxy", "addr", p.lis.Addr().String())
	p.stopped.Store(true)
	p.cancel()
	p.lis.Close()

	// Close all tracked connections under the lock. handleConn checks
	// p.stopped under p.mu before adding new connections, so after this
	// iteration no new connections can appear in p.conns.
	p.mu.Lock()
	for conn := range p.conns {
		conn.Close()
	}
	p.mu.Unlock()

	p.wg.Wait()
	return nil
}
