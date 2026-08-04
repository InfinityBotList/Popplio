// Package shop is the entry point for shop benefits attached to an entity.
//
// It is a stub: GetShopBenefits is not implemented yet and returns an error.
// The shop's data model lives in popplio/types and is served by the shop
// routes.
package shop

import "errors"

func GetShopBenefits(targetType string, targetId string) ([]string, error) {
	return nil, errors.New("not implemented yet")
}
