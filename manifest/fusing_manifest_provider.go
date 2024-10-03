package manifest

import (
	"context"
	"fmt"
	"time"

	"github.com/filecoin-project/go-f3/ec"
	"github.com/filecoin-project/go-f3/internal/clock"
	"go.uber.org/multierr"
	"golang.org/x/sync/errgroup"
)

// HeadGetter is the minimal subset of ec.Backend required by the
// FusingManifestProvider.
type HeadGetter interface {
	GetHead(context.Context) (ec.TipSet, error)
}

var _ ManifestProvider = (*FusingManifestProvider)(nil)

// FusingManifestProvider is a ManifestProvider that starts by providing dynamic manifest updates
// then switches to a static manifest when we get within finality of said manifest's bootstrap
// epoch.
type FusingManifestProvider struct {
	ec      HeadGetter
	dynamic ManifestProvider
	static  *Manifest

	manifestCh chan *Manifest

	errgrp     *errgroup.Group
	cancel     context.CancelFunc
	runningCtx context.Context
	clock      clock.Clock
}

func NewFusingManifestProvider(ctx context.Context, ec HeadGetter, dynamic ManifestProvider, static *Manifest) (*FusingManifestProvider, error) {
	if err := static.Validate(); err != nil {
		return nil, err
	}

	clk := clock.GetClock(ctx)
	ctx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	errgrp, ctx := errgroup.WithContext(ctx)

	return &FusingManifestProvider{
		ec:         ec,
		dynamic:    dynamic,
		static:     static,
		errgrp:     errgrp,
		cancel:     cancel,
		runningCtx: ctx,
		clock:      clk,
		manifestCh: make(chan *Manifest, 1),
	}, nil
}

func (m *FusingManifestProvider) ManifestUpdates() <-chan *Manifest {
	return m.manifestCh
}

func (m *FusingManifestProvider) Start(ctx context.Context) error {
	head, err := m.ec.GetHead(ctx)
	if err != nil {
		return fmt.Errorf("failed to determine current head epoch")
	}

	switchEpoch := m.static.BootstrapEpoch - m.static.EC.Finality
	headEpoch := head.Epoch()

	if headEpoch >= switchEpoch {
		m.manifestCh <- m.static
		return nil
	}

	epochDelay := switchEpoch - headEpoch
	start := head.Timestamp().Add(time.Duration(epochDelay) * m.static.EC.Period)

	if err := m.dynamic.Start(ctx); err != nil {
		return err
	}

	m.errgrp.Go(func() (err error) {
		defer func() {
			m.updateManifest(m.static)
			err = multierr.Append(err, m.dynamic.Stop(context.Background()))
		}()
		dynamicUpdates := m.dynamic.ManifestUpdates()

		timer := m.clock.Timer(m.clock.Until(start))
		defer timer.Stop()

		for ctx.Err() == nil {
			select {
			case <-timer.C:
				return nil
			case update := <-dynamicUpdates:
				m.updateManifest(update)
			case <-ctx.Done():
				return
			}
		}
		return
	})

	return nil
}

func (m *FusingManifestProvider) updateManifest(update *Manifest) {
	drain(m.manifestCh)
	m.manifestCh <- update
}

func (m *FusingManifestProvider) Stop(ctx context.Context) error {
	m.cancel()
	return m.errgrp.Wait()
}
