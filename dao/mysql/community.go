package mysql

import "bluebell/models"

func GetAllCommunities() (communities *[]models.Community, err error) {
	communities = new([]models.Community)
	sqlStatement := "select community_id,community_name from community"
	err = db.Select(communities, sqlStatement)
	if err != nil {
		return nil, err
	}
	return communities, nil
}

func GetCommunityById(id int64) (community *models.Community, err error) {
	community = new(models.Community)
	sqlStatement := "select community_id,community_name,introduction from community where community_id=?"
	err = db.Get(community, sqlStatement, id)
	if err != nil {
		return nil, err
	}
	return community, nil
}
