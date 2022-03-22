package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"

	"github.com/sirupsen/logrus"
)

// Define our message object
type Message struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Userid   string `json:"userid"`
	Message  string `json:"message"`
	Filelink string `json:"filelink"`
	Image    string `json:"image"`
}

type StringMessage struct {
	MessageType int
	Message     []byte
}

type LoginUser struct {
	UserName string
	Password string
}

type LoginUser2 struct {
	UserName      string
	Password      string
	ClientVersion string
}

type RegisterUser struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email"`
	Mobile   string `json:"mobile"`
}

type SqlConfig struct {
	SqliteFlag bool   `json:"sqliteFlag"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Database   string `json:"database"`
	UserName   string `json:"username"`
	Password   string `json:"password"`
	Charset    string `json:"charset"`
}

type SqlUser struct {
	Id         int
	UserName   string
	Mobile     string
	Password   string
	CreateTime string
	ModifyTime string
}

type OnlineSt struct {
	Userid   string
	Username string
}

type OnlineUser struct {
	Online OnlineSt
	Addr   string
}

type FileInfo struct {
	FileName   string
	FileSize   int
	UploadUser sql.NullString
	CreateTime sql.NullTime
}

type HttpResponse struct {
	Success  bool
	Msg      string
	Id       int
	Username string
}

type FilesResponse struct {
	Success bool
	Msg     string
	Files   []string
}

type FilesResponse2 struct {
	Success bool
	Msg     string
	Files   []FileInfo
}

type DeleteFile struct {
	FileName string
}

type DeleteFile2 struct {
	UserName string `json:"userName"`
	FileName string `json:"fileName"`
	UserId   int    `json:"userId"`
}

type NewClinetItem struct {
	NewClient     string
	UserId        int
	VersionNumber int
	FileName      string
	Md5Value      string
}

type ClientItem struct {
	Flag          bool
	FileName      string
	Md5Value      string
	VersionNumber int
}

type HttpResponse2 struct {
	Success   bool
	Msg       string
	Id        int
	Username  string
	NewClient ClientItem
}

type QueryFileParams struct {
	UserType int `json:"userType"`
	UserId   int `json:"userId"`
}

// 获取正在运行的函数名
func runFuncName() string {
	pc := make([]uintptr, 1)
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	return f.Name()
}

func NormalResponse(w http.ResponseWriter, result bool, msg string, id int, user string, funcName string) {
	res := HttpResponse{result, msg, id, user}
	ret, err := json.Marshal(res)
	if err != nil {
		logrus.Errorf(funcName + " json marshal failed.")
		io.WriteString(w, funcName+" json marshal failed.")
		return
	}
	io.WriteString(w, string(ret))
	return
}

func connectSql() (*sql.DB, error) {
	if g_sqlConfig.SqliteFlag {
		absDir, err := os.Getwd()
		if err != nil {
			fmt.Println("获取程序工作目录失败，错误描述：" + err.Error())
			return nil, err
		}
		db, err := sql.Open("sqlite3", absDir+"/serverDB.db")
		if err != nil {
			log.Printf("sqlite open failed:[%v]", err.Error())
			return nil, err
		}
		log.Printf("sqlite connect success.")

		return db, nil
	} else {
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s", g_sqlConfig.UserName,
			g_sqlConfig.Password,
			g_sqlConfig.Host,
			g_sqlConfig.Port,
			g_sqlConfig.Database,
			g_sqlConfig.Charset)
		log.Printf("dsn:%v", dsn)
		db, err := sql.Open("mysql", dsn)
		if err != nil {
			log.Printf("mysql connect failed, detail is [%v]", err.Error())
			return nil, err
		}
		return db, nil
	}
}

func initSql() (*sql.DB, error) {
	if g_sqlConfig.SqliteFlag {
		absDir, err := os.Getwd()
		if err != nil {
			fmt.Println("获取程序工作目录失败，错误描述：" + err.Error())
			return nil, err
		}
		db, err := sql.Open("sqlite3", absDir+"/serverDB.db")
		if err != nil {
			log.Printf("sqlite open failed:[%v]", err.Error())
			return nil, err
		}
		log.Printf("sqlite connect success.")
		tableSql := `CREATE TABLE IF NOT EXISTS "chat_user" (
			"id" INTEGER PRIMARY KEY AUTOINCREMENT,
			"user_name" varchar(255) NULL,
			"mobile" varchar(20) NULL,
			"password" varchar(255) NULL,
			"create_time" TIMESTAMP default (datetime('now', 'localtime')),
			"modify_time" NULL
		  );
		  CREATE TABLE IF NOT EXISTS "easy_chat_client" (
			"id" INTEGER PRIMARY KEY AUTOINCREMENT,
			"version" varchar(255) NULL,
			"version_number" int(4) DEFAULT NULL,
			"file_name" varchar(255) NULL,
			"file_size" bigint(12) DEFAULT NULL,
			"md5" varchar(255) NULL,
			"upload_time" TIMESTAMP default (datetime('now', 'localtime')),
			"upload_by" varchar(255) NULL
		  );
		  CREATE TABLE IF NOT EXISTS "chat_upload_files" (
			"id" INTEGER PRIMARY KEY AUTOINCREMENT,
			"file_name" varchar(255) NULL,
			"file_size" bigint(20) NULL,
			"upload_user" varchar(255) NULL,
			"create_time" TIMESTAMP default (datetime('now', 'localtime'))
		  );`
		res, err := db.Exec(tableSql)
		if err != nil {
			log.Printf("sqlite create table failed:[%v]", err.Error())
			return nil, err
		}
		id, err := res.LastInsertId()
		if err != nil {
			log.Printf("sqlite get LastInsertId failed:[%v]", err.Error())
			return nil, err
		}
		if id > 0 {
			log.Printf("sqlite create table last insert id:%d", id)
		}

		return db, nil
	} else {
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s", g_sqlConfig.UserName,
			g_sqlConfig.Password,
			g_sqlConfig.Host,
			g_sqlConfig.Port,
			g_sqlConfig.Database,
			g_sqlConfig.Charset)
		log.Printf("dsn:%v", dsn)
		db, err := sql.Open("mysql", dsn)
		if err != nil {
			log.Printf("mysql connect failed, detail is [%v]", err.Error())
			return nil, err
		}
		return db, nil
	}
}

func checkUserExist(userId string) bool {
	db, err := connectSql()
	if err != nil {
		logrus.Errorf("connect sql error:%v  func:%s", err, runFuncName())
		return false
	}
	defer db.Close()

	queryUserSql := "select user_type from chat_user where id=?;"
	stmt, err := db.Prepare(queryUserSql)
	if err != nil {
		logrus.Errorf("Prepare sql error:%v  func:%s", err, runFuncName())
		return false
	}
	rows, err := stmt.Query(userId)
	if err != nil {
		logrus.Errorf("Query sql error:%v  func:%s", err, runFuncName())
		return false
	}
	defer rows.Close()
	for rows.Next() {
		var userType sql.NullInt16
		err = rows.Scan(&userType)
		if err != nil {
			logrus.Errorf("Scan sql error:%v  func:%s", err, runFuncName())
			return false
		}
		break
	}
	return true
}
