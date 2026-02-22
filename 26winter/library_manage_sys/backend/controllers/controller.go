package controller

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/Dailiduzhou/library_manage_sys/config"
	"github.com/Dailiduzhou/library_manage_sys/models"
	"github.com/Dailiduzhou/library_manage_sys/utils"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrBookNotFound   = errors.New("图书不存在")
	ErrNoStock        = errors.New("图书库存不足")
	ErrRecordNotFound = errors.New("借书记录查询失败")
	ErrBookBorrowed   = errors.New("图书仍在借")
	ErrDeleteBook     = errors.New("图书删除失败")
	ErrDeleteCover    = errors.New("封面删除失败")
)

// @Summary 用户注册
// @Description 创建新用户账号
// @Tags auth
// @Accept json
// @Produce json
// @Param request body models.RegisterRequest true "注册请求"
// @Success 200 {object} models.Response "注册成功"
// @Failure 400 {object} models.Response "参数错误"
// @Failure 409 {object} models.Response "用户已存在"
// @Failure 500 {object} models.Response "服务器错误"
// @Router /api/auth/register [post]
func Register(c *gin.Context) {
	var req models.RegisterRequest
	var err error

	if err = c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Code: 400,
			Msg:  "参数设定错误",
		})
		return
	}

	var existingUser models.User
	err = config.DB.Where("username = ?", req.Username).First(&existingUser).Error
	if err == nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Code: 400,
			Msg:  "用户已存在",
		})
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code: 500,
			Msg:  "查询数据库失败",
		})
		return
	}

	hashedpassword, err := utils.HashPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code: 500,
			Msg:  "密码加密错误",
		})
	}

	newUser := models.User{
		Username: req.Username,
		Password: hashedpassword,
		Role:     "user",
	}

	if err = config.DB.Create(&newUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code: 500,
			Msg:  "创建用户失败",
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Code: 200,
		Msg:  "注册成功",
		Data: gin.H{
			"username": newUser.Username,
			"user_id":  newUser.ID,
		},
	})
}

// @Summary 用户登录
// @Description 用户身份验证
// @Tags auth
// @Accept json
// @Produce json
// @Param request body models.LoginRequest true "登录请求"
// @Success 200 {object} models.Response "登录成功"
// @Failure 400 {object} models.Response "参数错误"
// @Failure 403 {object} models.Response "认证失败"
// @Failure 500 {object} models.Response "服务器错误"
// @Router /api/auth/login [post]
func Login(c *gin.Context) {
	var req models.LoginRequest
	var err error

	if err = c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Code: 400,
			Msg:  "参数设定错误",
		})
		return
	}

	var user models.User
	err = config.DB.Where("username = ?", req.Username).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusForbidden, models.Response{
			Code: 403,
			Msg:  "用户不存在",
		})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code: 500,
			Msg:  "查询数据库失败",
		})
		return
	}

	err = utils.ComparePassword(user.Password, req.Password)
	if err != nil {
		c.JSON(http.StatusForbidden, models.Response{
			Code: 403,
			Msg:  "密码错误",
		})
		return
	}

	session := sessions.Default(c)
	session.Set("user_id", user.ID)
	session.Set("role", user.Role)
	if err := session.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code: 500,
			Msg:  "鉴权组件错误",
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Code: 200,
		Msg:  "登陆成功",
		Data: gin.H{
			"use_id": user.ID,
			"role":   user.Role,
		},
	})
}

// @Summary 用户登出
// @Description 清除会话，登出当前用户
// @Tags auth
// @Security ApiKeyAuth
// @Produce json
// @Success 200 {object} models.Response "登出成功"
// @Failure 500 {object} models.Response "服务器错误"
// @Router /api/auth/logout [post]
func Logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	if err := session.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code: 500,
			Msg:  "登出失败",
		})
	}

	// 希望前端实现跳转登录界面的功能
	c.JSON(http.StatusOK, models.Response{
		Code: 200,
		Msg:  "登出成功",
	})
}

// @Summary 创建图书
// @Description 添加新图书（管理员权限）
// @Tags books
// @Security ApiKeyAuth
// @Accept multipart/form-data
// @Produce json
// @Param title formData string true "书名"
// @Param author formData string true "作者"
// @Param summary formData string false "简介"
// @Param cover formData file false "封面图片"
// @Param initial_stock formData integer true "初始库存" minimum(0)
// @Success 200 {object} models.Response{data=models.Book} "创建成功"
// @Failure 400 {object} models.Response "参数错误"
// @Failure 409 {object} models.Response "图书已存在"
// @Failure 500 {object} models.Response "服务器错误"
// @Router /api/admin/books [post]
func CreateBook(c *gin.Context) {
	var req models.CreateBookRequest

	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Code: 400,
			Msg:  "参数设定错误",
		})
		return
	}

	var existingBook models.Book
	err := config.DB.Where("title = ? AND author = ?", req.Title, req.Author).First(&existingBook).Error
	if err == nil {
		c.JSON(http.StatusConflict, models.Response{
			Code: 409,
			Msg:  "该图书已存在(书名和作者相同)",
		})
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code: 500,
			Msg:  "数据库查询失败",
		})
		return
	}

	finalCoverPath := models.DefaultCoverPath
	if req.Cover != nil && req.Cover.Size > 0 {
		log.Printf("有封面文件上传，大小: %d", req.Cover.Size)
		savePath, err := utils.SaveImages(c, req.Cover)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.Response{
				Code: 500,
				Msg:  "图片保存失败",
			})
			return
		}
		finalCoverPath = savePath
	} else {
		log.Printf("没有封面文件上传或文件为空，使用默认路径: %s", finalCoverPath)
	}
	finalSummary := models.DefaultSummary
	if req.Summary != "" {
		finalSummary = req.Summary
	}

	newBook := models.Book{
		Title:        req.Title,
		Author:       req.Author,
		Summary:      finalSummary,
		CoverPath:    finalCoverPath,
		InitialStock: req.InitialStock,
		Stock:        req.InitialStock,
		TotalStock:   req.InitialStock,
	}

	if err := config.DB.Create(&newBook).Error; err != nil {
		if req.Cover != nil && req.Cover.Size != 0 {
			if err := utils.RemoveFile(finalCoverPath); err != nil {
				c.JSON(http.StatusInternalServerError, models.Response{
					Code: 500,
					Msg:  "删除封面失败",
				})
				return
			}
		}

		c.JSON(http.StatusInternalServerError, models.Response{
			Code: 500,
			Msg:  "创建图书失败",
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Code: 200,
		Msg:  "图书创建成功",
		Data: newBook,
	})
}

// @Summary 获取图书列表
// @Description 按条件查询图书
// @Tags books
// @Security ApiKeyAuth
// @Produce json
// @Param title query string false "按书名模糊查询"
// @Param author query string false "按作者模糊查询"
// @Param summary query string false "按简介模糊查询"
// @Success 200 {object} models.Response{data=[]models.Book} "查询成功"
// @Failure 500 {object} models.Response "数据库错误"
// @Router /api/books [get]
func GetBooks(c *gin.Context) {
	var books []models.Book

	title := c.Query("title")
	author := c.Query("author")
	summary := c.Query("summary")

	query := config.DB.Model(&models.Book{})

	if title != "" {
		query = query.Where("title LIKE ?", "%"+title+"%")
	}
	if author != "" {
		query = query.Where("author LIKE ?", "%"+author+"%")
	}
	if summary != "" {
		query = query.Where("summary LIKE ?", "%"+summary+"%")
	}

	if err := query.Order("id desc").Find(&books).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code: 500,
			Msg:  "数据库查询失败",
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Code: 200,
		Msg:  "查询成功",
		Data: books,
	})
}

// @Summary 更新图书
// @Description 修改图书信息（管理员权限）
// @Tags books
// @Security ApiKeyAuth
// @Accept multipart/form-data
// @Produce json
// @Param id path uint true "图书ID"
// @Param title formData string false "新书名"
// @Param author formData string false "新作者"
// @Param summary formData string false "新简介"
// @Param cover formData file false "新封面"
// @Param stock formData integer false "当前库存" minimum(0)
// @Param total_stock formData integer false "总库存" minimum(0)
// @Success 200 {object} models.Response{data=models.Book} "更新成功"
// @Failure 400 {object} models.Response "参数错误"
// @Failure 404 {object} models.Response "图书不存在"
// @Failure 500 {object} models.Response "服务器错误"
// @Router /api/admin/books/{id} [put]
func UpdateBook(c *gin.Context) {
	id := c.Param("id")
	bookID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Code: 400,
			Msg:  "无效的图书ID",
		})
		return
	}

	var req models.UpdateBookRequest

	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusNotFound, models.Response{
			Code: 404,
			Msg:  "参数设定错误",
		})
		return
	}
	req.ID = uint(bookID)

	tx := config.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var book models.Book
	result := tx.Set("gorm:query_option", "FOR UPDATE").First(&book, req.ID)

	if result.Error != nil {
		tx.Rollback()
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, models.Response{
				Code: 404,
				Msg:  "图书不存在",
			})
		} else {
			c.JSON(http.StatusInternalServerError, models.Response{
				Code: 500,
				Msg:  "数据库查询失败: ",
			})
		}
		return
	}

	updates := make(map[string]interface{})

	if req.Title != "" {
		updates["title"] = req.Title
	}
	if req.Author != "" {
		updates["author"] = req.Author
	}
	if req.Summary != "" {
		updates["summary"] = req.Summary
	}

	if req.Stock >= 0 || req.TotalStock > 0 {
		newStock := book.Stock
		newTotalStock := book.TotalStock

		if req.Stock >= 0 {
			newStock = req.Stock
		}
		if req.TotalStock > 0 {
			newTotalStock = req.TotalStock
		}

		if newStock > newTotalStock {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, models.Response{
				Code: 400,
				Msg:  "当前库存不能大于总库存",
			})
			return
		}

		updates["stock"] = newStock
		updates["total_stock"] = newTotalStock
	}

	finalCoverPath := book.CoverPath
	if req.Cover != nil && req.Cover.Size > 0 {
		log.Printf("有封面文件上传，大小: %d", req.Cover.Size)
		savePath, err := utils.SaveImages(c, req.Cover)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.Response{
				Code: 500,
				Msg:  "图片保存失败",
			})
			return
		}
		finalCoverPath = savePath
	} else {
		log.Printf("没有封面文件上传或文件为空,使用原有路径: %s", finalCoverPath)
	}

	updates["cover_path"] = finalCoverPath

	if len(updates) > 0 {
		if err := tx.Model(&book).Updates(updates).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, models.Response{
				Code: 500,
				Msg:  "更新失败: " + err.Error(),
			})
			return
		}
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code: 500,
			Msg:  "提交事务失败: ",
		})
		return
	}
	config.DB.First(&book, req.ID)

	c.JSON(http.StatusOK, models.Response{
		Code: 200,
		Msg:  "图书更新成功",
		Data: book,
	})
}

// @Summary 删除图书
// @Description 删除指定图书（管理员权限）
// @Tags books
// @Security ApiKeyAuth
// @Produce json
// @Param id path uint true "图书ID"
// @Success 200 {object} models.Response "删除成功"
// @Failure 400 {object} models.Response "参数错误"
// @Failure 404 {object} models.Response "图书不存在"
// @Failure 409 {object} models.Response "图书仍在借阅中"
// @Failure 500 {object} models.Response "服务器错误"
// @Router /api/admin/books/{id} [delete]
func DeleteBooks(c *gin.Context) {
	id := c.Param("id")
	bookID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Code: 400,
			Msg:  "无效的图书ID",
		})
		return
	}

	req := models.FindBookRequest{ID: uint(bookID)}

	err = config.DB.Transaction(func(tx *gorm.DB) error {
		var existingBook models.Book

		if err := tx.First(&existingBook, req.ID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrBookNotFound
			}
			return err
		}

		if existingBook.Stock != existingBook.TotalStock {
			return ErrBookBorrowed
		}

		if err := tx.Delete(&existingBook).Error; err != nil {
			return ErrDeleteBook
		}

		fmt.Printf("[Debug] DB路径: '%s' vs 默认常量: '%s'\n", existingBook.CoverPath, models.DefaultCoverPath)

		if existingBook.CoverPath == models.DefaultCoverPath {
			log.Printf("使用了默认封面,不执行删除")
		} else {
			if err := utils.RemoveFile(existingBook.CoverPath); err != nil {
				return ErrDeleteCover
			}
		}

		// if existingBook.CoverPath == models.DefaultCoverPath {
		// 	log.Printf("使用了默认封面, 跳过物理删除")
		// } else {
		// 	// 这里修改了！即使出错也不 return ErrDeleteCover
		// 	err := utils.RemoveFile(existingBook.CoverPath)
		// 	if err != nil {
		// 		// 只在控制台打印一条警告，不影响 HTTP 返回 200
		// 		fmt.Printf("[Warning] 封面文件删除失败 (不影响图书删除): path=%s, err=%v\n", existingBook.CoverPath, err)
		// 	} else {
		// 		fmt.Printf("[Info] 封面文件已删除: %s\n", existingBook.CoverPath)
		// 	}
		// }

		return nil
	})

	if err != nil {
		if errors.Is(err, ErrBookNotFound) {
			c.JSON(http.StatusNotFound, models.Response{
				Code: 404,
				Msg:  "图书不存在",
			})
			return
		}
		if errors.Is(err, ErrDeleteBook) {
			c.JSON(http.StatusInternalServerError, models.Response{
				Code: 500,
				Msg:  "删除图书失败",
			})
			return
		}
		if errors.Is(err, ErrDeleteCover) {
			c.JSON(http.StatusInternalServerError, models.Response{
				Code: 500,
				Msg:  "封面删除失败",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, models.Response{
			Code: 500,
			Msg:  "系统繁忙,请稍后再试",
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Code: 200,
		Msg:  "删除图书成功",
	})
}

// @Summary 借阅图书
// @Description 创建借阅记录，用户借阅指定图书 (return_date 初始为 null)
// @Tags borrows
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param request body models.FindBookRequest true "借阅请求"
// @Success 200 {object} models.Response{data=models.BorrowRecord} "借阅成功"
// @Failure 400 {object} models.Response "参数错误"
// @Failure 404 {object} models.Response "图书不存在"
// @Failure 409 {object} models.Response "库存不足"
// @Failure 500 {object} models.Response "服务器错误"
// @Router /api/borrows [post]
func BorrowBook(c *gin.Context) {
	var req models.FindBookRequest
	userID := c.GetUint("user_id")
	if userID == 0 {
		log.Println("严重错误: 上下文中没有获取到 user_id")
		c.JSON(http.StatusUnauthorized, models.Response{
			Code: 401,
			Msg:  "用户未登录或认证失效",
		})
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Code: 400,
			Msg:  "参数设定错误",
		})
		return
	}

	var borrowRecord models.BorrowRecord
	err := config.DB.Transaction(func(tx *gorm.DB) error {
		var book models.Book

		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&book, req.ID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrBookNotFound
			}
			return err
		}

		if book.Stock <= 0 {
			return ErrNoStock
		}

		if err := tx.Model(&book).Update("stock", book.Stock-1).Error; err != nil {
			return err
		}

		borrowRecord = models.BorrowRecord{
			UserID:     userID,
			BookID:     req.ID,
			BorrowDate: time.Now(),
			ReturnDate: nil,
			Status:     "borrowed",
		}

		if err := tx.Omit("User", "Book").Create(&borrowRecord).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		if errors.Is(err, ErrBookNotFound) {
			c.JSON(http.StatusNotFound, models.Response{
				Code: 404,
				Msg:  "图书不存在",
			})
			return
		}
		if errors.Is(err, ErrNoStock) {
			c.JSON(http.StatusConflict, models.Response{
				Code: 409,
				Msg:  "图书库存不足",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, models.Response{
			Code: 500,
			Msg:  "系统繁忙,请稍后再试",
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Code: 200,
		Msg:  "借书成功",
		Data: borrowRecord,
	})
}

// @Summary 归还图书
// @Description 更新借阅状态，用户归还图书
// @Tags borrows
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param request body models.FindBookRequest true "归还请求"
// @Success 200 {object} models.Response{data=models.BorrowRecord} "归还成功"
// @Failure 400 {object} models.Response "参数错误"
// @Failure 404 {object} models.Response "记录不存在"
// @Failure 500 {object} models.Response "服务器错误"
// @Router /api/borrows/return [post]
func ReturnBook(c *gin.Context) {
	var req models.FindBookRequest

	userID := c.GetUint("user_id")
	if userID == 0 {
		log.Println("严重错误: 上下文中没有获取到 user_id")
		c.JSON(http.StatusUnauthorized, models.Response{
			Code: 401,
			Msg:  "用户未登录或认证失效",
		})
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.Response{
			Code: 400,
			Msg:  "参数设定错误",
		})
		return
	}

	var borrowRecord models.BorrowRecord
	err := config.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND book_id = ? AND status = ?", userID, req.ID, "borrowed").
			First(&borrowRecord).Error; err != nil {
			return ErrRecordNotFound
		}

		var book models.Book
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&book, req.ID).Error; err != nil {
			if errors.Is(err, ErrBookNotFound) {
				return ErrBookNotFound
			}
			return err
		}

		if err := tx.Model(&book).Update("stock", book.Stock+1).Error; err != nil {
			return err
		}

		if err := tx.Where("user_id = ? AND book_id = ? AND status = ?", userID, req.ID, "borrowed").First(&borrowRecord).Error; err != nil {
			return ErrRecordNotFound
		}

		now := time.Now()
		borrowRecord.ReturnDate = &now
		borrowRecord.Status = "returned"

		if err := tx.Model(&borrowRecord).Updates(models.BorrowRecord{
			ReturnDate: &now,
			Status:     "returned",
		}).Error; err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		if errors.Is(err, ErrBookNotFound) {
			c.JSON(http.StatusNotFound, models.Response{
				Code: 404,
				Msg:  "图书不存在",
			})
			return
		}

		if errors.Is(err, ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, models.Response{
				Code: 404,
				Msg:  "未找到该书的借阅记录或已归还",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, models.Response{
			Code: 500,
			Msg:  "系统错误",
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Code: 200,
		Msg:  "还书成功",
		Data: borrowRecord,
	})
}

// @Summary 查询个人借书记录
// @Description 查询当前登录用户的借阅记录（需登录）
// @Tags borrows
// @Security ApiKeyAuth
// @Produce json
// @Param id path uint true "用户ID"
// @Success 200 {object} models.Response{data=[]models.BorrowRecord} "查询成功"
// @Failure 400 {object} models.Response "权限不足，无法查询他人记录"
// @Failure 404 {object} models.Response "查询成功,无借书记录"
// @Failure 500 {object} models.Response "用户ID解析错误或数据库查询失败"
// @Router /api/records/{id} [post]
func BorrowRecords(c *gin.Context) {
	id := c.Param("id")
	userID, err := strconv.ParseUint(id, 10, 32)

	session := sessions.Default(c)
	userID_session := session.Get("user_id")
	if userID_session == nil || uint(userID) != userID_session.(uint) {
		c.JSON(http.StatusBadRequest, models.Response{
			Code: 400,
			Msg:  "仅可查询自己的借书记录",
		})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code: 500,
			Msg:  "用户ID解析错误",
		})
		return
	}

	var records []models.BorrowRecord
	query := config.DB.Model(&models.BorrowRecord{})

	query = config.DB.Where("user_id = ?", userID)

	if err := query.Order("id DESC").Find(&records).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code: 500,
			Msg:  "数据库查询失败",
		})
		return
	}

	if len(records) == 0 {
		c.JSON(http.StatusNotFound, models.Response{
			Code: 404,
			Msg:  "查询成功,无借书记录",
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Code: 200,
		Msg:  "查询成功",
		Data: records,
	})
}

// @Summary 查询所有借书记录
// @Description 查询所有借阅记录（需管理员权限）
// @Tags records
// @Security ApiKeyAuth
// @Produce json
// @Success 200 {object} models.Response{data=[]models.BorrowRecord} "查询成功"
// @Failure 404 {object} models.Response "查询成功,无借书记录"
// @Failure 500 {object} models.Response "数据库查询失败"
// @Router /api/admin/records [get]
func GetAllBorrowRecords(c *gin.Context) {
	var records []models.BorrowRecord
	query := config.DB.Model(&models.BorrowRecord{})
	if err := query.Order("id DESC").Find(&records).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code: 500,
			Msg:  "数据库查询失败",
		})
		return
	}

	if len(records) == 0 {
		c.JSON(http.StatusNotFound, models.Response{
			Code: 404,
			Msg:  "查询成功,无借书记录",
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Code: 200,
		Msg:  "查询成功",
		Data: records,
	})
}

// @Summary 按用户ID查询借阅记录
// @Description 查询指定用户ID的借阅记录（需管理员权限）
// @Tags records
// @Security ApiKeyAuth
// @Produce json
// @Param id path uint true "用户ID"
// @Success 200 {object} models.Response{data=[]models.BorrowRecord} "查询成功"
// @Failure 404 {object} models.Response "查询成功,无借书记录"
// @Failure 500 {object} models.Response "用户ID解析错误或数据库查询失败"
// @Router /api/admin/records/{id} [post]
func BorrowRecordsByID(c *gin.Context) {
	id := c.Param("id")
	userID, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code: 500,
			Msg:  "用户ID解析错误",
		})
		return
	}

	var records []models.BorrowRecord
	query := config.DB.Model(&models.BorrowRecord{})

	query = config.DB.Where("user_id = ?", userID)

	if err := query.Order("id DESC").Find(&records).Error; err != nil {
		c.JSON(http.StatusInternalServerError, models.Response{
			Code: 500,
			Msg:  "数据库查询失败",
		})
		return
	}

	if len(records) == 0 {
		c.JSON(http.StatusNotFound, models.Response{
			Code: 404,
			Msg:  "查询成功,无借书记录",
		})
		return
	}

	c.JSON(http.StatusOK, models.Response{
		Code: 200,
		Msg:  "查询成功",
		Data: records,
	})
}
