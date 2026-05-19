package data

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"seckill/app/user/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-redsync/redsync/v4"
)

var USER_NOT_FOUND = errors.New("USER_NOT_FOUND")

type UserRepo struct {
	data *Data
}

func NewUserRepo(data *Data) *UserRepo {
	return &UserRepo{data: data}
}

func (r *UserRepo) Create(ctx context.Context) (*biz.User, error) {
	userID, err := r.data.q.CreatUser(ctx)
	if err != nil {
		return nil, err
	}
	return &biz.User{ID: userID}, nil
}

func (r *UserRepo) FindByID(ctx context.Context, ID int64) (*biz.User, error) {
	cacheKey := fmt.Sprintf("user:%d", ID)

	user, err := r.getCache(ctx, cacheKey)
	if err == nil {
		return user, nil
	}
	if err != nil {
		log.Errorf("Error get cache:%v", err)
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
		dbUserID, err := r.data.q.GetUser(ctx, ID)
		if err != nil {
			return nil, USER_NOT_FOUND
		}
		dbUser := &biz.User{ID: dbUserID}
		r.setCache(ctx, cacheKey, dbUser)
		return dbUser, nil
	})

	if err != nil {
		return nil, err
	}

	return val.(*biz.User), nil
}

func (r *UserRepo) getCache(ctx context.Context, key string) (*biz.User, error) {
	val, err := r.data.rdb.Get(ctx, key).Int64()
	if err != nil {
		return nil, err
	}
	return &biz.User{ID: val}, nil
}

func (r *UserRepo) setCache(ctx context.Context, key string, user *biz.User) {
	jitter := time.Duration(rand.Int63n(10))
	exp := jitter + 10*time.Minute
	r.data.rdb.Set(ctx, key, user.ID, exp)
}
