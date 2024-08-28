package redis

const (
	KeyPrefix          = "bluebell:"
	KeyPostTimeZset    = "post:time"           // zset 帖子以及发帖时间
	KeyPostScoreZset   = "post:score"          // zset 帖子以及投票分数
	KeyUserTokenHash   = "userid2access_token" // hash 记录用户id和accesstoken的映射关系
	KeyPostVotedZset   = "post:voted"          // zset 记录用户以及投票类型
	KeyCommunityPrefix = "community:"
)

func getKey(key string) string {
	return KeyPrefix + key
}
