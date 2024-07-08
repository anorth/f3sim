package certexchange

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/filecoin-project/go-f3/certstore"
	"github.com/filecoin-project/go-f3/gpbft"

	logging "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
)

var log = logging.Logger("f3/certexchange")

const maxResponseLen = 256

// Server is libp2p a certificate exchange server.
type Server struct {
	// Request timeouts. If non-zero, requests will be canceled after the specified duration.
	RequestTimeout time.Duration
	NetworkName    gpbft.NetworkName
	Host           host.Host
	Store          *certstore.Store

	runningLk sync.RWMutex
	stopFunc  context.CancelFunc
}

func (s *Server) withDeadline(ctx context.Context) (context.Context, context.CancelFunc) {
	if s.RequestTimeout > 0 {
		return context.WithTimeout(ctx, s.RequestTimeout)
	}
	return ctx, func() {}
}

func (s *Server) handleRequest(ctx context.Context, stream network.Stream) (_err error) {
	defer func() {
		if perr := recover(); perr != nil {
			_err = fmt.Errorf("panicked in server response: %v", perr)
			log.Errorf("%s\n%s", string(debug.Stack()))
		}
	}()

	if deadline, ok := ctx.Deadline(); ok {
		if err := stream.SetDeadline(deadline); err != nil {
			return err
		}
	}

	br := bufio.NewReader(stream)
	bw := bufio.NewWriter(stream)

	// Request has no variable-length fields, so we don't need a limited reader.
	var req Request
	if err := req.UnmarshalCBOR(br); err != nil {
		log.Debugf("failed to read request from stream: %w", err)
		return err
	}

	limit := req.Limit
	if limit > maxResponseLen {
		limit = maxResponseLen
	}
	var resp ResponseHeader
	if latest := s.Store.Latest(); latest != nil {
		resp.PendingInstance = latest.GPBFTInstance + 1
	}

	if resp.PendingInstance >= req.FirstInstance && req.IncludePowerTable {
		pt, err := s.Store.GetPowerTable(ctx, req.FirstInstance)
		if err != nil {
			log.Errorf("failed to load power table: %w", err)
			return err
		}
		resp.PowerTable = pt
	}

	if err := resp.MarshalCBOR(bw); err != nil {
		log.Debugf("failed to write header to stream: %w", err)
		return err
	}

	if resp.PendingInstance > req.FirstInstance {
		// Only try to return up-to but not including the pending instance we just told the
		// client about. Otherwise we could return instances _beyond_ that which is
		// inconsistent and confusing.
		end := req.FirstInstance + limit
		if end >= resp.PendingInstance {
			end = resp.PendingInstance - 1
		}

		certs, err := s.Store.GetRange(ctx, req.FirstInstance, end)
		if err == nil || errors.Is(err, certstore.ErrCertNotFound) {
			for i := range certs {
				if err := certs[i].MarshalCBOR(bw); err != nil {
					log.Debugf("failed to write certificate to stream: %w", err)
					return err
				}
			}
		} else {
			log.Errorf("failed to load finality certificates: %w", err)
		}
	}
	return bw.Flush()
}

// Start the server.
func (s *Server) Start() error {
	s.runningLk.Lock()
	defer s.runningLk.Unlock()
	if s.stopFunc != nil {
		return fmt.Errorf("certificate exchange already running")
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.stopFunc = cancel
	s.Host.SetStreamHandler(FetchProtocolName(s.NetworkName), func(stream network.Stream) {
		s.runningLk.RLock()
		defer s.runningLk.RUnlock()
		if s.stopFunc == nil {
			_ = stream.Reset()
			return
		}

		// Kill the stream if/when we shutdown the server.
		defer context.AfterFunc(ctx, func() { _ = stream.Reset() })()

		ctx, cancel := s.withDeadline(ctx)
		defer cancel()

		if err := s.handleRequest(ctx, stream); err != nil {
			_ = stream.Reset()
		} else {
			_ = stream.Close()
		}

	})
	return nil
}

// Stop the server.
func (s *Server) Stop() error {
	// Ask the handlers to cancel/stop.
	s.runningLk.RLock()
	cancel := s.stopFunc
	s.runningLk.RUnlock()
	if cancel == nil {
		return nil
	}
	cancel()

	// Wait and finish shutdown.
	s.runningLk.Lock()
	defer s.runningLk.Unlock()
	if s.stopFunc == nil {
		return nil
	}
	s.stopFunc = nil
	s.Host.RemoveStreamHandler(FetchProtocolName(s.NetworkName))

	return nil
}
