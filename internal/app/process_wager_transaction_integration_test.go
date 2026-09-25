package app_test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/cleidison-barradas/betwallet.api/internal/app"
	"github.com/cleidison-barradas/betwallet.api/internal/domain"
	"github.com/cleidison-barradas/betwallet.api/internal/infra/idgen"
	"github.com/cleidison-barradas/betwallet.api/internal/infra/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

func testDatabaseURL() string {
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url
	}
	return "postgres://postgres:postgres@localhost:5432/betwallet?sslmode=disable"
}

type testDeps struct {
	pool         *pgxpool.Pool
	openWallet   *app.OpenWallet
	processWager *app.ProcessWagerTransaction
}

func setupTestDeps(t *testing.T) *testDeps {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, testDatabaseURL())
	if err != nil {
		t.Fatalf("failed to connect to postgres: %v (rode scripts/test-integration.sh, que sobe o banco antes do teste)", err)
	}
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("postgres não respondeu ao ping: %v", err)
	}

	if _, err := pool.Exec(ctx, `TRUNCATE wallets, wager_transactions, wallet_ledger_entries, outbox_entries RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("failed to truncate tables: %v", err)
	}

	ids := idgen.New()
	uow := postgres.NewUnitOfWork(pool)
	wallets := postgres.NewWalletRepository(pool)
	transactions := postgres.NewWagerTransactionRepository(pool)
	ledger := postgres.NewLedgerRepository(pool)
	outbox := postgres.NewOutboxRepository(pool)

	t.Cleanup(func() { pool.Close() })

	return &testDeps{
		pool:         pool,
		openWallet:   app.NewOpenWallet(uow, wallets, transactions, ledger, outbox, ids),
		processWager: app.NewProcessWagerTransaction(uow, wallets, transactions, ledger, outbox, ids),
	}
}
func TestIntegration_OpenWallet_CreatesWalletWithOpeningCredit(t *testing.T) {
	deps := setupTestDeps(t)
	ctx := context.Background()

	balance, _ := domain.Parse("1000.00", "BRL")
	result, err := deps.openWallet.Execute(ctx, app.OpenWalletCommand{
		PlayerID:       "11111111-1111-1111-1111-111111111111",
		InitialBalance: balance,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Balance.String() != "1000.00" {
		t.Errorf("Balance = %s, expected 1000.00", result.Balance)
	}
}

func TestIntegration_ConcurrentBets_OnlyOneSucceeds(t *testing.T) {
	deps := setupTestDeps(t)
	ctx := context.Background()

	initial, _ := domain.Parse("100.00", "BRL")
	wallet, err := deps.openWallet.Execute(ctx, app.OpenWalletCommand{
		PlayerID:       "22222222-2222-2222-2222-222222222222",
		InitialBalance: initial,
	})
	if err != nil {
		t.Fatalf("failed to open wallet: %v", err)
	}

	betAmount, _ := domain.Parse("80.00", "BRL")

	type outcome struct {
		result *app.ProcessWagerTransactionResult
		err    error
	}
	results := make(chan outcome, 2)

	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			res, err := deps.processWager.Execute(ctx, app.ProcessWagerTransactionCommand{
				ProviderID:            "provider-a",
				ExternalTransactionID: fmt.Sprintf("bet-%d", i),
				IdempotencyKey:        fmt.Sprintf("provider-a:bet-%d", i),
				PlayerID:              "22222222-2222-2222-2222-222222222222",
				WalletID:              wallet.WalletID,
				RoundID:               "round-1",
				GameID:                "fortune-chimp",
				Kind:                  domain.KindBet,
				Money:                 betAmount,
			})
			results <- outcome{res, err}
		}(i)
	}

	wg.Wait()
	close(results)

	var processed, rejected int
	for o := range results {
		if o.err != nil {
			t.Fatalf("unexpected transport-level error: %v", o.err)
		}
		switch o.result.Status {
		case "PROCESSED":
			processed++
		case "REJECTED":
			rejected++
		default:
			t.Errorf("unexpected status: %s", o.result.Status)
		}
	}

	if processed != 1 {
		t.Errorf("processed = %d, expected 1", processed)
	}
	if rejected != 1 {
		t.Errorf("rejected = %d, expected 1", rejected)
	}

	var balanceMinorUnits int64
	err = deps.pool.QueryRow(ctx, `SELECT balance_minor_units FROM wallets WHERE id = $1`, wallet.WalletID).Scan(&balanceMinorUnits)
	if err != nil {
		t.Fatalf("failed to read wallet balance: %v", err)
	}
	if balanceMinorUnits != 2000 { // 20.00 BRL em centavos
		t.Errorf("stored balance = %d minor units, expected 2000 (20.00)", balanceMinorUnits)
	}

	var ledgerCount int
	err = deps.pool.QueryRow(ctx, `SELECT COUNT(*) FROM wallet_ledger_entries WHERE wallet_id = $1`, wallet.WalletID).Scan(&ledgerCount)
	if err != nil {
		t.Fatalf("failed to count ledger entries: %v", err)
	}

	if ledgerCount != 2 {
		t.Errorf("ledger entries = %d, expected 2 (1 opening credit + 1 bet debit)", ledgerCount)
	}

	var debitCount int
	err = deps.pool.QueryRow(ctx, `SELECT COUNT(*) FROM wallet_ledger_entries WHERE wallet_id = $1 AND direction = 'DEBIT'`, wallet.WalletID).Scan(&debitCount)
	if err != nil {
		t.Fatalf("failed to count debit entries: %v", err)
	}
	if debitCount != 1 {
		t.Errorf("debit entries = %d, expected 1 (only one bet should have been processed)", debitCount)
	}
}
