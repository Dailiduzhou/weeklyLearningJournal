package data

import (
	"context"
	"database/sql"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"math/rand"
	"time"

	userv1 "seckill/api/user/v1"
	"seckill/app/user/internal/biz"
	"seckill/app/user/internal/data/db"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-redis/redis/v8"
	"github.com/go-redsync/redsync/v4"
)

type UserRepo struct {
	data *Data
}

func NewUserRepo(data *Data) *UserRepo {
	return &UserRepo{data: data}
}

func (r *UserRepo) Create(ctx context.Context) (*biz.User, error) {
	user, err := r.data.q.CreatUser(ctx)
	if err != nil {
		return nil, errors.InternalServer("DB_ERROR", "failed to create")
	}
	return &biz.User{ID: user.ID, Balance: user.Balance}, nil
}

func (r *UserRepo) FindByID(ctx context.Context, ID int64) (*biz.User, error) {
	cacheKey := fmt.Sprintf("user:%d", ID)

	user, err := r.getCache(ctx, cacheKey)
	if err == nil {
		return user, nil
	}
	if !stderrors.Is(err, redis.Nil) {
		log.Errorf("Error get user cache: %v", err)
	}

	sfKey := fmt.Sprintf("sf:user:%d", ID)
	val, err, _ := r.data.sg.Do(sfKey, func() (interface{}, error) {
		lockKey := fmt.Sprintf("lock:user:%d", ID)
		mutex := r.data.rs.NewMutex(lockKey, redsync.WithExpiry(5*time.Second))

		if err := mutex.LockContext(ctx); err != nil {
			time.Sleep(100 * time.Millisecond)
			return r.getCache(ctx, cacheKey)
		}
		defer mutex.Unlock()

		userDoublecheck, err := r.getCache(ctx, cacheKey)
		if err == nil {
			return userDoublecheck, nil
		}

		log.Infof("User %d fetching from DB", ID)
		dbUser, err := r.data.q.GetUser(ctx, ID)
		if err != nil {
			if err == sql.ErrNoRows {
				return nil, userv1.ErrorUserNotFound("User %d not found", ID)
			}
			return nil, errors.InternalServer("DB_ERROR", "failed to fetch user")
		}
		finaldbUser := &biz.User{ID: dbUser.ID, Balance: dbUser.Balance}
		r.setCache(ctx, cacheKey, finaldbUser)
		return finaldbUser, nil
	})

	if err != nil {
		return nil, err
	}

	return val.(*biz.User), nil
}

func (r *UserRepo) getCache(ctx context.Context, key string) (*biz.User, error) {
	val, err := r.data.rdb.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}
	var user biz.User
	if err := json.Unmarshal(val, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepo) setCache(ctx context.Context, key string, user *biz.User) {
	data, err := json.Marshal(user)
	if err != nil {
		log.Errorf("Error marshal user cache: %v", err)
		return
	}
	jitter := time.Duration(rand.Intn(10)) * time.Minute
	exp := jitter + 10*time.Minute
	r.data.rdb.Set(ctx, key, data, exp)
}

func (r *UserRepo) DeducBalance(ctx context.Context, ID int64, amount int32) error {
	n, err := r.data.q.DeductBalance(ctx, db.DeductBalanceParams{
		ID:      ID,
		Balance: amount,
	})
	if err != nil {
		return errors.InternalServer("DB_ERROR", "failed to deduct balance")
	}
	if n == 0 {
		_, err := r.data.q.GetUser(ctx, ID)
		if err != nil {
			if err == sql.ErrNoRows {
				return userv1.ErrorUserNotFound("user %d not found", ID)
			}
			return errors.InternalServer("DB_ERROR", "failed to fetch user after deduct balance miss")
		}
		return userv1.ErrorLowBalance("user %d has insufficient balance", ID)
	}

	cacheKey := fmt.Sprintf("user:%d", ID)
	if err := r.data.rdb.Del(ctx, cacheKey).Err(); err != nil {
		log.Errorf("Error delete cache after deduct: %v", err)
	}

	return nil
}
