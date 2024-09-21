package validation

import (
	"bluebell/models"
	"fmt"
	"go.uber.org/zap"
)

var strategies []Strategy

func init() {
	strategies = append(strategies, &PublishFrequencyStrategy{})
}

func CheckPost(user *models.User, post *models.Post) error {
	if len(strategies) == 0 {
		return nil
	}
	for _, strategy := range strategies {
		if err := strategy.CheckPost(user, post); err != nil {
			zap.L().Error(fmt.Sprintf("[Post] hit strategy:%s", strategy.Name()), zap.Error(err))
			return err
		}
	}
	return nil
}

func CheckComment(user *models.User, comment *models.Comment) error {
	if len(strategies) == 0 {
		return nil
	}
	for _, strategy := range strategies {
		if err := strategy.CheckComment(user, comment); err != nil {
			zap.L().Error(fmt.Sprintf("[Post] hit strategy:%s", strategy.Name()), zap.Error(err))
			return err
		}
	}
	return nil
}
