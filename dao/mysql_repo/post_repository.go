package mysql_repo

import (
	"bluebell/models"
	"bluebell/pkg/sqls"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var PostRepository = newPostRepository()

func newPostRepository() *postRepository { return &postRepository{} }

type postRepository struct{}

func (r *postRepository) Create(db *gorm.DB, t *models.Post) (err error) {
	err = db.Create(t).Error
	return
}

func (r *postRepository) Get(db *gorm.DB, id int64) *models.Post {
	ret := &models.Post{}
	if err := db.First(ret, "post_id = ?", id).Error; err != nil {
		return nil
	}
	return ret
}

func (r *postRepository) Take(db *gorm.DB, where ...interface{}) *models.Post {
	ret := &models.Post{}
	if err := db.Take(ret, where...).Error; err != nil {
		return nil
	}
	return ret
}

func (r *postRepository) Find(db *gorm.DB, cnd *sqls.Cnd) (list []models.Post) {
	cnd.Find(db, &list)
	return
}

func (r *postRepository) Count(db *gorm.DB, cnd *sqls.Cnd) int64 {
	return cnd.Count(db, &models.Post{})
}

func (r *postRepository) FindPageByCnd(db *gorm.DB, cnd *sqls.Cnd) (list []models.Post, paging *sqls.Paging) {
	cnd.Find(db, &list)
	count := cnd.Count(db, &models.Post{})

	paging = &sqls.Paging{
		Page:  cnd.Paging.Page,
		Limit: cnd.Paging.Limit,
		Total: count,
	}
	return

}

func (r *postRepository) DeletePostInfo(db *gorm.DB, postId int64) (err error) {
	tx := db.Begin()
	if err = tx.Error; err != nil {
		zap.L().Error("create transaction failed in DeletePostInfo()", zap.Error(err))
		return err
	}
	if err = tx.Delete(&models.Post{}, "post_id = ?", postId).Error; err != nil {
		zap.L().Error("delete post failed in DeletePostInfo()", zap.Error(err))
		tx.Rollback()
		return err
	}
	if err = tx.Delete(&models.Like{}, "post_id = ?", postId).Error; err != nil {
		zap.L().Error("delete post in like  failed in DeletePostInfo()", zap.Error(err))
		tx.Rollback()
		return err
	}
	if err = tx.Delete(&models.Comment{}, "post_id = ?", postId).Error; err != nil {
		zap.L().Error("delete posts' comment failed in DeletePostInfo()", zap.Error(err))
		tx.Rollback()
		return err
	}
	if err = tx.Commit().Error; err != nil {
		zap.L().Error("commit transaction failed in DeletePostInfo()", zap.Error(err))
		tx.Rollback()
		return err
	}
	zap.L().Info("commit transaction successfully in DeletePostInfo()")
	return nil
}
