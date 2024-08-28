package logic

import (
	"bluebell/dao/mysql"
	"bluebell/models"
)

func GetAllCommunities() (communities *[]models.Community, err error) {
	return mysql.GetAllCommunities()
}

func GetCommunityById(id int64) (community *models.Community, err error) {
	return mysql.GetCommunityById(id)
}
