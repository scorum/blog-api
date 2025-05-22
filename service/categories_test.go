package service

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.scorum.com/blog/api/broadcast/types"
	"gitlab.scorum.com/blog/api/rpc"
)

func TestBlog_AddCategory(t *testing.T) {
	defer cleanUp(t)

	op := &types.AddCategoryAdminOperation{
		Account:         leonarda,
		Domain:          "me",
		Label:           "soccer",
		LocalizationKey: "me.soccer",
	}

	t.Run("wrong_domain", func(t *testing.T) {
		cop := *op
		cop.Domain = "not_domain"
		err := handler.AddCategoryAdmin(&cop)
		require.NotNil(t, err)
		require.Equal(t, err.Code, rpc.InvalidParameterCode)
	})

	t.Run("first", func(t *testing.T) {
		require.Nil(t, handler.AddCategoryAdmin(op))
	})

	t.Run("second", func(t *testing.T) {
		err := handler.AddCategoryAdmin(op)
		require.NotNil(t, err)
		require.Equal(t, err.Code, rpc.CategoryAlreadyExistsCode)
	})

	t.Run("not_admin", func(t *testing.T) {
		cop := *op
		cop.Account = sheldon
		err := handler.AddCategoryAdmin(&cop)
		require.NotNil(t, err)
		require.Equal(t, err.Code, rpc.AccessDeniedCode)
	})
}

func TestBlog_AddCategory_Validation(t *testing.T) {
	op := types.AddCategoryAdminOperation{
		Account:         leonarda,
		Domain:          "me",
		Label:           "soccer",
		LocalizationKey: "me.soccer",
	}

	t.Run("valid_op", func(t *testing.T) {
		require.NoError(t, validate.Struct(op))
	})

	t.Run("empty_domain", func(t *testing.T) {
		cop := op
		cop.Domain = ""
		require.Error(t, validate.Struct(cop))
	})

	t.Run("empty_label", func(t *testing.T) {
		cop := op
		cop.Label = ""
		require.Error(t, validate.Struct(cop))
	})

	t.Run("empty_localization_key", func(t *testing.T) {
		cop := op
		cop.LocalizationKey = ""
		require.Error(t, validate.Struct(cop))
	})
}

func TestBlog_UpdateCategory(t *testing.T) {
	defer cleanUp(t)

	domain := "me"

	for i := 1; i < 51; i++ {
		addOp := &types.AddCategoryAdminOperation{
			Account:         leonarda,
			Domain:          domain,
			Label:           strconv.Itoa(i),
			LocalizationKey: "me.soccer",
		}
		require.Nil(t, handler.AddCategoryAdmin(addOp))
	}

	updateOp := &types.UpdateCategoryAdminOperation{
		Account:         leonarda,
		Domain:          "me",
		Label:           "1",
		Order:           5,
		LocalizationKey: "me.soccer.new",
	}

	t.Run("admin", func(t *testing.T) {
		require.Nil(t, handler.UpdateCategoryAdmin(updateOp))
		category, err := handler.doGetCategory("me", updateOp.Label)
		require.Nil(t, err)
		require.Equal(t, category.LocalizationKey, updateOp.LocalizationKey)
		require.Equal(t, category.Order, updateOp.Order)

		categories, err := handler.doGetCategories(domain)
		require.Nil(t, err)

		for i, category := range categories {
			require.Equal(t, category.Order, uint32(i+1))
		}

		updateOp.Order = 30
		require.Nil(t, handler.UpdateCategoryAdmin(updateOp))
		categories, err = handler.doGetCategories(domain)
		require.Nil(t, err)
		for i, category := range categories {
			require.Equal(t, category.Order, uint32(i+1))
		}

		updateOp.Order = 1
		require.Nil(t, handler.UpdateCategoryAdmin(updateOp))
		categories, err = handler.doGetCategories(domain)
		require.Nil(t, err)
		for i, category := range categories {
			require.Equal(t, category.Order, uint32(i+1))
			require.Equal(t, category.Label, strconv.Itoa(i+1))
		}
	})

	t.Run("not_admin", func(t *testing.T) {
		cop := *updateOp
		cop.Account = sheldon
		err := handler.UpdateCategoryAdmin(&cop)
		require.NotNil(t, err)
		require.Equal(t, err.Code, rpc.AccessDeniedCode)
	})
}

func TestBlog_UpdateCategory_Validation(t *testing.T) {
	op := types.UpdateCategoryAdminOperation{
		Account:         leonarda,
		Domain:          "me",
		Label:           "soccer",
		Order:           1,
		LocalizationKey: "me.soccer",
	}

	t.Run("valid_op", func(t *testing.T) {
		require.NoError(t, validate.Struct(op))
	})

	t.Run("empty_domain", func(t *testing.T) {
		cop := op
		cop.Domain = ""
		require.Error(t, validate.Struct(cop))
	})

	t.Run("empty_label", func(t *testing.T) {
		cop := op
		cop.Label = ""
		require.Error(t, validate.Struct(cop))
	})

	t.Run("empty_localization_key", func(t *testing.T) {
		cop := op
		cop.LocalizationKey = ""
		require.Error(t, validate.Struct(cop))
	})
}

func TestBlog_RemoveCategory(t *testing.T) {
	defer cleanUp(t)

	domain := "me"

	for i := 1; i < 51; i++ {
		addOp := &types.AddCategoryAdminOperation{
			Account:         leonarda,
			Domain:          domain,
			Label:           strconv.Itoa(i),
			LocalizationKey: "me.soccer",
		}
		require.Nil(t, handler.AddCategoryAdmin(addOp))
	}

	categories, err := handler.doGetCategories(domain)
	require.Nil(t, err)
	require.Len(t, categories, 50)

	removeOp := &types.RemoveCategoryAdminOperation{
		Account: leonarda,
		Domain:  "me",
		Label:   "30",
	}

	t.Run("admin", func(t *testing.T) {
		require.Nil(t, handler.RemoveCategoryAdmin(removeOp))
		categories, err = handler.doGetCategories(domain)
		require.Nil(t, err)
		require.Len(t, categories, 49)

		for i, category := range categories {
			require.NotEqual(t, category.Label, "30")
			require.Equal(t, category.Order, uint32(i+1))
		}
	})

	t.Run("not_admin", func(t *testing.T) {
		cop := *removeOp
		cop.Account = sheldon
		err := handler.RemoveCategoryAdmin(&cop)
		require.NotNil(t, err)
		require.Equal(t, err.Code, rpc.AccessDeniedCode)
	})
}

func TestBlog_RemoveCategory_Validation(t *testing.T) {
	op := types.RemoveCategoryAdminOperation{
		Account: leonarda,
		Domain:  "me",
		Label:   "soccer",
	}

	t.Run("valid_op", func(t *testing.T) {
		require.NoError(t, validate.Struct(op))
	})

	t.Run("empty_domain", func(t *testing.T) {
		cop := op
		cop.Domain = ""
		require.Error(t, validate.Struct(cop))
	})

	t.Run("empty_label", func(t *testing.T) {
		cop := op
		cop.Label = ""
		require.Error(t, validate.Struct(cop))
	})
}

func TestBlog_GetCategories(t *testing.T) {
	defer cleanUp(t)

	require.Nil(t, handler.AddCategoryAdmin(&types.AddCategoryAdminOperation{
		Account:         leonarda,
		Domain:          "me",
		Label:           "hockey",
		LocalizationKey: "me.hockey",
	}))

	require.Nil(t, handler.AddCategoryAdmin(&types.AddCategoryAdminOperation{
		Account:         leonarda,
		Domain:          "me",
		Label:           "soccer",
		LocalizationKey: "me.soccer",
	}))

	require.Nil(t, handler.AddCategoryAdmin(&types.AddCategoryAdminOperation{
		Account:         leonarda,
		Domain:          "com",
		Label:           "soccer",
		LocalizationKey: "com.soccer",
	}))

	require.Nil(t, handler.AddCategoryAdmin(&types.AddCategoryAdminOperation{
		Account:         leonarda,
		Domain:          "tc",
		Label:           "soccer",
		LocalizationKey: "tc.soccer",
	}))

	require.Nil(t, handler.AddCategoryAdmin(&types.AddCategoryAdminOperation{
		Account:         leonarda,
		Domain:          "in",
		Label:           "soccer",
		LocalizationKey: "in.soccer",
	}))

	require.Nil(t, handler.AddCategoryAdmin(&types.AddCategoryAdminOperation{
		Account:         leonarda,
		Domain:          "fr",
		Label:           "soccer",
		LocalizationKey: "fr.soccer",
	}))

	categories, err := handler.doGetCategories("me")
	require.Nil(t, err)
	require.Len(t, categories, 2)
	require.Equal(t, "hockey", categories[0].Label)
	require.EqualValues(t, 1, categories[0].Order)
	require.Equal(t, "soccer", categories[1].Label)
	require.EqualValues(t, 2, categories[1].Order)

	categories, err = handler.doGetCategories("com")
	require.Nil(t, err)
	require.Len(t, categories, 1)
	require.Equal(t, "com.soccer", categories[0].LocalizationKey)

	categories, err = handler.doGetCategories("in")
	require.Nil(t, err)
	require.Len(t, categories, 1)
	require.Equal(t, "in.soccer", categories[0].LocalizationKey)

	t.Run("get_not_existsing_category", func(t *testing.T) {
		_, err := handler.doGetCategory("me", "not existing")
		require.NotNil(t, err)
		require.Equal(t, err.Code, rpc.CategoryNotFoundCode)
	})

	t.Run("get_existing_category", func(t *testing.T) {
		category, err := handler.doGetCategory("me", "hockey")
		require.Nil(t, err)
		require.Equal(t, category.LocalizationKey, "me.hockey")
	})
}
