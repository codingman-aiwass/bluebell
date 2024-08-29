package logic

import (
	"bluebell/settings"
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/k3a/html2text"
	"github.com/vanng822/go-premailer/premailer"
	"go.uber.org/zap"
	"gopkg.in/gomail.v2"
	"html/template"
	"os"
	"path/filepath"
	"strings"
)

type EmailVerificationData struct {
	URL              string
	Username         string
	VerificationCode string
	Subject          string
}

func GenEmailVerificationURL(info string) string {
	return fmt.Sprintf("%s:%d/api/v1/verify-email?info=%s",
		settings.GlobalSettings.AppCfg.Host, settings.GlobalSettings.AppCfg.Port, info)
}

func GenEmailVerificationData(email string, info string) *EmailVerificationData {
	return &EmailVerificationData{
		URL:      GenEmailVerificationURL(info),
		Username: email,
		Subject:  "Please activate your account",
	}
}

func GenEmailVerificationCodeData(email string, code string) *EmailVerificationData {
	return &EmailVerificationData{
		Username:         email,
		VerificationCode: code,
		Subject:          "Bluebell Email Verification Code",
	}
}

func ParseTemplateDir(dir string) (*template.Template, error) {
	t := template.New("") // 创建一个新的模板上下文
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".html") {
			_, err := t.ParseFiles(path)
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return t, nil
}

// 验证邮箱的邮件，包含激活链接
func SendVerificationInfoEmail(email string, data *EmailVerificationData) error {
	return SendEmail(email, "email-verify", true, data)

}

// 发送验证码的邮件，包含本次验证码
func SendVerificationCodeEmail(email string, code string) error {
	return SendEmail(email, "email-code", false, GenEmailVerificationCodeData(email, code))
}

func SendEmail(email, templateFileName string, alternative bool, data *EmailVerificationData) error {
	email_config := settings.GlobalSettings.EmailCfg
	m := gomail.NewMessage()
	//发送人
	m.SetHeader("From", email_config.From)
	//接收人
	m.SetHeader("To", email)

	var body bytes.Buffer

	tmpl, err := ParseTemplateDir("templates")
	if err != nil {
		return errors.New("could not parse template")
	}

	// 使用 base 模板并嵌入特定的内容模板
	err = tmpl.ExecuteTemplate(&body, templateFileName, data)
	if err != nil {
		return errors.New("could not execute base template")
	}

	htmlString := body.String()
	prem, _ := premailer.NewPremailerFromString(htmlString, nil)
	htmlInline, err := prem.Transform()

	//主题
	m.SetHeader("Subject", data.Subject)
	//内容
	m.SetBody("text/html", htmlInline)
	if alternative {
		m.AddAlternative("text/plain", html2text.HTML2Text(htmlString))
	}

	//拿到token，并进行连接,第4个参数是填授权码
	d := gomail.NewDialer(email_config.SmtpHost, email_config.SmtpPort, email_config.From, email_config.AuthKey)
	d.TLSConfig = &tls.Config{InsecureSkipVerify: true}

	// 发送邮件
	if err := d.DialAndSend(m); err != nil {
		zap.L().Error("could not send email", zap.Error(err))
		return err
	}
	return nil
}
