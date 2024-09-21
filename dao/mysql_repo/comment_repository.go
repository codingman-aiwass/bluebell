package mysql_repo

import (
	"bluebell/models"
	"bluebell/pkg/sqls"
	"errors"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var CommentRepository = newCommentRepository()

func newCommentRepository() *commentRepository { return &commentRepository{} }

type commentRepository struct{}

func (r *commentRepository) Create(db *gorm.DB, t *models.Comment) (err error) {
	err = db.Create(t).Error
	return
}

func (r *commentRepository) Get(db *gorm.DB, id int64) *models.Comment {
	ret := &models.Comment{}
	if err := db.First(ret, "comment_id = ?", id).Error; err != nil {
		return nil
	}
	return ret
}

func (r *commentRepository) Take(db *gorm.DB, where ...interface{}) *models.Comment {
	ret := &models.Comment{}
	if err := db.Take(ret, where...).Error; err != nil {
		return nil
	}
	return ret
}

func (r *commentRepository) Find(db *gorm.DB, cnd *sqls.Cnd) (list []models.Comment) {
	cnd.Find(db, &list)
	return
}

func (r *commentRepository) Count(db *gorm.DB, cnd *sqls.Cnd) int64 {
	return cnd.Count(db, &models.Comment{})
}

func (r *commentRepository) FindPageByCnd(db *gorm.DB, cnd *sqls.Cnd) (list []models.Post, paging *sqls.Paging) {
	cnd.Find(db, &list)
	count := cnd.Count(db, &models.Comment{})

	paging = &sqls.Paging{
		Page:  cnd.Paging.Page,
		Limit: cnd.Paging.Limit,
		Total: count,
	}
	return

}

func (r *commentRepository) GetRootCommentId(db *gorm.DB, parentCommentId int64) (commentId int64, err error) {
	// 根据parentCommentId,postId一路找到根comment的id
	var comment *models.Comment
	parentId := parentCommentId
	for {
		comment = r.Get(db, parentId)
		if comment.ParentCommentId == 0 {
			break
		}
		parentId = comment.ParentCommentId
	}
	if comment == nil {
		err = errors.New("not found root comment")
		return 0, err
	}
	return comment.CommentId, nil
}

// 目前的评论结构是一个多叉树，但是只有孩子指向父节点的指针，没有父节点指向孩子的指针
// 需要设计缓存之类的东西，让父节点能够指向孩子
// bluebell:comment:child_comment_record:commentId 下面存放该评论的所有子评论
// bluebell:comment:child_comment_record:commentId 在构建子评论的时候创建

func (r *commentRepository) DeleteCommentInfo(db *gorm.DB, commentIds []string) (err error) {
	// 传进来所有需要删除的comment_id
	tx := db.Begin()
	if err = tx.Error; err != nil {
		zap.L().Error("create transaction failed in DeleteCommentInfo()", zap.Error(err))
		return err
	}

	for _, id := range commentIds {
		if err = tx.Delete(&models.Comment{}, "comment_id = ?", id).Error; err != nil {
			zap.L().Error("delete comment error in DeleteCommentInfo()", zap.Error(err))
			tx.Rollback()
			return err
		}
	}
	if err = tx.Commit().Error; err != nil {
		zap.L().Error("commit transaction failed in DeleteCommentInfo()", zap.Error(err))
		tx.Rollback()
		return err
	}
	zap.L().Info("commit transaction successfully in DeleteCommentInfo()")
	return nil
}
