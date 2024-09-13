package mysql_repo

import (
	"bluebell/models"
	"bluebell/pkg/sqls"
	"gorm.io/gorm"
)

var UserRepository = newUserRepository()

func newUserRepository() *userRepository {
	return &userRepository{}
}

type userRepository struct{}

// Get get single user info via id
func (r *userRepository) Get(db *gorm.DB, id int64) *models.User {
	ret := &models.User{}
	if err := db.First(ret, "user_id = ?", id).Error; err != nil {
		return nil
	}
	return ret
}

// Take get single user info via other conditions
func (r *userRepository) Take(db *gorm.DB, where ...interface{}) *models.User {
	ret := &models.User{}
	if err := db.Take(ret, where...).Error; err != nil {
		return nil
	}
	return ret
}

func (r *userRepository) Find(db *gorm.DB, cnd *sqls.Cnd) (list []models.User) {
	cnd.Find(db, &list)
	return
}

func (r *userRepository) Create(db *gorm.DB, t *models.User) (err error) {
	err = db.Create(t).Error
	return
}

func (r *userRepository) Update(db *gorm.DB, t *models.User) (err error) {
	err = db.Save(t).Error
	return
}

func (r *userRepository) Updates(db *gorm.DB, id int64, columns map[string]interface{}) (err error) {
	err = db.Model(&models.User{}).Where("user_id = ?", id).Updates(columns).Error
	return
}

func (r *userRepository) UpdateColumn(db *gorm.DB, id int64, name string, value interface{}) (err error) {
	err = db.Model(&models.User{}).Where("user_id = ?", id).UpdateColumn(name, value).Error
	return
}

func (r *userRepository) GetByUsername(db *gorm.DB, username string) *models.User {
	return r.Take(db, "username = ?", username)
}

func (r *userRepository) GetByEmail(db *gorm.DB, email string) *models.User {
	return r.Take(db, "email = ?", email)
}
