package domain

import (
	"context"
	"errors"
	"github.com/jsiebens/ionscale/internal/util"
	"gorm.io/gorm"
)

type Tailnet struct {
	ID   uint64 `gorm:"primary_key;autoIncrement:false"`
	Name string `gorm:"type:varchar(64);unique_index"`
}

func (r *repository) GetOrCreateTailnet(ctx context.Context, name string) (*Tailnet, bool, error) {
	tailnet := &Tailnet{}
	id := util.NextID()

	tx := r.withContext(ctx).Where(Tailnet{Name: name}).Attrs(Tailnet{ID: id}).FirstOrCreate(tailnet)

	if tx.Error != nil {
		return nil, false, tx.Error
	}

	return tailnet, tailnet.ID == id, nil
}

func (r *repository) GetTailnet(ctx context.Context, id uint64) (*Tailnet, error) {
	var t Tailnet
	tx := r.withContext(ctx).Take(&t, "id = ?", id)

	if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if tx.Error != nil {
		return nil, tx.Error
	}

	return &t, nil
}

func (r *repository) ListTailnets(ctx context.Context) ([]Tailnet, error) {
	var tailnets = []Tailnet{}
	tx := r.withContext(ctx).Find(&tailnets)
	if tx.Error != nil {
		return nil, tx.Error
	}
	return tailnets, nil
}

func (r *repository) DeleteTailnet(ctx context.Context, id uint64) error {
	tx := r.withContext(ctx).Delete(&Tailnet{ID: id})
	return tx.Error
}
