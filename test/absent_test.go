package test

import (
	"github.com/filecoin-project/go-f3/adversary"
	"github.com/filecoin-project/go-f3/sim"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestAbsent(t *testing.T) {
	for i := 0; i < 1000; i++ {
		//fmt.Println("Iteration", i)
		sm := sim.NewSimulation(sim.Config{
			HonestCount: 3,
			LatencySeed: int64(i),
			LatencyMean: LATENCY_ASYNC,
		}, GraniteConfig(), sim.TraceNone)
		// Adversary has 1/4 of power.
		sm.SetAdversary(adversary.NewAbsent(99, sm.Network), 1)

		a := sm.Base.Extend(sm.CIDGen.Sample())
		sm.ReceiveChains(sim.ChainCount{Count: len(sm.Participants), Chain: a})

		require.True(t, sm.Run(MAX_ROUNDS), "%s", sm.Describe())
	}
}
