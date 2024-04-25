package test

import (
	"fmt"
	"testing"

	"github.com/filecoin-project/go-f3/sim"
	"github.com/stretchr/testify/require"
)

///// Tests for a single instance with no adversaries.

func TestSingleton(t *testing.T) {
	sm, err := sim.NewSimulation(SyncConfig(1), GraniteConfig(), sim.TraceNone)
	require.NoError(t, err)
	a := sm.Base(0).Extend(sm.TipGen.Sample())
	sm.SetChains(sim.ChainCount{Count: 1, Chain: a})

	require.NoErrorf(t, sm.Run(1, MAX_ROUNDS), "%s", sm.Describe())
	expectDecision(t, sm, a.Head())
}

func TestSyncPair(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		config sim.Config
	}{
		{
			name:   "no signing",
			config: SyncConfig(2),
		},
		{
			name:   "bls",
			config: SyncConfig(2).UseBLS(),
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			sm, err := sim.NewSimulation(test.config, GraniteConfig(), sim.TraceNone)
			require.NoError(t, err)
			a := sm.Base(0).Extend(sm.TipGen.Sample())
			sm.SetChains(sim.ChainCount{Count: len(sm.Participants), Chain: a})

			require.NoErrorf(t, sm.Run(1, MAX_ROUNDS), "%s", sm.Describe())
			expectDecision(t, sm, a.Head())
		})
	}
}

func TestASyncPair(t *testing.T) {
	t.Parallel()
	repeatInParallel(t, ASYNC_ITERS, func(t *testing.T, repetition int) {
		sm, err := sim.NewSimulation(AsyncConfig(2, repetition), GraniteConfig(), sim.TraceNone)
		require.NoError(t, err)
		a := sm.Base(0).Extend(sm.TipGen.Sample())
		sm.SetChains(sim.ChainCount{Count: len(sm.Participants), Chain: a})

		require.NoErrorf(t, sm.Run(1, MAX_ROUNDS), "%s", sm.Describe())
		expectDecision(t, sm, a.Head(), sm.Base(0).Head())
	})
}

func TestSyncPairDisagree(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		config sim.Config
	}{
		{
			name:   "no signing",
			config: SyncConfig(2),
		},
		{
			name:   "bls",
			config: SyncConfig(2).UseBLS(),
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			sm, err := sim.NewSimulation(test.config, GraniteConfig(), sim.TraceNone)
			require.NoError(t, err)
			a := sm.Base(0).Extend(sm.TipGen.Sample())
			b := sm.Base(0).Extend(sm.TipGen.Sample())
			sm.SetChains(sim.ChainCount{Count: 1, Chain: a}, sim.ChainCount{Count: 1, Chain: b})

			require.NoErrorf(t, sm.Run(1, MAX_ROUNDS), "%s", sm.Describe())
			// Decide base chain as the only common value.
			expectDecision(t, sm, sm.Base(0).Head())
		})
	}
}

func TestAsyncPairDisagree(t *testing.T) {
	repeatInParallel(t, ASYNC_ITERS, func(t *testing.T, repetition int) {
		sm, err := sim.NewSimulation(AsyncConfig(2, repetition), GraniteConfig(), sim.TraceNone)
		require.NoError(t, err)
		a := sm.Base(0).Extend(sm.TipGen.Sample())
		b := sm.Base(0).Extend(sm.TipGen.Sample())
		sm.SetChains(sim.ChainCount{Count: 1, Chain: a}, sim.ChainCount{Count: 1, Chain: b})

		require.NoErrorf(t, sm.Run(1, MAX_ROUNDS), "%s", sm.Describe())
		// Decide base chain as the only common value.
		expectDecision(t, sm, sm.Base(0).Head())
	})
}

func TestSyncAgreement(t *testing.T) {
	repeatInParallel(t, 50, func(t *testing.T, repetition int) {
		honestCount := 3 + repetition
		sm, err := sim.NewSimulation(SyncConfig(honestCount), GraniteConfig(), sim.TraceNone)
		require.NoError(t, err)
		a := sm.Base(0).Extend(sm.TipGen.Sample())
		sm.SetChains(sim.ChainCount{Count: len(sm.Participants), Chain: a})
		require.NoErrorf(t, sm.Run(1, MAX_ROUNDS), "%s", sm.Describe())
		// Synchronous, agreeing groups always decide the candidate.
		expectDecision(t, sm, a.Head())
	})
}

func TestAsyncAgreement(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	t.Parallel()
	// These iterations are much slower, so we can't test as many participants.
	for n := 3; n <= 16; n++ {
		honestCount := n
		t.Run(fmt.Sprintf("honest count %d", honestCount), func(t *testing.T) {
			repeatInParallel(t, ASYNC_ITERS, func(t *testing.T, repetition int) {
				sm, err := sim.NewSimulation(AsyncConfig(honestCount, repetition), GraniteConfig(), sim.TraceNone)
				require.NoError(t, err)
				a := sm.Base(0).Extend(sm.TipGen.Sample())
				sm.SetChains(sim.ChainCount{Count: len(sm.Participants), Chain: a})

				require.NoErrorf(t, sm.Run(1, MAX_ROUNDS), "%s", sm.Describe())
				expectDecision(t, sm, sm.Base(0).Head(), a.Head())
			})
		})
	}
}

func TestSyncHalves(t *testing.T) {
	t.Parallel()
	repeatInParallel(t, 15, func(t *testing.T, repetition int) {
		honestCount := repetition*2 + 2
		sm, err := sim.NewSimulation(SyncConfig(honestCount), GraniteConfig(), sim.TraceNone)
		require.NoError(t, err)
		a := sm.Base(0).Extend(sm.TipGen.Sample())
		b := sm.Base(0).Extend(sm.TipGen.Sample())
		sm.SetChains(sim.ChainCount{Count: honestCount / 2, Chain: a}, sim.ChainCount{Count: honestCount / 2, Chain: b})

		require.NoErrorf(t, sm.Run(1, MAX_ROUNDS), "%s", sm.Describe())
		// Groups split 50/50 always decide the base.
		expectDecision(t, sm, sm.Base(0).Head())
	})
}

func TestSyncHalvesBLS(t *testing.T) {
	if testing.Short() {
		t.Skip("too slow for testing.Short")
	}

	t.Parallel()
	repeatInParallel(t, 3, func(t *testing.T, repetition int) {
		honestCount := repetition*2 + 2
		sm, err := sim.NewSimulation(SyncConfig(honestCount).UseBLS(), GraniteConfig(), sim.TraceNone)
		require.NoError(t, err)
		a := sm.Base(0).Extend(sm.TipGen.Sample())
		b := sm.Base(0).Extend(sm.TipGen.Sample())
		sm.SetChains(sim.ChainCount{Count: honestCount / 2, Chain: a}, sim.ChainCount{Count: honestCount / 2, Chain: b})

		require.NoErrorf(t, sm.Run(1, MAX_ROUNDS), "%s", sm.Describe())
		// Groups split 50/50 always decide the base.
		expectDecision(t, sm, sm.Base(0).Head())
	})
}

func TestAsyncHalves(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	t.Parallel()
	for n := 4; n <= 20; n += 2 {
		honestCount := n
		t.Run(fmt.Sprintf("honest count %d", honestCount), func(t *testing.T) {
			repeatInParallel(t, ASYNC_ITERS, func(t *testing.T, repetition int) {
				sm, err := sim.NewSimulation(AsyncConfig(honestCount, repetition), GraniteConfig(), sim.TraceNone)
				require.NoError(t, err)
				a := sm.Base(0).Extend(sm.TipGen.Sample())
				b := sm.Base(0).Extend(sm.TipGen.Sample())
				sm.SetChains(sim.ChainCount{Count: honestCount / 2, Chain: a}, sim.ChainCount{Count: honestCount / 2, Chain: b})

				require.NoErrorf(t, sm.Run(1, MAX_ROUNDS), "%s", sm.Describe())
				// Groups split 50/50 always decide the base.
				expectDecision(t, sm, sm.Base(0).Head())
			})
		})
	}
}

func TestRequireStrongQuorumToProgress(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	t.Parallel()
	repeatInParallel(t, ASYNC_ITERS, func(t *testing.T, repetition int) {
		sm, err := sim.NewSimulation(AsyncConfig(30, repetition), GraniteConfig(), sim.TraceNone)
		require.NoError(t, err)
		a := sm.Base(0).Extend(sm.TipGen.Sample())
		b := sm.Base(0).Extend(sm.TipGen.Sample())
		// No strict > quorum.
		sm.SetChains(sim.ChainCount{Count: 20, Chain: a}, sim.ChainCount{Count: 10, Chain: b})

		require.NoErrorf(t, sm.Run(1, MAX_ROUNDS), "%s", sm.Describe())
		// Must decide base.
		expectDecision(t, sm, sm.Base(0).Head())
	})
}

func TestLongestCommonPrefix(t *testing.T) {
	// This test uses a synchronous configuration to ensure timely message delivery.
	// If async, it is possible to decide the base chain if QUALITY messages are delayed.
	sm, err := sim.NewSimulation(SyncConfig(4), GraniteConfig(), sim.TraceNone)
	require.NoError(t, err)
	ab := sm.Base(0).Extend(sm.TipGen.Sample())
	abc := ab.Extend(sm.TipGen.Sample())
	abd := ab.Extend(sm.TipGen.Sample())
	abe := ab.Extend(sm.TipGen.Sample())
	abf := ab.Extend(sm.TipGen.Sample())
	sm.SetChains(
		sim.ChainCount{Count: 1, Chain: abc},
		sim.ChainCount{Count: 1, Chain: abd},
		sim.ChainCount{Count: 1, Chain: abe},
		sim.ChainCount{Count: 1, Chain: abf},
	)

	require.NoErrorf(t, sm.Run(1, MAX_ROUNDS), "%s", sm.Describe())
	// Must decide ab, the longest common prefix.
	expectDecision(t, sm, ab.Head())
}
