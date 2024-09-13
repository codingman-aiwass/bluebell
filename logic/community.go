package logic

import (
	"bluebell/dao/mysql_repo"
	"bluebell/models"
	"bluebell/pkg/sqls"
	"errors"
)

var (
	ERROR_COMMUNITY_NOT_EXISTS = errors.New("community not exists")
)

func GetAllCommunities() (communities []models.Community, err error) {
	communities = mysql_repo.CommunityRepository.Find(sqls.DB(), sqls.NewCnd())
	if len(communities) == 0 {
		err = ERROR_COMMUNITY_NOT_EXISTS
	}
	return communities, err
}

func GetCommunityById(id int64) (community *models.Community, err error) {
	community = mysql_repo.CommunityRepository.Get(sqls.DB(), id)
	if community == nil {
		err = ERROR_COMMUNITY_NOT_EXISTS
	}
	return community, err
}
