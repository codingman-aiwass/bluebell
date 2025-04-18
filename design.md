**举报暂时不实现**
### post设计
1. post下会有若干评论，在用户发表post之前，需要对用户的状态做出判断，判断用户此时是否处于已经验证通过且未被禁言状态.禁言状态由管理员进行设置，也需要检查用户是否超过了一定时间内的发帖上限
2. 禁言状态可以由可以是永久禁言/临时禁言。临时禁言会额外设置一个定时任务，在到期以后自动修改用户状态
3. 用户发送post有时间限制，防止恶意用户短时间产生大量post，且需要检查post中是否存在违禁关键字(**检查违禁内容后面补充**)
4. 用户可以查看/评论/关注/点赞/点踩/转发/举报post
5. 需要根据点赞/评论/转发/观看post的数量，综合计算post的热度，展现动态热帖排行
6. 当用户被作者拉黑，其评论将被自动删除，且以后无法在该帖子下发言

#### post接口
##### GET
1. 按分页获取帖子简略信息 （默认一页10条）(显示标题/作者名/点赞数/评论数/50个字正文) √
2. 获取更简略的帖子信息（CSDN评论区下的帖子推荐）(显示标题/作者名/20个字正文/浏览量/发布时间) √
3. 按照ID获取帖子详细信息(标题/正文/作者名/点赞数/评论数/浏览量/发布时间/所属社区) (下面所属评论由前端自行请求评论接口获取) √
4. 返回热度最高的前10条帖子（按照2的形式返回）
5. 获取帖子的转发链接 √
6. 每隔一段时间自动执行帖子热度计算算法
##### POST
1. 为帖子点赞/点踩/举报 √
2. 收藏帖子 √
3. 发布帖子 √
4. 根据ID删除帖子 √

需要在mysql的帖子数据表中添加帖子的浏览数，评论数，点赞数，点踩数，以及分数（热贴）

### comment设计
1. comment可以是一个用户对post的评论，也可以是对评论的追评。在用户发表评论之前，也需要对用户的状态做出判断，判断是否超出一定时间内的发贴上限
2. 发帖者可以删除评论
3. 用户可以评论/点赞/点踩/举报comment
4. 需要计算评论下有多少追评
5. 需要保存当前帖子点赞/评论/数量

##### GET
1. 根据帖子ID，分页获取帖子的评论以及追评 √
2. 根据帖子ID，查看该帖子下的评论总数 √
3. 根据评论ID，获取该评论的追评数量 √
4. 根据评论ID，查看该评论的详细信息 √
##### POST
1. 根据帖子ID，进行评论 √
2. 根据帖子ID以及评论ID，进行追评 √
3. 点赞/点踩/举报评论 √
4. 根据评论ID删除评论（当评论被删除时，其下的子评论也会被删除）√

需要在mysql的评论数据表中添加对该评论的点赞数，评论数（跟评论需要，子评论不需要），点踩数，根评论ID（方便增加根评论的评论数，方便在删除根评论的时候，删除其下所有的子评论），分数（热评）
需要重写评论树实现的方法，去除在Redis中手动维护一棵评论树的方法，去除删除一个子评论时删除其下所有追评的逻辑，只有在删除根评论时才删除其下所有子评论
可以准备一下，放弃手动维护评论树的原因
需要添加一个表，记录用户对帖子和评论的点赞情况，t_like表示用户收藏了帖子
帖子的点赞数量的获取，首先通过查询redis缓存post:vote_up/down，缓存中没有的话，去t_vote表中，通过计数符合下面条件：val=1，target id = post id，type = 1的记录，并写入redis post:vote_up/down
用户修改点赞情况时，除了修改MySQL，redis中的user_action，还需要修改post:vote_up,post:vote_down。并设置一个定时任务，定期检查缓存中的点赞数量是否和MySQL数据库一致

### message设计
1. 用户可以向其他用户发送消息，也可以是系统向用户推送消息
2. 当有人评论/点赞帖子时，向帖子发布者发送消息提醒
3. 当有人追评/点赞评论时，向父评论发送消息提醒
4. 当用户拉黑其他用户后，被拉黑用户无法发送消息


### 数据一致性
为尽可能保证缓存和数据库数据的一致性，采用先更新数据库，然后通过引入消息队列，将删除缓存的操作加入到消息队列，让消费者执行一定尝试次数的删除缓存操作。

「先更新数据库，再删除缓存」的方案虽然保证了数据库与缓存的数据一致性，但是每次更新数据的时候，缓存的数据都会被删除，这样会对缓存的命中率带来影响。
因此可以采用先更新数据库，再更新缓存的方法。但是为防止多线程并行执行导致的数据库/缓存数据不一致问题，需要加上分布式锁/缓存过期时间，保证数据库/缓存一致性

限流策略的设计：使用`juju/ratelimit`，使用一个sync.Map存储每个用户对应的令牌桶，保证并发安全。Key为user_id+IP，或者email+IP，限制每分钟只能获取一个令牌

```go

import (
    "fmt"
    "net/http"
    "sync"
    "time"

    "github.com/juju/ratelimit"
)

var userBuckets = sync.Map{}

func handleRequest(w http.ResponseWriter, r *http.Request) {
    user := r.RemoteAddr

    userBucket, ok := userBuckets.LoadOrStore(user, ratelimit.NewBucketWithRate(1, time.Minute))

    if userBucket.(*ratelimit.Bucket).Take(1) == ratelimit.Wait {
        http.Error(w, "Too many requests", http.StatusTooManyRequests)
        return
    }

    fmt.Fprintf(w, "Request handled at %s\n", time.Now())
}

```

### 本地cache和redis的选择
#### 适用于`goburrow/cache`的场景
1. Frequently Accessed, Non-Critical Data
   - User Profile Information:  Cache user profile data like usernames, avatars, or roles. These details are frequently accessed but rarely updated, making them ideal for local caching.
   - Post Metadata:Metadata such as simple post summaries (e.g., title, creation date). Since this data is frequently read but infrequently updated, caching it locally can improve the performance of post listings or previews.
2. Computed Data or Aggregations
   -  Cache query results like paginated results of posts or comments, where the data may change slightly but doesn't affect user experience critically.
      Example: Caching the list of posts for a category or the current page of comments on a post can improve browsing speed without needing global consistency.

3. Data That Changes Frequently (Temporary Caching):
   - Rate Limit:Track the number of actions (e.g., posting comments or sending messages) a user has performed within a short time window to enforce rate limiting.
     Example: Caching rate limits in local memory is quick, and since these limits are time-sensitive and node-specific, consistency across nodes isn't critical.
4. Session-Specific or Short-Lived Data:
   - Draft Posts or Comments: If users write drafts of posts or comments that are only stored temporarily (before being committed to the database), these can be cached locally.
     Example: Caching a user's current post draft in memory while they are typing, then persisting it to Redis or the database when they save.
5. Computation-Heavy but Short-Lived Data:(not to do now.)
   - Temporary Computation Results: Cache the results of computationally expensive tasks like calculating the number of unread notifications for a user, which can be regenerated if lost.
     Example: When a user logs in, cache the count of unread messages or notifications for quick display while they are active on the same node.

#### 适用于`Redis`的场景
1. Globally Consistent and Shared Data:
   - Hot/Trending Posts:
      - Since trending posts need to be consistent across all nodes to provide the same experience to all users, storing them in Redis ensures that all nodes access the same set of data.
      - Example: Cache a list of top 10 trending posts in Redis, updated periodically, and serve it to all users across different nodes.
   - Global Post Metadata:
      - Data like the total number of comments, upvotes, or post views should be shared across nodes. Storing this in Redis ensures that all nodes are synchronized.
      - Example: Cache the view counts of popular posts or the total upvotes on a post in Redis for consistency across all servers.
2. User Session Data:
   - User Authentication Sessions:
      - Storing user session tokens or login states in Redis ensures that users can be authenticated consistently regardless of which node they interact with.
     - Example: Cache user login sessions, tokens, and roles in Redis so users remain logged in when they switch nodes.
3. User Activity Feeds (if shared across instances):
   - User's Recent Posts or Comments:
      - If a user’s activity feed (e.g., posts they’ve commented on or liked) needs to be consistent across different nodes, storing it in Redis makes it accessible to all nodes.
      - Example: Cache the list of a user's most recent posts or comments in Redis to ensure consistency across multiple sessions or devices.
4. Global Caches for Expensive Queries or Results:
   - Search Results for Popular Queries:
      - Cache the results of frequently performed searches, such as "most upvoted posts" or "most commented threads," in Redis to provide consistent results across all nodes.
      - Example: Store results for popular forum search queries, such as the top threads by category, in Redis to avoid repeated expensive queries.
5. Notification Data:
   - User Notifications:
      - Notifications such as replies to posts, likes, or private messages should be stored in Redis so that users can access them no matter which node they are served by.
      - Example: Cache unread notifications or messages in Redis so users see the same notifications across different devices or sessions.
6. Global Rate Limiting:
   - Rate Limiting for Shared Resources:
      - For features that need to be rate-limited across all users and instances (e.g., API usage or forum-wide actions), Redis can store and track these globally.
      - Example: If a certain API endpoint should limit requests to 100 per minute across all users, storing the rate limit counters in Redis ensures consistent enforcement across all nodes.


#### **When to Use Both (Hybrid Approach)**:
In some cases, you can use both `goburrow/cache` and Redis in a complementary fashion:
- **Cache Hot Data in Redis, Cold Data in Local Cache**: Store the most frequently accessed (hot) data globally in Redis, and use `goburrow/cache` for less frequently accessed (cold) data on each node.
- **Use Local Cache for Speed, Redis for Consistency**: Cache small, frequently used items locally in `goburrow/cache` for performance, but store critical or shared data in Redis to ensure consistency across nodes.

#### Summary of Data for Each Cache:
| **Data Type**               | **goburrow/cache**                         | **Redis**                               |
|-----------------------------|--------------------------------------------|-----------------------------------------|
| User-Specific, Non-Critical  | Last visited post, UI preferences          | Global session data, authentication     |
| Frequently Accessed Metadata | Local post metadata, comment count         | Global post metadata, trending posts    |
| Computed Data                | Page rendering data, cached search results | Expensive queries (e.g., top contributors) |
| Temporary Data               | Drafts, form data, rate-limiting (local)   | Global rate-limiting, notifications     |
| Global Consistent Data       | Not suitable                               | Hot posts, recently accessed data       |
| Distributed Locking          | Not suitable                               | Task locks, concurrency control         |
| Leaderboard/Reputation       | Not suitable                               | Reputation scores, global leaderboards  |

In a **distributed forum project**, `goburrow/cache` is great for node-local, ephemeral data that improves performance without requiring cross-node consistency, while Redis is essential for globally shared, critical data that needs to remain consistent across all nodes.