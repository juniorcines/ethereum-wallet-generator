package generators

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/Planxnx/eth-wallet-gen/pkg/progressbar"
	"github.com/Planxnx/eth-wallet-gen/pkg/wallets"
)

type Config struct {
	ProgressBar progressbar.ProgressBar
	DryRun      bool
	Concurrency int
	Number      int
}

type Generator struct {
	walletsRepo *wallets.Repository
	config      Config
}

func New(walletsRepo *wallets.Repository, cfg Config) *Generator {
	return &Generator{
		walletsRepo: walletsRepo,
		config:      cfg,
	}
}

func (g *Generator) Start(ctx context.Context) (err error) {
	bar := g.config.ProgressBar
	defer func() {
		_ = bar.Finish()
		if g.config.DryRun {
			return
		}

		if err := g.walletsRepo.Commit(); err != nil {
			// Ignore error
			log.Printf("Gerate Error: %+v", err)
		}

		if result := g.walletsRepo.Results(); len(result) > 0 {
			fmt.Printf("\n%-42s %s\n", "Address", "Seed")
			fmt.Printf("%-42s %s\n", strings.Repeat("-", 42), strings.Repeat("-", 160))
			fmt.Println(result)
		}
	}()

	var resolvedCount atomic.Int64

	var wg sync.WaitGroup
	// generate wallets without db
	semph := make(chan int, g.config.Concurrency)
	for i := 0; i < g.config.Number || g.config.Number < 0; i++ {
		select {
		case <-ctx.Done():
			return
		default:
			semph <- 1
			wg.Add(1)

			go func() {
				defer func() {
					<-semph
					wg.Done()
				}()

				ok, err := g.walletsRepo.Generate()
				if err != nil {
					// Ignore error
					log.Printf("Gerate Error: %+v", err)
					return
				}
				if !ok {
					return
				}

				resolvedCount.Add(1)
				_ = bar.Increment()
				_ = bar.SetResolved(int(resolvedCount.Load()))
			}()
		}
	}
	wg.Wait()

	return nil
}