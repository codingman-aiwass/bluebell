package mysql_repo

import (
	"bluebell/models"
	"bluebell/pkg/sqls"
	"gorm.io/gorm"
)

var LikeRepository = newLikeRepository()

func newLikeRepository() *likeRepository { return &likeRepository{} }

type likeRepository struct{}

func (r *likeRepository) Create(db *gorm.DB, t *models.Like) (err error) {
	err = db.Create(t).Error
	return
}

func (r *likeRepository) Get(db *gorm.DB, id int64) *models.Like {
	ret := &models.Like{}
	if err := db.First(ret, "like_id = ?", id).Error; err != nil {
		return nil
	}
	return ret
}

func (r *likeRepository) Take(db *gorm.DB, where ...interface{}) *models.Like {
	ret := &models.Like{}
	if err := db.Take(ret, where...).Error; err != nil {
		return nil
	}
	return ret
}

func (r *likeRepository) Find(db *gorm.DB, cnd *sqls.Cnd) (list []models.Like) {
	cnd.Find(db, &list)
	return
}

func (r *likeRepository) FindOne(db *gorm.DB, cnd *sqls.Cnd) *models.Like {
	ret := &models.Like{}
	if err := cnd.FindOne(db, &ret); err != nil {
		return nil
	}
	return ret
}

func (r *likeRepository) Count(db *gorm.DB, cnd *sqls.Cnd) int64 {
	return cnd.Count(db, &models.Like{})
}

func (r *likeRepository) FindPageByCnd(db *gorm.DB, cnd *sqls.Cnd) (list []models.Like, paging *sqls.Paging) {
	cnd.Find(db, &list)
	count := cnd.Count(db, &models.Like{})

	paging = &sqls.Paging{
		Page:  cnd.Paging.Page,
		Limit: cnd.Paging.Limit,
		Total: count,
	}
	return

}

func (r *likeRepository) Delete(db *gorm.DB, id int64) {
	db.Delete(&models.Like{}, "like_id = ?", id)
}
func (r *likeRepository) UpdateColumn(db *gorm.DB, id int64, name string, value interface{}) (err error) {
	err = db.Model(&models.Like{}).Where("like_id = ?", id).UpdateColumn(name, value).Error
	return
}
