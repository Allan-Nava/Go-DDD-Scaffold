package database


import (
	"github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func InitDB() *gorm.DB {
	//
	//config := config.CONFIGURATION
	/*dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", config.dbUsername, config.dbPassword, config.dbHost, config.dbPort, config.dbName )
	conn, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		logrus.Fatalf("error connection on %s, err: %s", dsn, err.Error())
	}
	db, _ := conn.DB()
	db.SetMaxIdleConns(idle)
	db.SetMaxOpenConns(max)*/

	// github.com/mattn/go-sqlite3
	dsn := sqlite.Open("gorm.db")
	conn, err := gorm.Open(dsn, &gorm.Config{})
	if err != nil {
		logrus.Fatalf("error connection on %s, err: %s", dsn, err.Error())
	}
	db, _ := conn.DB()
	db.SetMaxIdleConns(10)
	//
	return conn
}