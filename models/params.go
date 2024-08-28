package models

type ParamUserSignUp struct {
	Username   string `json:"username" binding:"required"`
	Password   string `json:"password" binding:"required"`
	RePassword string `json:"re_password" binding:"required,eqfield=Password"`
	Email      string `json:"email"`
}

type ParamUserSignIn struct {
	Username string `json:"username" validate:"omitempty,maxoneempty"`
	Email    string `json:"email" validate:"omitempty,maxoneempty"`
	Password string `json:"password" binding:"required"`
}

type ParamUserEditInfo struct {
	Gender int8   `json:"gender"`
	Email  string `json:"email"`
}

type ParamVotePost struct {
	PostId    string `json:"post_id" binding:"required"`
	Direction int8   `json:"direction" binding:"required,oneof=0 1 -1"`
}

const (
	OrderByTime  = "time"
	OrderByScore = "score"
)

type ParamPostList struct {
	Page        int64  `form:"page"`
	Size        int64  `form:"size"`
	Order       string `form:"order"`
	CommunityId string `form:"community_id"`
}
