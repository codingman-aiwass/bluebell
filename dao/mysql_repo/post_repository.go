package mysql_repo

import (
	"bluebell/models"
	"bluebell/pkg/sqls"
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
