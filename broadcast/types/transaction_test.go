package types

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"
	"time"

	scorumtype "github.com/scorum/scorum-go/types"
	"github.com/stretchr/testify/require"
	"gopkg.in/go-playground/validator.v9"
)

var validate *validator.Validate

var trx *Transaction

func init() {
	validate = validator.New()

	exp := time.Unix(100, 0)

	trx = &Transaction{
		RefBlockNum:    16,
		RefBlockPrefix: 1234,
		Expiration: &scorumtype.Time{
			Time: &exp,
		},
		Operations: Operations{
			&FollowOperation{
				Account: "acc1",
				Follow:  "acc2",
			},
		},
		Signatures: []string {
			"206d494f8445d3fb6ba2ab6e5561f83351c27a10a062d2401fcde5126aa0b62f6224220c49eae4f74e2fc069d2de63d88f126b18150dcce85325ffcffc68d0c209",
		},
	}
}

func TestTransaction_Serialize(t *testing.T) {
	_, err := trx.Serialize()
	require.NoError(t, err)
}

func TestTransaction_UnmarshalFollow(t *testing.T) {
	testdata, _ := os.Open("testdata/follow.json")
	bytes, _ := ioutil.ReadAll(testdata)

	var trx Transaction
	require.NoError(t, json.Unmarshal(bytes, &trx))

	require.Len(t, trx.Operations, 1)
	op := trx.Operations[0].(*FollowOperation)
	require.Equal(t, FollowOpType, op.Type())
	require.Equal(t, op.Account, "azucena")
	require.Equal(t, op.Follow, "leonarda")
}

func TestTransaction_UnmarshalUpdateProfile(t *testing.T) {
	testdata, _ := os.Open("testdata/update_profile.json")
	bytes, _ := ioutil.ReadAll(testdata)

	var trx Transaction
	require.NoError(t, json.Unmarshal(bytes, &trx))

	require.Len(t, trx.Operations, 1)
	op := trx.Operations[0].(*UpdateProfileOperation)
	require.Equal(t, UpdateProfileOpType, op.Type())
	require.Equal(t, op.Account, "azucena")
	require.Equal(t, op.DisplayName, "display_name")
	require.Equal(t, op.Location, "location")
	require.Equal(t, op.Bio, "bio")
}

func TestTransaction_ValidationTest(t *testing.T) {
	t.Run("valid_transaction", func(t *testing.T) {
		require.NoError(t, validate.Struct(trx))
	})
	t.Run("no_one_operations", func(t *testing.T) {
		ctrx := *trx
		ctrx.Operations = Operations{}
		require.Error(t, validate.Struct(ctrx))
	})
	t.Run("too_much_operations", func(t *testing.T) {
		ctrx := *trx
		op := FollowOperation{}
		ctrx.Operations = Operations{&op, &op}
		require.Error(t, validate.Struct(ctrx))
	})
	t.Run("no_one_signatures", func(t *testing.T) {
		ctrx := *trx
		ctrx.Signatures = []string{}
		require.Error(t, validate.Struct(ctrx))
	})
	t.Run("empty_signature", func(t *testing.T) {
		ctrx := *trx
		ctrx.Signatures = []string{""}
		require.Error(t, validate.Struct(ctrx))
	})
	t.Run("invalid_expiration", func(t *testing.T) {
		ctrx := *trx
		ctrx.Expiration = nil
		require.Error(t, validate.Struct(ctrx))
	})
}
