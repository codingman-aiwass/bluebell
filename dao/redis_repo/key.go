package redis_repo

const (
	KeyPrefix                   = "bluebell:"
	KeyPostTimeZset             = "post:create_time"      // zset 帖子以及发帖时间
	KeyPostScoreZset            = "post:score"            // zset 帖子以及投票分数
	KeyUserTokenHash            = "userid2access_token"   // hash 记录用户id和accesstoken的映射关系
	KeyPostActionPrefix         = "post:user_action:"     // 记录用户的对帖子的投票类型,后面跟post id，整体是一个hset，key为user id，值为none, like, dislike
	KeyPostUserCollection       = "post:user_collection:" // 记录用户收藏的所有帖子，后面跟user id,整体是一个Set
	KeyCommunityPrefix          = "community:"
	KeyMailVerification         = "mail_verification"
	KeyUserLastLoginToken       = "user:last_login"
	KeyMailLoginCode            = "mail_login_code"
	KeyPostVoteUpZset           = "post:vote_up"                 // zset 帖子以及点赞数量
	KeyPostVoteDownZset         = "post:vote_down"               // zset 帖子点踩数量
	KeyPostCollectionZset       = "post:collection_numbers"      // zset 帖子收藏数量
	KeyPostCommentZset          = "post:comment_numbers"         // zset 帖子评论数量，统计这一帖子下面一共有多少评论。
	KeyPostClickZset            = "post:click_numbers"           // zset 帖子浏览数量
	KeyUserBlackListSet         = "user:blacklist"               // set 用户黑名单
	KeyCommentTimeZset          = "comment:time"                 // zset 评论以及评论时间
	KeyCommentScoreZset         = "comment:score"                // zset 评论以及分数（计算热评）
	KeyPostPrefix               = "post:"                        // 后面可以拼接postId，表内存放根评论
	KeyCommentVotedZset         = "comment:voted"                // zset 记录用户以及投票类型
	KeyCommentVoteZset          = "comment:vote"                 // zset comment以及点赞数量
	KeyCommentDevoteZset        = "comment:devote"               // zset comment以及点踩数量
	KeyCommentSubCommentCntZset = "comment:comment_numbers"      // zset comment 的子评论总数,存放所有根评论的评论总数。统计每个根comment下共有多少追评
	KeyCommentSubCommentSet     = "comment:child_comment_record" // set,存放当前评论的所有子评论
)

func getKey(key string) string {
	return KeyPrefix + key
}
