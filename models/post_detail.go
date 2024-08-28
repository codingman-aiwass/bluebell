package models

type PostDetail struct {
	AuthorName string `json:"author_name" db:"username"`
	YesVotes   int64  `json:"yes_votes"`
	NoVotes    int64  `json:"no_votes"`
	*Post      `json:"post"`
	*Community `json:"community"`
}
