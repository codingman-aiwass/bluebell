package mysql_repo

import (
	"bluebell/models"
	"bluebell/pkg/sqls"
	"gorm.io/gorm"
)

var CommunityRepository = newCommunityRepository()

func newCommunityRepository() *communityRepository {
	return &communityRepository{}
}

type communityRepository struct{}

func (r *communityRepository) Get(db *gorm.DB, id int64) *models.Community {
	ret := &models.Community{}
	if err := db.First(&ret, "community_id = ?", id).Error; err != nil {
		return nil
	}
	return ret
}

func (r *communityRepository) Find(db *gorm.DB, cnd *sqls.Cnd) (list []models.Community) {
	cnd.Find(db, &list)
	return
}
