package service

import (
	"database/sql"
	"strings"

	"gitlab.scorum.com/blog/api/broadcast/types"
	"gitlab.scorum.com/blog/api/db"
	"gitlab.scorum.com/blog/api/rpc"
	"gitlab.scorum.com/blog/api/utils/postgres"
)

func (blog *Blog) AddCategoryAdmin(op types.Operation) *rpc.Error {
	in := op.(*types.AddCategoryAdminOperation)

	if in.Account != blog.Config.Admin {
		return NewError(rpc.AccessDeniedCode, "access denied")
	}

	_, err := blog.DB.Write.NamedExec(`SELECT add_category(:domain, :label, :localization_key)`,
		db.Category{
			Domain:          in.Domain,
			Label:           in.Label,
			LocalizationKey: in.LocalizationKey,
		})
	if err != nil {
		if isErr, _ := postgres.IsUniqueError(err); isErr {
			return NewError(rpc.CategoryAlreadyExistsCode, "category already exists")
		}

		if isInvalidDomainValueErr(err) {
			return NewError(rpc.InvalidParameterCode, "domain is invalid")
		}
		return WrapError(rpc.InternalErrorCode, err)
	}

	return nil
}

func (blog *Blog) UpdateCategoryAdmin(op types.Operation) *rpc.Error {
	in := op.(*types.UpdateCategoryAdminOperation)

	if in.Account != blog.Config.Admin {
		return NewError(rpc.AccessDeniedCode, "access denied")
	}

	_, err := blog.DB.Write.NamedExec(`SELECT update_category(:domain, :label, :order, :localization_key)`,
		db.Category{
			Domain:          in.Domain,
			Label:           in.Label,
			Order:           in.Order,
			LocalizationKey: in.LocalizationKey,
		})
	if err != nil {
		if isInvalidDomainValueErr(err) {
			return NewError(rpc.InvalidParameterCode, "domain is invalid")
		}

		return WrapError(rpc.InternalErrorCode, err)
	}

	return nil
}

func (blog *Blog) RemoveCategoryAdmin(op types.Operation) *rpc.Error {
	in := op.(*types.RemoveCategoryAdminOperation)

	if in.Account != blog.Config.Admin {
		return NewError(rpc.AccessDeniedCode, "access denied")
	}

	_, err := blog.DB.Write.Exec(`SELECT remove_category($1, $2)`, in.Domain, in.Label)
	if err != nil {
		if isInvalidDomainValueErr(err) {
			return NewError(rpc.InvalidParameterCode, "domain is invalid")
		}
		return WrapError(rpc.InternalErrorCode, err)
	}

	return nil
}

func (blog *Blog) GetCategories(ctx *rpc.Context) {
	var domain string
	if err := ctx.Param(0, &domain); err != nil {
		ctx.WriteError(rpc.InvalidParameterCode, err.Error())
		return
	}

	categories, err := blog.doGetCategories(domain)
	if err != nil {
		ctx.WriteError(err.Code, err.Message)
		return
	}
	ctx.WriteResult(categories)
}

func (blog *Blog) doGetCategories(domain string) ([]*Category, *rpc.Error) {
	var categories []*db.Category
	err := blog.DB.Write.Select(&categories,
		`SELECT "domain", label, "order", localization_key
                FROM categories
				WHERE domain = $1
                ORDER BY "order" ASC`, domain)

	if err != nil {
		if isInvalidDomainValueErr(err) {
			return nil, NewError(rpc.InvalidParameterCode, "domain is invalid")
		}
		return nil, WrapError(rpc.InternalErrorCode, err)
	}
	return toAPICategories(categories), nil
}

func (blog *Blog) GetCategory(ctx *rpc.Context) {
	var domain string
	if err := ctx.Param(0, &domain); err != nil {
		ctx.WriteError(rpc.InvalidParameterCode, err.Error())
		return
	}

	var label string
	if err := ctx.Param(1, &label); err != nil {
		ctx.WriteError(rpc.InvalidParameterCode, err.Error())
		return
	}

	categories, err := blog.doGetCategory(domain, label)
	if err != nil {
		ctx.WriteError(err.Code, err.Message)
		return
	}
	ctx.WriteResult(categories)
}

func (blog *Blog) doGetCategory(domain, label string) (*Category, *rpc.Error) {
	var category db.Category
	err := blog.DB.Read.Get(&category,
		`SELECT "domain", label, "order", localization_key
                FROM categories
				WHERE domain = $1 and label = $2
                ORDER BY "order" ASC`, domain, label)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, NewError(rpc.CategoryNotFoundCode, "category not found")
		}
		if isInvalidDomainValueErr(err) {
			return nil, NewError(rpc.InvalidParameterCode, "domain is invalid")
		}
		return nil, WrapError(rpc.InternalErrorCode, err)
	}
	return toAPICategory(category), nil
}

func isInvalidDomainValueErr(err error) bool {
	return postgres.IsInvalidTextRepresentation(err) && strings.Contains(err.Error(), "enum domain")
}
