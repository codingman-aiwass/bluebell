package mysql

import (
	"bluebell/models"
	"bluebell/modules"
	"fmt"
	"go.uber.org/zap"
)

// 和用户相关数据库操作
// CheckUserExist 用户注册时判断用户是否存在
func CheckUserExist(username string) (err error) {
	sqlStatement := "select count(*) from user where username = ?"
	var count int
	if err = db.Get(&count, sqlStatement, username); err != nil {
		return
	}
	if count > 0 {
		return ERROR_USER_EXISTS
	}
	return nil
}

func SaveUser(user *models.User) (err error) {
	sqlStatement := "insert into user(user_id,username,password) values(?,?,?)"
	_, err = db.Exec(sqlStatement, user.UserId, user.Username, user.Password)
	return
}

// CheckValidUser 用户登录时判断用户是否合法
func CheckValidUser(user *models.User) (err error) {
	oPassword := user.Password
	// 先判断用户是否存在
	sqlStatement := "select user_id,username,password from user where username = ?"
	if err = db.Get(user, sqlStatement, user.Username); err != nil {
		zap.L().Error("User not found...", zap.Error(err))
		return ERROR_USER_NOT_EXISTED
	}
	// 计算密钥
	password := modules.Encrypt(oPassword)
	if password != user.Password {
		zap.L().Error("Invalid password...", zap.Error(err))
		return ERROR_WRONG_PASSWORD
	}
	return nil
}

func GetUsernameById(userId int64) (username string, err error) {
	sqlStatement := "select username from user where user_id = ?"
	err = db.Get(&username, sqlStatement, userId)
	if err != nil {
		zap.L().Error("User not found...", zap.Error(err))
		return "", ERROR_USER_NOT_EXISTED
	}
	return username, nil

}

func GetUserEditableInfoById(userId int64) (editable_info *models.ParamUserEditInfo, err error) {
	editable_info = &models.ParamUserEditInfo{}
	sqlStatement := "select gender,email from user where user_id = ?"
	err = db.Get(editable_info, sqlStatement, userId)
	if err != nil {
		zap.L().Error("User not found...", zap.Error(err))
		return nil, ERROR_USER_NOT_EXISTED
	}
	return
}

func SaveUserEditableInfo(userId int64, editable_info *models.ParamUserEditInfo) (err error) {
	sqlStatement := "update user set gender = ?, email = ? where user_id = ?"
	_, err = db.Exec(sqlStatement, editable_info.Gender, editable_info.Email, userId)
	if err != nil {
		zap.L().Error("User not found...", zap.Error(err))
		return ERROR_USER_NOT_EXISTED
	}
	return nil
}

func GetUserByEmail(email string) (user *models.User, err error) {
	user = new(models.User)
	sqlStatement := "select user_id,username,gender,verified from user where email = ?"
	err = db.Get(user, sqlStatement, email)
	if err != nil {
		zap.L().Error("User not found...", zap.Error(err))
		return nil, ERROR_USER_NOT_EXISTED
	}
	return user, nil
}

func UpdateUserFieldByEmail(email string, field string, value interface{}) (err error) {
	sqlStatement := fmt.Sprintf("update user set %s = ? where email = ?", field)
	_, err = db.Exec(sqlStatement, value, email)
	return err
}
