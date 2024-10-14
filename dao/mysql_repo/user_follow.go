package mysql_repo

import (
	"bluebell/models"
	"bluebell/pkg/sqls"
	"gorm.io/gorm"
)

var UserFollowRepository = newUserFollowRepository()

func newUserFollowRepository() *userFollowRepository { return &userFollowRepository{} }

type userFollowRepository struct{}

func (r *userFollowRepository) Create(db *gorm.DB, t *models.Follow) (err error) {
	err = db.Create(t).Error
	return
}

func (r *userFollowRepository) Get(db *gorm.DB, id int64) *models.Follow {
	ret := &models.Follow{}
	if err := db.First(ret, "follow_id = ?", id).Error; err != nil {
		return nil
	}
	return ret
}

func (r *userFollowRepository) Take(db *gorm.DB, where ...interface{}) *models.Follow {
	ret := &models.Follow{}
	if err := db.Take(ret, where...).Error; err != nil {
		return nil
	}
	return ret
}

func (r *userFollowRepository) Find(db *gorm.DB, cnd *sqls.Cnd) (list []models.Follow) {
	cnd.Find(db, &list)
	return
}

func (r *userFollowRepository) FindOne(db *gorm.DB, cnd *sqls.Cnd) *models.Follow {
	ret := &models.Follow{}
	if err := cnd.FindOne(db, &ret); err != nil {
		return nil
	}
	return ret
}

func (r *userFollowRepository) UpdateColumn(db *gorm.DB, id int64, name string, value interface{}) (err error) {
	err = db.Model(&models.Follow{}).Where("follow_id = ?", id).UpdateColumn(name, value).Error
	return
}

func (r *userFollowRepository) Creates(db *gorm.DB, ts []*models.Follow) error {
	err := db.Create(&ts).Error
	return err
}
