package usecases

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/newrelic/go-agent/v3/newrelic"
	"github.com/stretchr/testify/assert"

	"github.com/stone-co/the-amazing-ledger/app/domain/instrumentators"
	"github.com/stone-co/the-amazing-ledger/app/domain/vos"
	"github.com/stone-co/the-amazing-ledger/app/pagination"
	"github.com/stone-co/the-amazing-ledger/app/tests/mocks"
	"github.com/stone-co/the-amazing-ledger/app/tests/testdata"
)

func TestLedgerUseCase_ListAccountEntries(t *testing.T) {
	t.Run("should list account entries successfully", func(t *testing.T) {
		account, err := vos.NewAnalyticAccount(testdata.GenerateAccountPath())
		assert.NoError(t, err)

		mockedRepository := &mocks.RepositoryMock{
			ListAccountEntriesFunc: func(ctx context.Context, req vos.AccountEntryRequest) ([]vos.AccountEntry, pagination.Cursor, error) {
				return []vos.AccountEntry{
					{
						ID:             uuid.New(),
						Version:        vos.NextAccountVersion,
						Operation:      vos.CreditOperation,
						Amount:         100,
						Event:          1,
						CompetenceDate: time.Now().Round(time.Nanosecond),
						Metadata:       nil,
					},
				}, nil, nil
			},
		}
		usecase := NewLedgerUseCase(mockedRepository, instrumentators.NewLedgerInstrumentator(&newrelic.Application{}))

		page, err := pagination.NewPage(nil)
		assert.NoError(t, err)
		got, err := usecase.ListAccountEntries(context.Background(), vos.AccountEntryRequest{
			Account:   account,
			StartDate: time.Now(),
			EndDate:   time.Now(),
			Page:      page,
		})
		assert.NoError(t, err)

		assert.Len(t, got.Entries, 1)
		assert.Nil(t, got.NextPage)
	})

	t.Run("should return empty value if no result found", func(t *testing.T) {
		account, err := vos.NewAnalyticAccount(testdata.GenerateAccountPath())
		assert.NoError(t, err)

		mockedRepository := &mocks.RepositoryMock{
			ListAccountEntriesFunc: func(ctx context.Context, req vos.AccountEntryRequest) ([]vos.AccountEntry, pagination.Cursor, error) {
				return []vos.AccountEntry{}, nil, nil
			},
		}
		usecase := NewLedgerUseCase(mockedRepository, instrumentators.NewLedgerInstrumentator(&newrelic.Application{}))

		page, err := pagination.NewPage(nil)
		assert.NoError(t, err)
		got, err := usecase.ListAccountEntries(context.Background(), vos.AccountEntryRequest{
			Account:   account,
			StartDate: time.Now(),
			EndDate:   time.Now(),
			Page:      page,
		})
		assert.NoError(t, err)

		assert.Len(t, got.Entries, 0)
		assert.Nil(t, got.NextPage)
	})
}
