package validation

import "bluebell/models"

type Strategy interface {
	// Name 策略名称
	Name() string
	// CheckPost 检查post
	CheckPost(user *models.User, postInfo *models.Post) error
	// CheckComment 检查评论
	CheckComment(user *models.User, commentInfo *models.Comment) error
}
