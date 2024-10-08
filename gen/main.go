package main

import (
	"fmt"
	"os"

	"github.com/filecoin-project/go-f3/certexchange"
	"github.com/filecoin-project/go-f3/certs"
	"github.com/filecoin-project/go-f3/cmd/f3/msgdump"
	"github.com/filecoin-project/go-f3/gpbft"
	gen "github.com/whyrusleeping/cbor-gen"
	"golang.org/x/sync/errgroup"
)

//go:generate go run .

func main() {
	var eg errgroup.Group
	eg.Go(func() error {
		return gen.WriteTupleEncodersToFile("../gpbft/cbor_gen.go", "gpbft",
			gpbft.TipSet{},
			gpbft.GMessage{},
			gpbft.SupplementalData{},
			gpbft.Payload{},
			gpbft.Justification{},
			gpbft.PowerEntry{},
			gpbft.PowerEntries{},
		)
	})
	eg.Go(func() error {
		return gen.WriteTupleEncodersToFile("../certs/cbor_gen.go", "certs",
			certs.PowerTableDelta{},
			certs.PowerTableDiff{},
			certs.FinalityCertificate{},
		)
	})
	eg.Go(func() error {
		return gen.WriteTupleEncodersToFile("../certexchange/cbor_gen.go", "certexchange",
			certexchange.Request{},
			certexchange.ResponseHeader{},
		)
	})
	eg.Go(func() error {
		return gen.WriteTupleEncodersToFile("../cmd/f3/msgdump/cbor_gen.go", "msgdump",
			msgdump.GMessageEnvelope{},
			msgdump.GMessageEnvelopeDeferred{},
		)
	})
	if err := eg.Wait(); err != nil {
		fmt.Printf("Failed to complete cborg_gen: %v\n", err)
		os.Exit(1)
	}
}
