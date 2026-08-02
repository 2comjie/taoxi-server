package store

import (
	"context"
	"fmt"

	entsql "entgo.io/ent/dialect/sql"
	paymentent "github.com/2comjie/taoxi-server/app/Api/payment/internal/store/ent"
	paymentTypes "github.com/2comjie/taoxi-server/app/Api/payment/types"
)

type Store struct {
	client *paymentent.Client
}

func New(driver *entsql.Driver) *Store {
	return &Store{
		client: paymentent.NewClient(paymentent.Driver(driver)),
	}
}

func (s *Store) Migrate(ctx context.Context) error {
	if err := s.client.Schema.Create(ctx); err != nil {
		return fmt.Errorf("payment: 创建支付订单表失败: %w", err)
	}
	return nil
}

func (s *Store) FindPendingOrder(ctx context.Context, uid uint64, paymentType paymentTypes.PaymentType, productId int32) (*paymentent.PaymentOrder, bool, error) {

}
