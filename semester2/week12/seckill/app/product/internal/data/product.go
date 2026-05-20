package data

import (
	"context"
	"database/sql"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"math/rand"
	"time"

	productv1 "seckill/api/product/v1"
	uv1 "seckill/api/user/v1"
	"seckill/app/product/internal/biz"
	"seckill/app/product/internal/data/db"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-redis/redis/v8"
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

	product, err := r.getCache(ctx, cacheKey)
	if err == nil {
		return product, nil
	}
	if !stderrors.Is(err, redis.Nil) {
		log.Errorf("Error get product cache: %v", err)
	}

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

		log.Infof("Product %d fetching from DB", ID)
		dbProduct, err := r.data.q.GetProduct(ctx, ID)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, errors.InternalServer("DB_ERROR", "no product")
			}
			return nil, errors.InternalServer("DB_ERROR", "failed to fetch product")
		}
		finalProduct := &biz.Product{
			ID:    dbProduct.ID,
			Price: dbProduct.Price,
			Stock: dbProduct.Stock,
		}
		r.setCache(ctx, cacheKey, finalProduct)
		return finalProduct, nil
	})

	if err != nil {
		return nil, err
	}

	return val.(*biz.Product), nil
}

func (r *ProductRepo) DeductStock(ctx context.Context, userID int64, ID int64, amount int32) error {
	product, err := r.FindByID(ctx, ID)
	if err != nil {
		log.Errorf("DB_ERROR: %q", err)
		return err
	}

	if product.Stock == 0 {
		log.Infof("Product %d sold out", product.ID)
		return productv1.ErrorSoldOut("Product %d sold out", product.ID)
	}

	_, err = r.data.userclient.DeductBalance(ctx, &uv1.DeductBalanceRequest{Id: userID, Amount: int64(product.Price)})
	if err != nil {
		if uv1.IsUserNotFound(err) {
			log.Errorf("User %d not found", userID)
			return productv1.ErrorServiceBusy("User %d not found", userID)
		}
		if uv1.IsLowBalance(err) {
			log.Errorf("User %d has insufficient balance", userID)
		}
		return err
	}

	rows, err := r.data.q.DeductStock(ctx, db.DeductStockParams{
		ID:    ID,
		Stock: amount,
	})
	if err != nil {
		return errors.InternalServer("DB_ERROR", "failed to deduct stock")
	}
	if rows == 0 {
		return productv1.ErrorSoldOut("product %d sold out or not found", ID)
	}

	cacheKey := fmt.Sprintf("product:%d", ID)
	if err := r.data.rdb.Del(ctx, cacheKey).Err(); err != nil {
		log.Errorf("Error delete cache after deduct: %v", err)
	}

	return nil
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
