package domain

import (
	"context"
	"errors"
	"github.com/hashicorp/go-bexpr"
	"github.com/mitchellh/pointerstructure"
	"gorm.io/gorm"
)

type AuthFilter struct {
	ID           uint64 `gorm:"primary_key;autoIncrement:false"`
	Expr         string
	AuthMethodID uint64
	AuthMethod   AuthMethod
	TailnetID    *uint64
	Tailnet      *Tailnet
}

type AuthFilters []AuthFilter

func (f *AuthFilter) Evaluate(v interface{}) (bool, error) {
	if f.Expr == "*" {
		return true, nil
	}

	eval, err := bexpr.CreateEvaluator(f.Expr)
	if err != nil {
		return false, err
	}

	result, err := eval.Evaluate(v)
	if err != nil && !errors.Is(err, pointerstructure.ErrNotFound) {
		return false, err
	}

	return result, err
}

func (fs AuthFilters) Evaluate(v interface{}) []Tailnet {
	var tailnetIDMap = make(map[uint64]bool)
	var tailnets []Tailnet

	for _, f := range fs {
		approved, err := f.Evaluate(v)
		if err == nil && approved {
			if f.TailnetID != nil {
				_, alreadyApproved := tailnetIDMap[*f.TailnetID]
				if !alreadyApproved {
					tailnetIDMap[*f.TailnetID] = true
					tailnets = append(tailnets, *f.Tailnet)
				}
			}
		}
	}

	return tailnets
}

func (r *repository) GetAuthFilter(ctx context.Context, id uint64) (*AuthFilter, error) {
	var t AuthFilter
	tx := r.withContext(ctx).Take(&t, "id = ?", id)

	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if tx.Error != nil {
		return nil, tx.Error
	}

	return &t, nil
}

func (r *repository) SaveAuthFilter(ctx context.Context, m *AuthFilter) error {
	tx := r.withContext(ctx).Save(m)

	if tx.Error != nil {
		return tx.Error
	}

	return nil
}

func (r *repository) ListAuthFilters(ctx context.Context) (AuthFilters, error) {
	var filters = []AuthFilter{}

	tx := r.withContext(ctx).
		Preload("Tailnet").
		Preload("AuthMethod").
		Find(&filters)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return filters, nil
}

func (r *repository) ListAuthFiltersByAuthMethod(ctx context.Context, authMethodID uint64) (AuthFilters, error) {
	var filters = []AuthFilter{}

	tx := r.withContext(ctx).
		Preload("Tailnet").
		Preload("AuthMethod").
		Where("auth_method_id = ?", authMethodID).Find(&filters)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return filters, nil
}

func (r *repository) DeleteAuthFilter(ctx context.Context, id uint64) error {
	tx := r.withContext(ctx).Delete(&AuthFilter{ID: id})
	return tx.Error
}

func (r *repository) DeleteAuthFiltersByTailnet(ctx context.Context, tailnetID uint64) error {
	tx := r.withContext(ctx).Where("tailnet_id = ?", tailnetID).Delete(&AuthFilter{})
	return tx.Error
}
