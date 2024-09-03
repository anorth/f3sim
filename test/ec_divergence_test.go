package test

import (
	"math"
	"testing"

	"github.com/filecoin-project/go-f3/gpbft"
	"github.com/filecoin-project/go-f3/sim"
	"github.com/stretchr/testify/require"
)

func TestEcDivergence_AbsoluteDivergenceConvergesOnBase(t *testing.T) {
	t.Parallel()
	const (
		instanceCount     = 14
		divergeAtInstance = 9
	)

	tests := []struct {
		name    string
		options []sim.Option
	}{
		{
			name:    "sync",
			options: syncOptions(),
		},
		{
			name:    "async",
			options: asyncOptions(985623),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			seedFuzzer := uint64(985623)

			// uniformECChainGenerator generates different EC chain per instance but the same
			// chain for all participants. i.e. with no divergence among votes.
			uniformECChainGenerator := sim.NewUniformECChainGenerator(17*seedFuzzer, 5, 10)

			// randomECChainGenerator generates different EC chain per instance per participant, i.e. total disagreement across the network.
			randomECChainGenerator := sim.NewRandomECChainGenerator(23*seedFuzzer, 5, 10)

			// divergeAfterECChainGenerator uses uniformECChainGenerator up until
			// divergeAtInstance and randomECChainGenerator after it. This simulates a
			// scenario where all participants initially propose the same chain at each
			// instance. But then totally diverge to propose different chains only sharing
			// the same base.
			divergeAfterECChainGenerator := &ecChainGeneratorSwitcher{
				switchAtInstance: divergeAtInstance,
				before:           uniformECChainGenerator,
				after:            randomECChainGenerator,
			}

			sm, err := sim.NewSimulation(
				append(test.options,
					sim.AddHonestParticipants(20, divergeAfterECChainGenerator, uniformOneStoragePower),
				)...)
			require.NoError(t, err)
			require.NoErrorf(t, sm.Run(instanceCount, maxRounds), "%s", sm.Describe())

			// Assert that every instance has reached consensus on the expected base chain,
			// where:
			// * before divergeAtInstance the decision must match the chain generated by
			// uniformECChainGenerator, and
			// * after it the decision must match the base of last decision made before
			// divergeAtInstance.
			//
			// Because:
			// * before divergeAtInstance, all nodes propose the same chain via
			// uniformECChainGenerator, and
			// * after it, every node proposes a chain (only sharing base tipset as required
			// by gPBFT)
			instance := sm.GetInstance(0)
			require.NotNil(t, instance, "instance 0")
			latestBaseECChain := instance.BaseChain
			for i := uint64(0); i < instanceCount; i++ {
				instance = sm.GetInstance(i + 1)
				require.NotNil(t, instance, "instance %d", i)

				var wantDecision gpbft.ECChain
				if i < divergeAtInstance {
					wantDecision = divergeAfterECChainGenerator.GenerateECChain(i, *latestBaseECChain.Head(), math.MaxUint64)
					// Sanity check that the chains generated are not the same but share the same
					// base.
					require.Equal(t, wantDecision.Base(), latestBaseECChain.Head())
					require.NotEqual(t, wantDecision.Suffix(), latestBaseECChain.Suffix())
				} else {
					// After divergeAtInstance all nodes propose different chains. Therefore, the
					// only agreeable chain is the base chain of instance before divergeAtInstance.
					wantDecision = latestBaseECChain
				}

				// Assert the consensus is reached at the head of expected chain.
				requireConsensusAtInstance(t, sm, i, *wantDecision.Head())
				latestBaseECChain = instance.BaseChain
			}
		})
	}
}

func TestEcDivergence_PartitionedNetworkConvergesOnChainWithMostPower(t *testing.T) {
	t.Parallel()
	const (
		instanceCount       = 23
		partitionAtInstance = 13
	)

	tests := []struct {
		name    string
		options []sim.Option
	}{
		{
			name:    "sync",
			options: syncOptions(),
		},
		{
			name:    "async",
			options: asyncOptions(4656),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			seedFuzzer := uint64(784523)

			chainGeneratorBeforePartition := sim.NewUniformECChainGenerator(17*seedFuzzer, 5, 10)

			groupOneChainGeneratorAfterPartition := &ecChainGeneratorSwitcher{
				switchAtInstance: partitionAtInstance,
				before:           chainGeneratorBeforePartition,
				after:            sim.NewUniformECChainGenerator(23*seedFuzzer, 8, 9),
			}
			groupTwoChainGeneratorAfterPartition := &ecChainGeneratorSwitcher{
				switchAtInstance: partitionAtInstance,
				before:           chainGeneratorBeforePartition,
				after:            sim.NewUniformECChainGenerator(29*seedFuzzer, 20, 23),
			}

			sm, err := sim.NewSimulation(
				append(test.options,
					sim.AddHonestParticipants(21, groupOneChainGeneratorAfterPartition, uniformOneStoragePower),
					sim.AddHonestParticipants(9, groupTwoChainGeneratorAfterPartition, uniformOneStoragePower),
				)...)
			require.NoError(t, err)
			require.NoErrorf(t, sm.Run(instanceCount, maxRounds), "%s", sm.Describe())

			// Assert that every instance has reached consensus on the expected base chain,
			// where:
			// * before partitionAtInstance the decision must match the chain generated by
			// chainGeneratorBeforePartition, and
			// * after it the decision must match the chains generated by
			// groupOneChainGeneratorAfterPartition.
			//
			// Because:
			//  * before partitionAtInstance, all nodes propose the same chain at each
			//  instance via chainGeneratorBeforePartition, and
			// * after it, they all should converge on the chains generated by the group with
			//  most power, i.e. group one
			instance := sm.GetInstance(0)
			require.NotNil(t, instance, "instance 0")
			latestBaseECChain := instance.BaseChain
			for i := uint64(0); i < instanceCount; i++ {
				instance = sm.GetInstance(i + 1)
				require.NotNil(t, instance, "instance %d", i)

				var wantDecision gpbft.ECChain

				if i < partitionAtInstance {
					// Before partitionAtInstance all participants should converge on the chains
					// generated by chainGeneratorBeforePartition.
					wantDecision = chainGeneratorBeforePartition.GenerateECChain(i, *latestBaseECChain.Head(), math.MaxUint64)
				} else {
					// After partitionAtInstance all participants should converge on the chains
					// generated by groupOneChainGeneratorAfterPartition. Because that group has over
					// 2/3 of power across the network.
					wantDecision = groupOneChainGeneratorAfterPartition.GenerateECChain(i, *latestBaseECChain.Head(), math.MaxUint64)
				}

				// Sanity check that the chains generated are not the same but share the same
				// base.
				require.Equal(t, wantDecision.Base(), latestBaseECChain.Head())
				require.NotEqual(t, wantDecision.Suffix(), latestBaseECChain.Suffix())

				// Assert the consensus is reached at the head of expected chain.
				requireConsensusAtInstance(t, sm, i, wantDecision...)
				latestBaseECChain = instance.BaseChain
			}
		})
	}
}

var _ sim.ECChainGenerator = (*ecChainGeneratorSwitcher)(nil)

type ecChainGeneratorSwitcher struct {
	switchAtInstance uint64
	before           sim.ECChainGenerator
	after            sim.ECChainGenerator
}

func (d *ecChainGeneratorSwitcher) GenerateECChain(instance uint64, base gpbft.TipSet, id gpbft.ActorID) gpbft.ECChain {
	if instance < d.switchAtInstance {
		return d.before.GenerateECChain(instance, base, id)
	}
	return d.after.GenerateECChain(instance, base, id)
}
