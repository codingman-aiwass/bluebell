package redis_repo

const (
	KeyPrefix             = "bluebell:"
	KeyPostTimeZset       = "post:time"           // zset 帖子以及发帖时间
	KeyPostScoreZset      = "post:score"          // zset 帖子以及投票分数
	KeyUserTokenHash      = "userid2access_token" // hash 记录用户id和accesstoken的映射关系
	KeyPostVotedZset      = "post:voted"          // zset 记录用户以及投票类型
	KeyCommunityPrefix    = "community:"
	KeyMailVerification   = "mail_verification"
	KeyUserLastLoginToken = "user:last_login"
	KeyMailLoginCode      = "mail_login_code"
	KeyPostVoteZset       = "post:vote"               // zset 帖子以及点赞数量
	KeyPostDevoteZset     = "post:devote"             // zset 帖子点踩数量
	KeyPostCollectionZset = "post:collection_numbers" // zset 帖子收藏数量
	KeyPostCommentZset    = "post:comment_numbers"    // zset 帖子评论数量
	KeyPostClickZset      = "post:click_numbers"      // zset 帖子浏览数量
)

func getKey(key string) string {
	return KeyPrefix + key
}
