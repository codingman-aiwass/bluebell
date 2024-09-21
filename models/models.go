package models

import (
	"errors"
	"gorm.io/gorm"
	"time"
)

var Models = []interface{}{

	&User{}, &Community{}, &Post{}, &Comment{}, &Like{}, &Conversation{}, &Message{}, &Follow{},
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
	Username    string `json:"username" binding:"required"`
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
type ParamPostCreate struct {
	Title       string `json:"title" binding:"required"`
	Content     string `json:"content" binding:"required"`
	CommunityId int64  `json:"community_id,string" binding:"required"`
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

type ParamPostList2 struct {
	Page    int      `form:"page"`
	Size    int      `form:"size"`
	PostIds []string `form:"post_ids"`
}

type ParamCaptchaInfo struct {
	Id   string `form:"captcha-id"`
	Code string `form:"captcha-code"`
}

type ParamCreateComment struct {
	PostId          int64  `json:"post-id,string" binding:"required"`
	ParentCommentId int64  `json:"parent-comment-id,string"`
	Content         string `json:"content" binding:"required"`
}

type ParamVoteComment struct {
	CommentId int64 `json:"comment-id,string" binding:"required"`
	Direction int8  `json:"direction" binding:"required,oneof=0 1 -1"`
}

type Model struct {
	Id       int64          `gorm:"size:64;primaryKey;autoIncrement;column:id" json:"id"`
	CreateAt time.Time      `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP;column:create_at" json:"create_at"`
	DeleteAt gorm.DeletedAt `gorm:"index;column:delete_at" json:"deleted_at,omitempty"`
}

type Post struct {
	Model
	PostId      int64  `gorm:"size:64;not null;uniqueIndex:idx_post_id;column:post_id" json:"id,string"`
	AuthorID    int64  `gorm:"index:idx_author_id;size:64;not null;column:author_id" json:"author_id,string"`
	CommunityID int64  `gorm:"index:idx_community_id;column:community_id;size:64;not null" json:"community_id,string" binding:"required"`
	Status      int32  `gorm:"size:4;not null;column:status" json:"status"`
	Title       string `gorm:"size:128;not null;column:title" json:"title" binding:"required"`
	Content     string `gorm:"size:8192;not null;column:content" json:"content" binding:"required"`

	UpdateAt time.Time `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP on update CURRENT_TIMESTAMP;;column:update_at" json:"update_at"`
}

type PostDetail struct {
	Title         string    `json:"title"`
	AuthorName    string    `json:"author_name"`
	YesVotes      int64     `json:"yes_votes"`
	CommentNum    int64     `json:"comment_nums"`
	ClickNums     int64     `json:"click_nums"`
	Content       string    `json:"content"`
	UpdateAt      time.Time `json:"update_at"`
	CommunityName string    `json:"community_name,omitempty"`
}

type User struct {
	Model
	UserId   int64     `gorm:"size:64;not null;uniqueIndex:idx_user_id;column:user_id" json:"user_id,string"`
	Username string    `gorm:"size:64;not null;uniqueIndex:idx_username;column:username" json:"username"`
	Password string    `gorm:"size:64;not null;column:password"`
	Gender   int8      `gorm:"size:4;not null;default:0;column:gender" json:"gender"`
	Status   int8      `gorm:"size:4;not null;default:0;column:status" json:"status"`
	Email    string    `gorm:"size:64;column:email" json:"email"`
	Verified bool      `gorm:"default:false;column:verified"`
	UpdateAt time.Time `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP on update CURRENT_TIMESTAMP;column:update_at" json:"update_at"`
}

type Community struct {
	Model
	CommunityId   int64     `gorm:"size:64;not null;uniqueIndex:idx_community_id;column:community_id" json:"id,string"`
	CommunityName string    `gorm:"size:128;not null;uniqueIndex:idx_community_name;column:community_name" json:"community_name"`
	Introduction  string    `gorm:"size:256;not null;column:introduction" json:"introduction,omitempty"`
	UpdateAt      time.Time `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP on update CURRENT_TIMESTAMP;column:update_at" json:"update_at"`
}

type Comment struct {
	Model
	CommentId       int64     `gorm:"size:64;not null;uniqueIndex:idx_comment_id;column:comment_id" json:"id,string"`
	PostId          int64     `gorm:"size:64;not null;index;column:post_id" json:"post_id,string"`
	UserId          int64     `gorm:"size:64;not null;index;column:user_id" json:"user_id,string"`
	ParentCommentId int64     `gorm:"size:64;index;column:parent_comment_id" json:"parent_comment_id,string"`
	Content         string    `gorm:"size:8192;type:varchar(8192);not null;column:content" json:"content"`
	UpdateAt        time.Time `gorm:"type:timestamp;not null;default:CURRENT_TIMESTAMP on update CURRENT_TIMESTAMP;column:update_at" json:"update_at"`

	// Relationships
	//Post          Post      `gorm:"foreignKey:PostId;references:PostId"`
	//User          User      `gorm:"foreignKey:UserId;references:UserId"`
	//ParentComment *Comment  `gorm:"foreignKey:ParentCommentId;references:CommentId;constraint:OnDelete:CASCADE"`
	//Replies       []Comment `gorm:"foreignKey:ParentCommentId;references:CommentId"`
}

type Like struct {
	Model
	LikeId    int64 `gorm:"size:64;not null;uniqueIndex:idx_like_id;column:like_id" json:"id,string"`
	UserId    int64 `gorm:"size:64;not null;index;column:user_id" json:"user_id,string"`
	PostId    int64 `gorm:"size:64;null;index;column:post_id" json:"post_id,string"`
	CommentId int64 `gorm:"size:64;null;index;column:comment_id" json:"comment_id,string"`

	// Relationships
	//User    User     `gorm:"foreignKey:UserId;references:UserId"`
	//Post    *Post    `gorm:"foreignKey:PostId;references:PostId"`
	//Comment *Comment `gorm:"foreignKey:CommentId;references:CommentId"`
}

// Custom validation logic to enforce CHECK constraint
func (like *Like) BeforeSave(tx *gorm.DB) (err error) {
	if like.PostId == 0 && like.CommentId == 0 {
		return errors.New("either PostID or CommentID must be set")
	}
	return nil
}

type Conversation struct {
	Model
	ConversationId int64 `gorm:"size:64;not null;uniqueIndex:idx_conversation_id;column:conversation_id" json:"id,string"`
	User1Id        int64 `gorm:"size:64;not null;index:idx_user_ids,unique;column:user_1_id" json:"user-1-id,string"`
	User2Id        int64 `gorm:"size:64;not null;index:idx_user_ids,unique;column:user_2_id" json:"user-2-id,string"`

	// Relationships
	//User1 User `gorm:"foreignKey:User1Id;references:UserId"`
	//User2 User `gorm:"foreignKey:User2Id;references:UserId"`
}

type Message struct {
	Model
	MessageId      int64  `gorm:"size:64;not null;uniqueIndex:idx_message_id;column:message_id" json:"id,string"`
	ConversationId int64  `gorm:"size:64;not null;index;column:conversation_id" json:"conversation_id,string"`
	SenderId       int64  `gorm:"size:64;not null;index;column:sender_id" json:"sender_id,string"`
	Content        string `gorm:"size:8192;not null;column:content" json:"content"`

	// Relationships
	//User         User         `gorm:"foreignKey:SenderId;references:UserId"`
	//Conversation Conversation `gorm:"foreignKey:ConversationId;references:ConversationId"`
}

type Follow struct {
	Model
	FollowId    int64 `gorm:"size:64;not null;uniqueIndex:idx_follow_id;column:follow_id" json:"follow_id,string"`
	FollowerId  int64 `gorm:"size:64;not null;index:idx_follow_ids,unique;column:follower_id" json:"follower_id,string"`
	FollowingId int64 `gorm:"size:64;not null;index:idx_follow_ids,unique;column:following_id" json:"following_id,string"`

	// Relationships
	//Follower  User `gorm:"foreignKey:FollowerId;references:UserId"`
	//Following User `gorm:"foreignKey:FollowingId;references:UserId"`
}
