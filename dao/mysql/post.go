package mysql

import (
	"bluebell/models"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
	"strings"
)

func CreatePost(post *models.Post) (err error) {
	sqlStatement := `insert into post (post_id,title,content,author_id,community_id) values(?,?,?,?,?)`
	_, err = db.Exec(sqlStatement, post.ID, post.Title, post.Content, post.AuthorID, post.CommunityID)
	if err != nil {
		zap.L().Error("create post in mysql failed", zap.Error(err))
		return err
	}
	return nil
}

func GetPostById(postId int64) (post *models.Post, err error) {
	post = new(models.Post)
	sqlStatement := `select 
    post_id,title,content,author_id,community_id,create_time 
	from post
	where post_id = ?`

	err = db.Get(post, sqlStatement, postId)
	return post, err
}

func GetPosts(pageNum, pageSize int64) (posts []*models.Post, err error) {
	posts = make([]*models.Post, 0, pageSize)
	sqlStatement := `select
	post_id,title,content,author_id,community_id,create_time 
	from post
	limit ?,?`
	err = db.Select(&posts, sqlStatement, pageNum, pageSize)
	return posts, err

}
func GetPostsByIds(postIds []string) (posts []*models.Post, err error) {
	// 通过id列表查询post表数据
	posts = make([]*models.Post, 0, len(postIds))
	sqlStatement := `select post_id,title,content,author_id,community_id,create_time
						from post
						where post_id in (?)
						order by find_in_set(post_id,?)`
	query, args, err := sqlx.In(sqlStatement, postIds, strings.Join(postIds, ","))
	query = db.Rebind(query)

	err = db.Select(&posts, query, args...)
	if err != nil {
		zap.L().Error("get basic posts by ids failed", zap.Error(err))
		return nil, err
	}

	return posts, err
}
