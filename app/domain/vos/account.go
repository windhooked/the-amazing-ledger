package vos

import (
	"strings"

	"github.com/stone-co/the-amazing-ledger/app"
)

// Account represents a countability account which, underneath, can be a single ledger account or a group of accounts.
// It's value is composed of labels, which can only contains letters (uppercase will be lowered),
// numbers and underscores, and they are separated  by a dot (.). A label cannot be empty, so the accounts
// 'foo.', '.foo', 'foo..bar' are all invalid. Also, each label has a maximum size of 255 characters with
// a maximum number of total labels of 65535.
//
// When the account represents a group, it has a more flexible syntax, allowing a wildcard '*' in a label,
// which follows the behavior described in the Postgres docs (https://www.postgresql.org/docs/current/ltree.html).
//
// The fist label if a given account is called 'class', and it can only be one of the following:
//  - liability
//  - asset
//  - revenue
//  - expense
//  - equity
//  - conciliate_credit
//  - conciliate_debit
//
// Some examples:
//  - asset.account.treasury
//  - liability.available.96a131a8_c4ac_495e_8971_fcecdbdd003a
//  - liability.available.96a131a8_c4ac_495e_8971_fcecdbdd003a.some_detail
//  - liability.clients.available.96a131a8_c4ac_495e_8971_fcecdbdd003a.detail1.detail2
//  - asset.*.treasury
type Account struct {
	accountType AccountType
	value       string
}

func (a Account) Value() string {
	return a.value
}

func (a Account) Type() AccountType {
	return a.accountType
}

// AccountType indicates what the given account represents, being either analytic or a synthetic.
type AccountType uint8

const (
	Analytic AccountType = iota + 1
	Synthetic
)

// Available classes
const (
	asset            = "asset"
	conciliateCredit = "conciliate_credit"
	conciliateDebit  = "conciliate_debit"
	equity           = "equity"
	expense          = "expense"
	liability        = "liability"
	revenue          = "revenue"
)

// Symbols
const (
	lowerLetterStart = 'a'
	lowerLetterEnd   = 'z'
	digitStart       = '0'
	digitEnd         = '9'
	upperLetterStart = 'A'
	upperLetterEnd   = 'Z'
	underscore       = '_'
	dot              = '.'
	star             = '*'
)

// Limits
const (
	maxLabelSize  uint = 256
	maxComponents uint = 65535
)

type state struct {
	totalComponents  uint
	componentSize    uint
	strategy         AccountType
	needsLower       bool
	componentHasStar bool
}

// NewAnalyticAccount creates a new valid Account, which can only be of analytic type.
func NewAnalyticAccount(account string) (Account, error) {
	return newAccount(account, true)
}

// NewAccount creates a new valid Account.
func NewAccount(account string) (Account, error) {
	return newAccount(account, false)
}

func newAccount(account string, analyticOnly bool) (Account, error) {
	if len(account) == 0 {
		return Account{}, app.ErrInvalidAccountStructure
	}

	st := &state{strategy: Analytic}

	var (
		r   rune
		err error
	)
	for _, r = range account {
		switch _ = r; {
		case r >= lowerLetterStart && r <= lowerLetterEnd:
			st.componentSize += 1
		case r >= digitStart && r <= digitEnd:
			st.componentSize += 1
		case r >= upperLetterStart && r <= upperLetterEnd:
			st.componentSize += 1
			st.needsLower = true
		case r == underscore:
			st.componentSize += 1
		case r == dot:
			err = treatDot(account, st)
		case r == star:
			if analyticOnly {
				err = app.ErrInvalidSingleAccountComponentCharacters
				break
			}
			err = treatStar(st)
		default:
			err = app.ErrInvalidAccountComponentCharacters
		}

		if err != nil {
			return Account{}, err
		}
	}

	if r == dot {
		return Account{}, app.ErrInvalidAccountComponentSize
	}

	if st.totalComponents == 0 && !st.componentHasStar {
		switch account[:st.componentSize] {
		case asset, conciliateCredit, conciliateDebit, equity, expense, revenue, liability:
		default:
			return Account{}, app.ErrAccountPathViolation
		}
	} else if st.totalComponents < 2 && st.strategy != Synthetic {
		return Account{}, app.ErrInvalidAccountStructure
	}

	if st.needsLower {
		account = lowerAccount(account)
	}

	return Account{
		value:       account,
		accountType: st.strategy,
	}, nil
}

func treatDot(account string, st *state) error {
	// Check if the current component is empty or greater than maximum.
	if st.componentSize == 0 || st.componentSize > maxLabelSize {
		return app.ErrInvalidAccountComponentSize
	}

	// Checks if the account has a valid class and if number of components is greater than maximum.
	if st.totalComponents == 0 && !st.componentHasStar {
		switch account[:st.componentSize] {
		case asset, conciliateCredit, conciliateDebit, equity, expense, revenue, liability:
		default:
			return app.ErrAccountPathViolation
		}
	} else if st.totalComponents >= maxComponents {
		return app.ErrInvalidAccountStructure
	}

	st.totalComponents += 1
	st.componentSize = 0
	st.componentHasStar = false

	return nil
}

func treatStar(st *state) error {
	// Checks is the current component already have a '*'.
	if st.componentHasStar {
		return app.ErrInvalidAccountStructure
	}

	st.strategy = Synthetic
	st.componentSize += 1
	st.componentHasStar = true

	return nil
}

func lowerAccount(account string) string {
	const (
		start uint8 = 'A'
		end   uint8 = 'Z'
		delta       = uint8('a') - uint8('A')
	)

	var b strings.Builder
	b.Grow(len(account))

	for i := 0; i < len(account); i++ {
		c := account[i]
		if start <= c && c <= end {
			c += delta
		}
		b.WriteByte(c)
	}
	return b.String()
}
