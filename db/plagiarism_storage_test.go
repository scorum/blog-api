package db

import (
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestPlagiarismStorageInsert(t *testing.T) {
	defer cleanUp(t)

	permlink := "perm"

	registerAccount(t, leonarda)

	ps := NewPlagiarismStorage(dbWrite)
	p := PostPlagiarism{
		Author:            leonarda,
		Permlink:          permlink,
		LastCheckAt:       time.Time{},
		UniquenessPercent: 0.0,
		Urls:              nil,
		ChecksNum:         1,
		Status:            PlagiarismStatusPending,
	}
	require.NoError(t, ps.Insert(p))
	require.NoError(t, ps.Insert(p))
}

func TestPlagiarismStorageUpsert(t *testing.T) {
	defer cleanUp(t)

	permlink := "perm"

	registerAccount(t, leonarda)

	ps := NewPlagiarismStorage(dbWrite)
	p := PostPlagiarism{
		Author:            leonarda,
		Permlink:          permlink,
		LastCheckAt:       time.Time{},
		UniquenessPercent: 0.0,
		Urls:              nil,
		ChecksNum:         1,
		Status:            PlagiarismStatusPending,
	}
	require.NoError(t, ps.Upsert(p))
	require.NoError(t, ps.Upsert(p))
}

func TestPlagiarismStorageGet(t *testing.T) {
	defer cleanUp(t)

	permlink := "perm"

	registerAccount(t, leonarda)

	ps := NewPlagiarismStorage(dbWrite)
	p := PostPlagiarism{
		Author:            leonarda,
		Permlink:          permlink,
		LastCheckAt:       time.Time{},
		UniquenessPercent: 0.0,
		Urls: PlagiarismUrls{
			PlagiarismUrl{
				Url:     "test",
				Title:   "title",
				Plagiat: 1.0,
			},
		},
		ChecksNum: 1,
		Status:    PlagiarismStatusPending,
	}
	require.NoError(t, ps.Upsert(p))

	pp, err := ps.Get(leonarda, permlink)

	require.NoError(t, err)
	require.EqualValues(t, pp.Author, p.Author)
	require.EqualValues(t, pp.Status, p.Status)
	require.EqualValues(t, pp.UniquenessPercent, p.UniquenessPercent)
	require.EqualValues(t, pp.ChecksNum, p.ChecksNum)
	require.EqualValues(t, len(pp.Urls), len(p.Urls))
}

func TestPlagiarismStorageUpdateStatus(t *testing.T) {
	defer cleanUp(t)

	permlink := "perm"

	registerAccount(t, leonarda)

	ps := NewPlagiarismStorage(dbWrite)
	p := PostPlagiarism{
		Author:            leonarda,
		Permlink:          permlink,
		LastCheckAt:       time.Time{},
		UniquenessPercent: 0.0,
		Urls:              nil,
		ChecksNum:         1,
		Status:            PlagiarismStatusFailed,
	}
	require.NoError(t, ps.Upsert(p))

	pp, err := ps.Get(leonarda, permlink)
	require.NoError(t, err)
	require.EqualValues(t, pp.Status, p.Status)

	p.Status = PlagiarismStatusPending
	require.NoError(t, ps.UpdateStatus(leonarda, permlink, PlagiarismStatusPending))

	pp, err = ps.Get(leonarda, permlink)
	require.NoError(t, err)
	require.EqualValues(t, pp.Status, PlagiarismStatusPending)
}
