package data

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"seckill/app/product/internal/biz"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-redsync/redsync/v4"
)

type ProductRepo struct {
	data *Data
}

func NewProductRepo(data *Data) *ProductRepo {
	return &ProductRepo{data: data}
}

func (r *ProductRepo) FindByID(ctx context.Context, ID int64) (*biz.Product, error) {
	cacheKey := fmt.Sprintf("product:%d", ID)

	user, err := r.getCache(ctx, cacheKey)
	if err == nil {
		return user, nil
	}
	log.Errorf("Error get cache:%v", err)

	sfKey := fmt.Sprintf("sf:product:%d", ID)
	val, err, _ := r.data.sg.Do(sfKey, func() (interface{}, error) {
		lockKey := fmt.Sprintf("lock:product:%d", ID)
		mutex := r.data.rs.NewMutex(lockKey, redsync.WithExpiry(5*time.Second))

		if err := mutex.LockContext(ctx); err != nil {
			time.Sleep(100 * time.Millisecond)
			return r.getCache(ctx, cacheKey)
		}
		defer mutex.Unlock()

		productDoublecheck, err := r.getCache(ctx, cacheKey)
		if err == nil {
			return productDoublecheck, nil
		}

		log.Infof("User %d fetching from DB", ID)
		dbProduct, err := r.data.q.GetProduct(ctx, ID)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, errors.InternalServer("DB_ERROR", "no product")
			}
			return nil, errors.InternalServer("DB_ERROR", "failed to fetch user")
		}
		finalProduct := &biz.Product{
			ID:    dbProduct.ID,
			Price: dbProduct.Price.(int32),
			Stock: dbProduct.Stock.(int32),
		}
		r.setCache(ctx, cacheKey, finalProduct)
		return finalProduct, nil
	})

	if err != nil {
		return nil, err
	}

	return val.(*biz.Product), nil
}

func (r *ProductRepo) getCache(ctx context.Context, key string) (*biz.Product, error) {
	val, err := r.data.rdb.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}
	var product biz.Product
	if err := json.Unmarshal(val, &product); err != nil {
		return nil, err
	}
	return &product, nil
}

func (r *ProductRepo) setCache(ctx context.Context, key string, product *biz.Product) {
	data, err := json.Marshal(product)
	if err != nil {
		log.Errorf("Error marshal user cache: %v", err)
		return
	}
	jitter := time.Duration(rand.Intn(10)) * time.Minute
	exp := jitter + 10*time.Minute
	r.data.rdb.Set(ctx, key, data, exp)
}
