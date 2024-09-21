DROP TABLE IF EXISTS `t_user`;
CREATE TABLE `t_user` (
    `id` bigint(64) NOT NULL AUTO_INCREMENT,
    `user_id` bigint(64) NOT NULL,
    `username` varchar(64) COLLATE utf8mb4_general_ci NOT NULL,
    `password` varchar(64) COLLATE utf8mb4_general_ci NOT NULL,
    `email` varchar(64) COLLATE utf8mb4_general_ci,
    `gender` tinyint(4) NOT NULL DEFAULT '0',
    `verified` boolean DEFAULT FALSE ,
    `status` tinyint(4) NOT NULL DEFAULT '0',
    `create_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `update_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE
        CURRENT_TIMESTAMP,
    `delete_at` TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `idx_username` (`username`) USING BTREE,
    UNIQUE KEY `idx_user_id` (`user_id`) USING BTREE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;


DROP TABLE IF EXISTS `t_community`;
CREATE TABLE `t_community` (
     `id` bigint(64) NOT NULL AUTO_INCREMENT,
     `community_id` int(64) unsigned NOT NULL,
     `community_name` varchar(128) COLLATE utf8mb4_general_ci NOT NULL,
     `introduction` varchar(256) COLLATE utf8mb4_general_ci NOT NULL,
     `create_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
     `update_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
     `delete_at` TIMESTAMP,
     PRIMARY KEY (`id`),
     UNIQUE KEY `idx_community_id` (`community_id`),
     UNIQUE KEY `idx_community_name` (`community_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;


# INSERT INTO `t_community` VALUES ('1', '1', 'Go', 'Golang', '2016-11-01 08:10:10', '2016-11-01 08:10:10');
# INSERT INTO `t_community` VALUES ('2', '2', 'leetcode', '刷题刷题刷题', '2020-01-01 08:00:00', '2020-01-01 08:00:00');
# INSERT INTO `t_community` VALUES ('3', '3', 'CS:GO', 'Rush B。。。', '2018-08-07 08:30:00', '2018-08-07 08:30:00');
# INSERT INTO `t_community` VALUES ('4', '4', 'LOL', '欢迎来到英雄联盟!', '2016-01-01 08:00:00', '2016-01-01 08:00:00');

DROP TABLE IF EXISTS `t_post`;
CREATE TABLE `t_post` (
    `id` bigint(64) NOT NULL AUTO_INCREMENT,
    `post_id` bigint(64) NOT NULL COMMENT '帖子id',
    `title` varchar(128) COLLATE utf8mb4_general_ci NOT NULL COMMENT '标题',
    `content` varchar(8192) COLLATE utf8mb4_general_ci NOT NULL COMMENT '内容',
    `author_id` bigint(64) NOT NULL COMMENT '作者的用户id',
    `community_id` bigint(64) NOT NULL COMMENT '所属社区',
    `status` tinyint(4) NOT NULL DEFAULT '1' COMMENT '帖子状态',
    `create_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    `update_time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    `delete_at` TIMESTAMP,
    UNIQUE KEY `idx_post_id` (`post_id`),
    KEY `idx_author_id` (`author_id`),
    KEY `idx_community_id` (`community_id`),
    PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

# ALTER TABLE t_post
# ADD UNIQUE KEY `idx_post_id` (`post_id`);
#
# ALTER TABLE t_user
#     ADD UNIQUE KEY `idx_user_id` (`user_id`);
# ALTER TABLE t_user
#     ADD UNIQUE KEY `idx_username` (`username`);
DROP TABLE IF EXISTS `t_comment`;
CREATE TABLE `t_comment`(
    `id` bigint(64) NOT NULL AUTO_INCREMENT,
    `comment_id` bigint(64) NOT NULL ,
    `post_id` bigint(64) NOT NULL,
    `user_id` bigint(64) NOT NULL, -- The user who made the comment
    `parent_comment_id` bigint(64) NULL, -- NULL if it's a comment on the post, or the id of the comment it replies to
    `content` varchar(8192) COLLATE utf8mb4_general_ci NOT NULL,
    `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    `update_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `delete_at` TIMESTAMP,
    UNIQUE KEY `idx_comment_id`(`comment_id`),
#     FOREIGN KEY (post_id) REFERENCES t_post(post_id),
#     FOREIGN KEY (user_id) REFERENCES t_user(user_id),
#     FOREIGN KEY (parent_comment_id) REFERENCES t_comment(comment_id) ON DELETE CASCADE,
    PRIMARY KEY (`id`)
)ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

DROP TABLE IF EXISTS `t_like`;
CREATE TABLE t_like (
    `id` bigint(64) NOT NULL AUTO_INCREMENT,
    `like_id` bigint(64) NOT NULL,
    `user_id` bigint(64) NOT NULL, -- The user who liked the post or comment
    `post_id` bigint(64) NULL, -- Post being liked (if not NULL)
    `comment_id` bigint(64) NULL, -- Comment being liked (if not NULL)
    `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    `delete_at` TIMESTAMP,
    UNIQUE KEY `idx_like_id`(`like_id`),
#     FOREIGN KEY (user_id) REFERENCES t_user(user_id),
#     FOREIGN KEY (post_id) REFERENCES t_post(post_id),
#     FOREIGN KEY (comment_id) REFERENCES t_comment(comment_id),
    CHECK (post_id IS NOT NULL OR comment_id IS NOT NULL), -- Ensure at least one is set
    PRIMARY KEY (`id`)
);

DROP TABLE IF EXISTS `t_conversation`;
CREATE TABLE t_conversation (
   `id` bigint(64) NOT NULL AUTO_INCREMENT,
   `conversation_id` bigint(64) NOT NULL ,
   `user1_id` bigint(64) NOT NULL , -- One user in the conversation
   `user2_id` bigint(64) NOT NULL , -- The other user in the conversation
   `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
   `delete_at` TIMESTAMP,
   UNIQUE KEY `idx_conversation_id`(`conversation_id`),
#    FOREIGN KEY (user1_id) REFERENCES t_user(user_id),
#    FOREIGN KEY (user2_id) REFERENCES t_user(user_id),
   UNIQUE(user1_id, user2_id), -- Ensures one conversation between two users
   PRIMARY KEY (`id`)
);

DROP TABLE IF EXISTS `t_message`;
CREATE TABLE t_message (
    `id` bigint(64) NOT NULL AUTO_INCREMENT,
    `message_id` bigint(64) NOT NULL,
    `conversation_id` bigint(64) NOT NULL, -- The conversation the message belongs to
    `sender_id` bigint(64) NOT NULL, -- The user who sent the message
    `content` varchar(8192) NOT NULL COLLATE utf8mb4_general_ci, -- The actual message content
    `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    `delete_at` TIMESTAMP,
    UNIQUE KEY `idx_message_id`(`message_id`),
#     FOREIGN KEY (conversation_id) REFERENCES t_conversation(conversation_id),
#     FOREIGN KEY (sender_id) REFERENCES t_user(user_id),
    PRIMARY KEY (`id`)
);

DROP TABLE IF EXISTS `t_follow`;
CREATE TABLE t_follow (
    `id` bigint(64) NOT NULL AUTO_INCREMENT,
    `follow_id` bigint(64) NOT NULL,
    `follower_id` bigint(64) NOT NULL, -- The user who is following
    `following_id` bigint(64) NOT NULL, -- The user being followed
    `created_at` TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    `delete_at` TIMESTAMP,
    UNIQUE KEY `idx_follow_id`(`follow_id`),
#     FOREIGN KEY (follower_id) REFERENCES t_user(id),
#     FOREIGN KEY (following_id) REFERENCES t_user(id),
    UNIQUE(follower_id, following_id), -- Prevent duplicate follows
    PRIMARY KEY (`id`)
);

