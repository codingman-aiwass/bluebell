package mysql_repo

import (
	"bluebell/models"
	"bluebell/pkg/sqls"
	"gorm.io/gorm"
)

var CommentRepository = newCommentRepository()

func newCommentRepository() *commentRepository { return &commentRepository{} }

type commentRepository struct{}

func (r *commentRepository) Create(db *gorm.DB, t *models.Comment) (err error) {
	err = db.Create(t).Error
	return
}

func (r *commentRepository) Get(db *gorm.DB, id int64) *models.Comment {
	ret := &models.Comment{}
	if err := db.First(ret, "post_id = ?", id).Error; err != nil {
		return nil
	}
	return ret
}

func (r *commentRepository) Take(db *gorm.DB, where ...interface{}) *models.Comment {
	ret := &models.Comment{}
	if err := db.Take(ret, where...).Error; err != nil {
		return nil
	}
	return ret
}

func (r *commentRepository) Find(db *gorm.DB, cnd *sqls.Cnd) (list []models.Comment) {
	cnd.Find(db, &list)
	return
}

func (r *commentRepository) Count(db *gorm.DB, cnd *sqls.Cnd) int64 {
	return cnd.Count(db, &models.Comment{})
}

func (r *commentRepository) FindPageByCnd(db *gorm.DB, cnd *sqls.Cnd) (list []models.Post, paging *sqls.Paging) {
	cnd.Find(db, &list)
	count := cnd.Count(db, &models.Comment{})

	paging = &sqls.Paging{
		Page:  cnd.Paging.Page,
		Limit: cnd.Paging.Limit,
		Total: count,
	}
	return

}
