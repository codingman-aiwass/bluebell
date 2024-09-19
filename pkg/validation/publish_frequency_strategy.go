package validation

import (
	"bluebell/dao/mysql_repo"
	"bluebell/models"
	"bluebell/pkg/dates"
	"bluebell/pkg/sqls"
	"time"
)

// 发布评论/帖子频率限制

type PublishFrequencyStrategy struct {
}

func (PublishFrequencyStrategy) Name() string {
	return "PublishFrequencyStrategy"
}

func (PublishFrequencyStrategy) CheckPost(user *models.User, post *models.Post) error {

	var (
		maxCountInTenMinutes int64 = 1 // 十分钟内最高发帖数量
		maxCountInOneHour    int64 = 2 // 一小时内最高发帖量
		maxCountInOneDay     int64 = 3 // 一天内最高发帖量
	)
	// 注册时间超过24小时，限制宽松一些
	if user.CreateAt.Unix() < dates.Timestamp(time.Now().Add(-time.Hour*24)) {
		maxCountInTenMinutes = 3
		maxCountInOneHour = 5
		maxCountInOneDay = 10
	}
	if mysql_repo.PostRepository.Count(sqls.DB(), sqls.NewCnd().Eq("user_id", user.Id).
		Gt("create_at", dates.Timestamp(time.Now().Add(-time.Hour*24)))) >= maxCountInOneDay {
		return ERROR_TOO_MANY_PUBLISH
	}

	if mysql_repo.PostRepository.Count(sqls.DB(), sqls.NewCnd().Eq("user_id", user.Id).
		Gt("create_at", dates.Timestamp(time.Now().Add(-time.Hour)))) >= maxCountInOneHour {
		return ERROR_TOO_MANY_PUBLISH
	}

	if mysql_repo.PostRepository.Count(sqls.DB(), sqls.NewCnd().Eq("user_id", user.Id).
		Gt("create_at", dates.Timestamp(time.Now().Add(-time.Minute*10)))) >= maxCountInTenMinutes {
		return ERROR_TOO_MANY_PUBLISH
	}
	return nil
}

func (PublishFrequencyStrategy) CheckComment(user *models.User, comment *models.Comment) error {
	var (
		maxCountInTenMinutes int64 = 10  // 十分钟内最高评论数量
		maxCountInOneHour    int64 = 60  // 一小时内最高评论量
		maxCountInOneDay     int64 = 100 // 一天内最高评论量
	)
	// 注册时间超过24小时，限制宽松一些
	if user.CreateAt.Unix() < dates.Timestamp(time.Now().Add(-time.Hour*24)) {
		maxCountInTenMinutes = 20
		maxCountInOneHour = 120
		maxCountInOneDay = 300
	}
	if mysql_repo.CommentRepository.Count(sqls.DB(), sqls.NewCnd().Eq("user_id", user.Id).
		Gt("create_at", dates.Timestamp(time.Now().Add(-time.Hour*24)))) >= maxCountInOneDay {
		return ERROR_TOO_MANY_PUBLISH
	}

	if mysql_repo.CommentRepository.Count(sqls.DB(), sqls.NewCnd().Eq("user_id", user.Id).
		Gt("create_at", dates.Timestamp(time.Now().Add(-time.Hour)))) >= maxCountInOneHour {
		return ERROR_TOO_MANY_PUBLISH
	}

	if mysql_repo.CommentRepository.Count(sqls.DB(), sqls.NewCnd().Eq("user_id", user.Id).
		Gt("create_at", dates.Timestamp(time.Now().Add(-time.Minute*10)))) >= maxCountInTenMinutes {
		return ERROR_TOO_MANY_PUBLISH
	}
	return nil
}
