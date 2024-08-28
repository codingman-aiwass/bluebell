package models

type Community struct {
	Id            int64  `json:"id,string" db:"community_id"`
	CommunityName string `json:"community_name" db:"community_name"`
	Introduction  string `json:"introduction,omitempty" db:"introduction" `
}
