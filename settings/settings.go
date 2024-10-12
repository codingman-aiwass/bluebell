package settings

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

type AppSettings struct {
	AppCfg   *AppConfig          `mapstructure:"app"`
	LogCfg   *LogConfig          `mapstructure:"log"`
	MysqlCfg *MysqlConfig        `mapstructure:"mysql"`
	RedisCfg *RedisConfig        `mapstructure:"redis"`
	EmailCfg *EmailConfig        `mapstructure:"email"`
	MQCfg    *MessageQueueConfig `mapstructure:"message_queue"`
}
type AppConfig struct {
	Name      string `mapstructure:"name"`
	Mode      string `mapstructure:"mode"`
	Host      string `mapstructure:"host"`
	Port      int    `mapstructure:"port"`
	StartTime string `mapstructure:"start_time"`
	MachineID int64  `mapstructure:"machine_id"`
}
type LogConfig struct {
	Level      string `mapstructure:"level"`
	Filename   string `mapstructure:"filename"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxAge     int    `mapstructure:"max_age"`
	MaxBackups int    `mapstructure:"max_backups"`
}
type MysqlConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	Database     string `mapstructure:"database"`
	MaxOpenConns int    `mapstructure:"max_open_connections"`
	MaxIdleConns int    `mapstructure:"max_idle_connections"`
}
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	Database int    `mapstructure:"database"`
	PoolSize int    `mapstructure:"pool_size"`
}
type EmailConfig struct {
	From     string `mapstructure:"email_from"`
	AuthKey  string `mapstructure:"email_auth_key"`
	SmtpHost string `mapstructure:"smtp_host"`
	SmtpPort int    `mapstructure:"smtp_port"`
}

type MessageQueueConfig struct {
	Brokers        []string `mapstructure:"brokers"`
	MaxWaitingTime int      `mapstructure:"max_waiting_time"`
	MaxBatchSize   int      `mapstructure:"max_batch_size"`
}

var GlobalSettings = new(AppSettings)

func Init() (err error) {
	viper.AddConfigPath(".") // 还可以在工作目录中查找配置
	//viper.SetConfigFile("config.yaml") // 指定配置文件路径
	viper.SetConfigName("config") // 配置文件名称(无扩展名)
	viper.SetConfigType("yaml")   // 如果配置文件的名称中没有扩展名，则需要配置此项
	err = viper.ReadInConfig()    // 查找并读取配置文件
	if err != nil {               // 处理读取配置文件的错误
		fmt.Println("viper.ReadInConfig() failed,", err)
		return
	}
	err = viper.Unmarshal(GlobalSettings)
	if err != nil {
		fmt.Println("viper.Unmarshal() failed,", err)
		return err
	}
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		fmt.Println("Config file changed:", e.Name)
		if err := viper.Unmarshal(GlobalSettings); err != nil {
			fmt.Println("viper.Unmarshal() failed,", err)
		}
	})
	return

}
