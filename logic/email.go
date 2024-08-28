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
)

type EmailData struct {
	URL      string
	UserName string
	Subject  string
}

func GenEmailVerificationURL(info string) string {
	return fmt.Sprintf("%s:%d/api/v1/verify-email?info=%s",
		settings.GlobalSettings.AppCfg.Host, settings.GlobalSettings.AppCfg.Port, info)
}

func GenEmailData(email string, info string) *EmailData {
	return &EmailData{
		URL:      GenEmailVerificationURL(info),
		UserName: email,
		Subject:  "Please activate your account",
	}
}

func ParseTemplateDir(dir string) (*template.Template, error) {
	var paths []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			paths = append(paths, path)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return template.ParseFiles(paths...)
}

func SendEmail(email string, data *EmailData) error {
	m := gomail.NewMessage()
	email_config := settings.GlobalSettings.EmailCfg
	//发送人
	m.SetHeader("From", email_config.From)
	//接收人
	m.SetHeader("To", email)
	//抄送人
	//m.SetAddressHeader("Cc", "xxx@qq.com", "xiaozhujiao")

	var body bytes.Buffer

	template, err := ParseTemplateDir("templates")
	if err != nil {
		return errors.New("could not parse template")
	}

	template.ExecuteTemplate(&body, "email-verify.html", &data)
	htmlString := body.String()
	prem, _ := premailer.NewPremailerFromString(htmlString, nil)
	htmlInline, err := prem.Transform()

	//主题
	m.SetHeader("Subject", data.Subject)
	//内容
	m.SetBody("text/html", htmlInline)
	m.AddAlternative("text/plain", html2text.HTML2Text(htmlString))
	//附件
	//m.Attach("./myIpPic.png")

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
