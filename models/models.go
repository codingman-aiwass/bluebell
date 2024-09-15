package models

import "time"

var Models = []interface{}{

	&User{}, &Community{}, &Post{},
}

type ParamUserSignUp struct {
	Username    string `json:"username" binding:"required"`
	Password    string `json:"password" binding:"required"`
	RePassword  string `json:"re_password" binding:"required,eqfield=Password"`
	Email       string `json:"email"`
	CaptchaId   string `json:"captcha-id" binding:"required"`
	CaptchaCode string `json:"captcha-code" binding:"required"`
}

type ParamUserSignIn struct {
	Username    string `json:"username" validate:"omitempty,maxoneempty"`
	Password    string `json:"password" binding:"required"`
	CaptchaId   string `json:"captcha-id" binding:"required"`
	CaptchaCode string `json:"captcha-code" binding:"required"`
}
type ParamUserSignInViaEmail struct {
	Email            string `json:"email" bind:"required"`
	VerificationCode string `json:"code" binding:"required"`
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
	Page        int    `form:"page"`
	Size        int    `form:"size"`
	Order       string `form:"order"`
	CommunityId string `form:"community_id"`
}
type ParamCaptchaInfo struct {
	Id   string `form:"captcha-id"`
	Code string `form:"captcha-code"`
}

type Model struct {
	Id int64 `gorm:"size:64;primaryKey;autoIncrement" json:"id" db:"id"`
}

type Post struct {
	Model
	PostId      int64     `gorm:"size:64;not null;uniqueKey:post_id" json:"id,string" db:"post_id"`
	AuthorID    int64     `gorm:"index:idx_author_id;size:64;not null" json:"author_id,string" db:"author_id"`
	CommunityID int64     `gorm:"index:idx_community_id;size:64;not null" json:"community_id,string" db:"community_id" binding:"required"`
	Status      int32     `gorm:"size:4;not null" json:"status" db:"status"`
	Title       string    `gorm:"size:128;not null" json:"title" db:"title" binding:"required"`
	Content     string    `gorm:"size:8192;not null" json:"content" db:"content" binding:"required"`
	CreateTime  time.Time `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP" json:"createTime" db:"createTime"`
	UpdateTime  time.Time `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP on update CURRENT_TIMESTAMP" json:"updateTime" db:"updateTime"`
}

type PostDetail struct {
	AuthorName string `json:"author_name" db:"username"`
	YesVotes   int64  `json:"yes_votes"`
	NoVotes    int64  `json:"no_votes"`
	*Post      `json:"post"`
	*Community `json:"community"`
}

type User struct {
	Model
	UserId     int64     `gorm:"size:64;not null;uniqueKey:idx_user_id" db:"user_id"`
	Username   string    `gorm:"size:64;not null;uniqueKey:idx_username" db:"username"`
	Password   string    `gorm:"size:64;not null" db:"password"`
	Gender     int8      `gorm:"size:4;not null;default:0" db:"gender"`
	Email      string    `gorm:"size:64;" db:"emails"`
	Verified   bool      `gorm:"default:false" db:"verified"`
	CreateTime time.Time `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP" json:"createTime" db:"createTime"`
	UpdateTime time.Time `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP on update CURRENT_TIMESTAMP" json:"updateTime" db:"updateTime"`
}

type Community struct {
	Model
	CommunityId   int64     `gorm:"size:64;not null;uniqueIndex:idx_community_id" json:"id,string" db:"community_id"`
	CommunityName string    `gorm:"size:128;not null;uniqueIndex:idx_community_name" json:"community_name" db:"community_name"`
	Introduction  string    `gorm:"size:256;not null" json:"introduction,omitempty" db:"introduction" `
	CreateTime    time.Time `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP" json:"createTime" db:"createTime"`
	UpdateTime    time.Time `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP on update CURRENT_TIMESTAMP" json:"updateTime" db:"updateTime"`
}
