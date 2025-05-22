package postgres

import "github.com/lib/pq"

const (
	uniqueViolation           = "unique_violation"
	foreignKeyViolation       = "foreign_key_violation"
	invalidTextRepresentation = "invalid_text_representation"
)

func IsUniqueError(err error) (isUniqueErr bool, constraint string) {
	pqErr, isPqError := err.(*pq.Error)

	if isPqError && pqErr.Code.Name() == uniqueViolation {
		return true, pqErr.Constraint
	}
	return false, ""
}

func IsForeignKeyViolationError(err error) (isForeignKeyViolationErr bool, constraint string) {
	pqErr, isPqError := err.(*pq.Error)

	if isPqError && pqErr.Code.Name() == foreignKeyViolation {
		return true, pqErr.Constraint
	}

	return false, ""
}

func IsInvalidTextRepresentation(err error) bool { // postgres provides for this error only code and message
	pqErr, isPqError := err.(*pq.Error)

	return isPqError && pqErr.Code.Name() == invalidTextRepresentation
}
