package mysql_repo

import (
	"bluebell/models"
	"bluebell/pkg/sqls"
	"gorm.io/gorm"
)

var VoteRepository = newVoteRepository()

func newVoteRepository() *voteRepository { return &voteRepository{} }

type voteRepository struct{}

func (r *voteRepository) Create(db *gorm.DB, t *models.Vote) (err error) {
	err = db.Create(t).Error
	return
}

func (r *voteRepository) Get(db *gorm.DB, id int64) *models.Vote {
	ret := &models.Vote{}
	if err := db.First(ret, "vote_id = ?", id).Error; err != nil {
		return nil
	}
	return ret
}

func (r *voteRepository) Take(db *gorm.DB, where ...interface{}) *models.Vote {
	ret := &models.Vote{}
	if err := db.Take(ret, where...).Error; err != nil {
		return nil
	}
	return ret
}

func (r *voteRepository) Find(db *gorm.DB, cnd *sqls.Cnd) (list []models.Vote) {
	cnd.Find(db, &list)
	return
}

func (r *voteRepository) FindOne(db *gorm.DB, cnd *sqls.Cnd) *models.Vote {
	ret := &models.Vote{}
	if err := cnd.FindOne(db, &ret); err != nil {
		return nil
	}
	return ret
}
func (r *voteRepository) Update(db *gorm.DB, t *models.Vote) (err error) {
	err = db.Save(t).Error
	return
}
func (r *voteRepository) UpdateColumn(db *gorm.DB, id int64, name string, value interface{}) (err error) {
	err = db.Model(&models.Vote{}).Where("vote_id = ?", id).UpdateColumn(name, value).Error
	return
}
func (r *voteRepository) Count(db *gorm.DB, cnd *sqls.Cnd) int64 {
	return cnd.Count(db, &models.Vote{})
}

func (r *voteRepository) FindPageByCnd(db *gorm.DB, cnd *sqls.Cnd) (list []models.Vote, paging *sqls.Paging) {
	cnd.Find(db, &list)
	count := cnd.Count(db, &models.Vote{})

	paging = &sqls.Paging{
		Page:  cnd.Paging.Page,
		Limit: cnd.Paging.Limit,
		Total: count,
	}
	return

}

func (r *voteRepository) Delete(db *gorm.DB, id int64) {
	db.Delete(&models.Like{}, "vote_id = ?", id)
}
