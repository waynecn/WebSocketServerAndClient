package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"mime"
	_ "mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"

	"database/sql"
	"time"

	_ "github.com/go-sql-driver/mysql"
	rotatelogs "github.com/lestrrat/go-file-rotatelogs"
	"github.com/rifflock/lfshook"
	"github.com/sirupsen/logrus"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

var clients = make(map[*websocket.Conn]bool) // connected clients
var broadcast = make(chan StringMessage)     // broadcast channel
var onlineusers []OnlineUser

// Configure the upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

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
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
	UserName string `json:"username"`
	Password string `json:"password"`
	Charset  string `json:"charset"`
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
	UserName string
	FileName string
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

func ConfigLocalFileSystemLogger(logPath string, logFileName string, maxAge time.Duration, rotationTime time.Duration) {
	baseLogPath := path.Join(logPath, logFileName)
	fmt.Println("baseLogPath:", baseLogPath)
	writer, err := rotatelogs.New(baseLogPath+".%Y%m%d%H%M",
		rotatelogs.WithLinkName(baseLogPath),
		rotatelogs.WithMaxAge(maxAge),
		rotatelogs.WithRotationTime(rotationTime))
	if err != nil {
		fmt.Println("config local file system logger error, detail err:", err)
		logrus.Errorf("config local file system logger error. %+v", errors.WithStack(err))
	}

	lfHook := lfshook.NewHook(lfshook.WriterMap{
		logrus.DebugLevel: writer,
		logrus.InfoLevel:  writer,
		logrus.WarnLevel:  writer,
		logrus.ErrorLevel: writer,
		logrus.FatalLevel: writer,
		logrus.PanicLevel: writer}, &logrus.TextFormatter{})

	logrus.AddHook(lfHook)
}

var g_sqlConfig SqlConfig
var g_Db *sql.DB
var g_strWorkDir string

func ReadConfig(path string) SqlConfig {
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		log.Panicln("load config conf failed: ", err)
	}
	err = json.Unmarshal(buf, &g_sqlConfig)
	if err != nil {
		log.Panicln("decode config file failed:", string(buf), err)
	}

	return g_sqlConfig
}

func connectSql() (*sql.DB, error) {
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

func main() {
	absDir, err := os.Getwd()
	if err != nil {
		fmt.Println("获取程序工作目录失败，错误描述：" + err.Error())
		return
	}
	logDir := absDir + "/log"
	err = os.MkdirAll(logDir, 0777)
	if err != nil {
		fmt.Println("创建目录:", logDir, "失败")
		return
	}
	ConfigLocalFileSystemLogger("./log", "WebSocketServer.log", 30*24*time.Hour, 24*time.Hour)

	logrus.Infof("Use -p xxxx to use the given port xxxx")
	port := "5133"
	if len(os.Args) >= 3 {
		port = os.Args[2]
	}
	g_strWorkDir = getCurrentDirectory()
	//Read config
	configPath := "./config/config.json"
	g_sqlConfig = ReadConfig(configPath)
	log.Printf("sqlConfig:%v", g_sqlConfig)
	g_Db, err = connectSql()
	if err != nil {
		logrus.Errorf("Connect sql failed.")
		return
	}
	defer g_Db.Close()

	// Create a simple file server
	fs := http.FileServer(http.Dir("./public"))
	http.Handle("/", fs)

	//http request response
	http.HandleFunc("/uploads", uploadFunction)
	http.HandleFunc("/uploads2", uploadFunction2)
	http.HandleFunc("/loginnew", loginFunction2)
	http.HandleFunc("/login", loginFunction2)
	http.HandleFunc("/register", registerFunction)
	http.HandleFunc("/uploadfiles", queryUploadFilesFunction)
	http.HandleFunc("/uploadfiles2", queryUploadFilesFunction2)
	http.HandleFunc("/delfile", deleteFile)
	http.HandleFunc("/delfile2", deleteFile2)
	http.HandleFunc("/uploadClient", uploadClient)

	// Configure websocket route
	http.HandleFunc("/ws", handleConnections)

	// Start listening for incoming chat messages
	go handleMessages()

	// Start the server on localhost port 8000 and log any errors
	logrus.Infof("http server started on :%s", port)
	err = http.ListenAndServe(":"+port, nil)
	if err != nil {
		logrus.Errorf("ListenAndServe: %s", err)
	}
}

func getCurrentDirectory() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	return strings.Replace(dir, "\\", "/", -1)
}

func checkErr(err error, msg string, w http.ResponseWriter) bool {
	if err != nil {
		response := HttpResponse{false, msg + err.Error(), -1, ""}
		bts, err := json.Marshal(response)
		if err != nil {
			logrus.Infof("json marshal failed.")
			io.WriteString(w, "json marshal failed.")
			return false
		}
		logrus.Infof(string(bts))
		io.WriteString(w, string(bts))
		return false
	}
	return true
}

func uploadFunction(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		logrus.Infof("invalid request")
		io.WriteString(w, "invalid request")
		return
	}

	contentType := r.Header.Get("Content-Type")
	contentLen := r.ContentLength
	logrus.Infof("contentType: %s, contentLen:%s", contentType, contentLen)
	mediatype, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		logrus.Errorf("ParseMediaType error: %s", err)
		w.Write([]byte("ParseMediaType error"))
		return
	}
	curDir := getCurrentDirectory()
	logrus.Infof("curDir:%s", curDir)
	dir := curDir + "/public/uploads"
	logrus.Infof("mediatype:%s", mediatype)
	if mediatype == "multipart/form-data" {
		logrus.Infof("in multipart parsing...")
		logrus.Infof("r.MultipartForm:%s", r.MultipartForm)
		if r.MultipartForm != nil {
			for name, files := range r.MultipartForm.File {
				logrus.Infof("req.MultipartForm.File,name=:%s files:%s", name, len(files))
				if len(files) != 1 {
					w.Write([]byte("too many files"))
					return
				}
				if name == "" {
					w.Write([]byte("is not FileData"))
					return
				}
				for _, f := range files {
					handle, err := f.Open()
					if err != nil {
						w.Write([]byte(fmt.Sprintf("unknown error,fileName=%s,fileSize=%d,err:%s", f.Filename, f.Size, err.Error())))
						return
					}

					path := dir + f.Filename
					dst, _ := os.Create(path)
					io.Copy(dst, handle)
					dst.Close()
					logrus.Infof("successful uploaded,fileName=%s,fileSize=%.2f MB,savePath=%s \n", f.Filename, float64(contentLen)/1024/1024, path)

					w.Write([]byte("successful,url=" + url.QueryEscape(f.Filename)))
				}
			}
		} else {
			reader, err := r.MultipartReader()
			if err != nil {
				panic(err)
			}
			for {
				p, err := reader.NextPart()
				if err == io.EOF {
					logrus.Infof("EOF break")
					break
				}

				if err != nil {
					logrus.Infof("reader.NextPart error:%s", err)
					break
				}

				fileName := p.FileName()
				logrus.Infof("fileName:%s", fileName)
				if fileName != "" {
					_, err = os.Stat(dir)
					if err != nil && os.IsNotExist(err) {
						//目录不存在
						err = os.MkdirAll(dir, 0777)
						if err != nil {
							logrus.Infof("create dir failed:%s", err)
							continue
						}
					}
					fo, err := os.Create(dir + "/" + fileName)
					if err != nil {
						logrus.Infof("os create file err:%s", err)
						continue
					}
					defer fo.Close()
					defer recordToSql(fileName)
					formValue, _ := ioutil.ReadAll(p)
					fo.Write(formValue)
				}
			}
		}
	}

	logrus.Infof("***********************************")

	var bts = []byte("success")
	w.Write(bts)
}

func uploadFunction2(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		logrus.Errorf("invalid request")
		io.WriteString(w, "invalid request")
		return
	}

	contentType := r.Header.Get("Content-Type")
	contentLen := r.ContentLength
	userName := r.Header.Get("UserName")
	logrus.Infof("contentType:%s contentLen:%s user:%s", contentType, contentLen, userName)
	mediatype, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		logrus.Errorf("ParseMediaType error:%s", err)
		w.Write([]byte("ParseMediaType error"))
		return
	}
	curDir := getCurrentDirectory()
	logrus.Infof("curDir:%s", curDir)
	dir := curDir + "/public/uploads"
	logrus.Infof("mediatype: ", mediatype)
	if mediatype == "multipart/form-data" {
		logrus.Infof("in multipart parsing...")
		logrus.Infof("r.MultipartForm:%s", r.MultipartForm)
		if r.MultipartForm != nil {
			for name, files := range r.MultipartForm.File {
				logrus.Errorf("req.MultipartForm.File,name=:%s files:%s", name, len(files))
				if len(files) != 1 {
					w.Write([]byte("too many files"))
					return
				}
				if name == "" {
					w.Write([]byte("is not FileData"))
					return
				}
				for _, f := range files {
					handle, err := f.Open()
					if err != nil {
						w.Write([]byte(fmt.Sprintf("unknown error,fileName=%s,fileSize=%d,err:%s", f.Filename, f.Size, err.Error())))
						return
					}

					path := dir + f.Filename
					dst, _ := os.Create(path)
					io.Copy(dst, handle)
					dst.Close()
					logrus.Infof("successful uploaded,fileName=%s,fileSize=%.2f MB,savePath=%s", f.Filename, float64(contentLen)/1024/1024, path)

					w.Write([]byte("successful,url=" + url.QueryEscape(f.Filename)))
				}
			}
		} else {
			reader, err := r.MultipartReader()
			if err != nil {
				panic(err)
			}
			for {
				p, err := reader.NextPart()
				if err == io.EOF {
					logrus.Infof("EOF break")
					break
				}

				if err != nil {
					logrus.Infof("reader.NextPart error:%s", err)
					break
				}

				fileName := p.FileName()
				logrus.Infof("fileName:%s", fileName)
				if fileName != "" {
					_, err = os.Stat(dir)
					if err != nil && os.IsNotExist(err) {
						//目录不存在
						err = os.MkdirAll(dir, 0777)
						if err != nil {
							logrus.Infof("create dir failed:%s", err)
							continue
						}
					}
					fo, err := os.Create(dir + "/" + fileName)
					if err != nil {
						logrus.Infof("os create file err: ", err)
						continue
					}
					defer fo.Close()
					defer recordToSql2(fileName, userName, contentLen)
					formValue, _ := ioutil.ReadAll(p)
					fo.Write(formValue)
				}
			}
		}
	}

	logrus.Infof("***********************************")

	var bts = []byte("success")
	w.Write(bts)
}

func loginFunction(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	msg := "ReadAll body failed."
	if !checkErr(err, msg, w) {
		return
	}

	var userinfo LoginUser
	err = json.Unmarshal(body, &userinfo)
	msg = "json unmarshal failed."
	if !checkErr(err, msg, w) {
		return
	}

	//Check username and password from mysql
	strSql := "select id, user_name, password from chat_user where mobile=?"
	stmt, err := g_Db.Prepare(strSql)
	msg = "Prepare sql failed."
	if !checkErr(err, msg, w) {
		return
	}

	rows, err := stmt.Query(userinfo.UserName)
	msg = "Query sql failed."
	if !checkErr(err, msg, w) {
		return
	}
	defer rows.Close()

	count := 0
	var id int
	var user_name string
	var password string
	for rows.Next() {
		err := rows.Scan(&id, &user_name, &password)
		msg = "Failed to get sql item."
		if !checkErr(err, msg, w) {
			return
		}

		if userinfo.Password != password {
			response := HttpResponse{false, "Password is not correct.", -1, ""}
			bts, err := json.Marshal(response)
			if err != nil {
				logrus.Errorf("json marshal failed.")
				io.WriteString(w, "json marshal failed.")
				return
			}
			logrus.Infof(string(bts))
			io.WriteString(w, string(bts))
			return
		}
		count = count + 1
	}
	if count < 1 {
		response := HttpResponse{false, "User does not exist.", -1, ""}
		bts, err := json.Marshal(response)
		if err != nil {
			logrus.Errorf("json marshal failed.")
			io.WriteString(w, "json marshal failed.")
			return
		}
		logrus.Infof(string(bts))
		io.WriteString(w, string(bts))
		return
	}
	response := HttpResponse{true, "success", id, user_name}
	bts, err := json.Marshal(response)
	if err != nil {
		logrus.Errorf("json marshal failed.")
		io.WriteString(w, "json marshal failed.")
		return
	}
	io.WriteString(w, string(bts))
}

func loginFunction2(w http.ResponseWriter, r *http.Request) {
	logrus.Infof("interface:loginFunction2")
	body, err := ioutil.ReadAll(r.Body)
	msg := "ReadAll body failed."
	if !checkErr(err, msg, w) {
		return
	}

	var userinfo LoginUser2
	err = json.Unmarshal(body, &userinfo)
	msg = "json unmarshal failed."
	if !checkErr(err, msg, w) {
		return
	}
	logrus.Infof("userInfo2:%v", userinfo)

	//Check username and password from mysql
	strSql := "select id, user_name, password from chat_user where mobile=?"
	stmt, err := g_Db.Prepare(strSql)
	msg = "Prepare sql failed."
	if !checkErr(err, msg, w) {
		return
	}

	rows, err := stmt.Query(userinfo.UserName)
	msg = "Query sql failed."
	if !checkErr(err, msg, w) {
		return
	}
	defer rows.Close()

	count := 0
	var userId int
	var user_name string
	var password string
	for rows.Next() {
		err := rows.Scan(&userId, &user_name, &password)
		msg = "Failed to get sql item."
		if !checkErr(err, msg, w) {
			return
		}

		if userinfo.Password != password {
			response := HttpResponse{false, "Password is not correct.", -1, ""}
			bts, err := json.Marshal(response)
			if err != nil {
				logrus.Errorf("json marshal failed.")
				io.WriteString(w, "json marshal failed.")
				return
			}
			logrus.Infof(string(bts))
			io.WriteString(w, string(bts))
			return
		}
		count = count + 1
	}
	if count < 1 {
		response := HttpResponse{false, "User does not exist.", -1, ""}
		bts, err := json.Marshal(response)
		if err != nil {
			logrus.Errorf("json marshal failed.")
			io.WriteString(w, "json marshal failed.")
			return
		}
		logrus.Infof(string(bts))
		io.WriteString(w, string(bts))
		return
	}

	count = 0
	fileName := ""
	md5Str := ""
	maxVersionNumber := 0
	finalFileName := ""
	finalMd5Str := ""
	newClientFlag := false
	if userinfo.ClientVersion == "" {
		return
	} else {
		versionNumber := getVersionNumber(userinfo.ClientVersion)
		strSql2 := "select file_name, md5, version_number from easy_chat_client where version_number > ?"
		stmt2, err2 := g_Db.Prepare(strSql2)
		if err2 != nil {
			logrus.Errorf("loginFunction2 prepare sql error:%v", err2)
		} else {
			rows2, err2 := stmt2.Query(versionNumber)
			if err2 != nil {
				logrus.Errorf("loginFunction2 Query by versionNumber:%i error:%v", versionNumber, err2)
			} else {
				defer rows2.Close()
				for rows2.Next() {
					verNumber := 0
					err := rows2.Scan(&fileName, &md5Str, &verNumber)
					msg := "Failed to get sql item."
					if err != nil {
						logrus.Error(msg)
						continue
					}

					if maxVersionNumber < verNumber {
						maxVersionNumber = verNumber
						finalFileName = fileName
						finalMd5Str = md5Str
					}

					count = count + 1
				}
				if count < 1 {
					logrus.Infof("there is no new client")
					newClientFlag = false
				} else {
					newClientFlag = true
				}
			}
		}
	}

	clientItem := ClientItem{newClientFlag, finalFileName, finalMd5Str, maxVersionNumber}
	response := HttpResponse2{true, "success", userId, user_name, clientItem}
	bts, err := json.Marshal(response)
	if err != nil {
		logrus.Errorf("json marshal failed.")
		io.WriteString(w, "json marshal failed.")
		return
	}
	io.WriteString(w, string(bts))
}

func registerFunction(w http.ResponseWriter, r *http.Request) {
	token := r.Header["Token"][0]
	logrus.Infof("token:%s", token)
	if token != "20200101" {
		res := HttpResponse{false, "Token verify failed.", -1, ""}
		_, err := json.Marshal(res)
		if err != nil {
			logrus.Errorf("json marshal failed.")
			io.WriteString(w, "json marshal failed.")
			return
		}
	}
	body, err := ioutil.ReadAll(r.Body)
	msg := "ReadAll body failed."
	if !checkErr(err, msg, w) {
		return
	}

	var regUser RegisterUser
	err = json.Unmarshal(body, &regUser)
	msg = "json unmarshal body failed."
	if !checkErr(err, msg, w) {
		return
	}

	//check the new user exists or not
	strSql := "select id from chat_user where mobile=?;"
	stmt, err := g_Db.Prepare(strSql)
	msg = "Prepare sql failed 1."
	if !checkErr(err, msg, w) {
		return
	}

	rows, err := stmt.Query(regUser.Mobile)
	msg = "Query sql failed."
	if !checkErr(err, msg, w) {
		return
	}
	defer rows.Close()

	var id int
	for rows.Next() {
		err := rows.Scan(&id)
		msg = "Failed to get sql item."
		if !checkErr(err, msg, w) {
			return
		}
		if id > 0 {
			response := HttpResponse{false, "User already exists.", -1, ""}
			bts, err := json.Marshal(response)
			if err != nil {
				logrus.Errorf("json marshal failed.")
				io.WriteString(w, "json marshal failed.")
				return
			}
			logrus.Infof(string(bts))
			io.WriteString(w, string(bts))
			return
		}
	}

	strSql = "insert chat_user (user_name,mobile,password) values(?,?,?);"
	stmt, err = g_Db.Prepare(strSql)
	msg = "Prepare sql failed 2."
	if !checkErr(err, msg, w) {
		return
	}

	res, err := stmt.Exec(regUser.Username, regUser.Mobile, regUser.Password)
	msg = "Insert into sql failed."
	if !checkErr(err, msg, w) {
		return
	}

	newId, err := res.LastInsertId()
	if !checkErr(err, "Get LastInsertId failed.", w) {
		return
	}
	logrus.Infof("new Insert id:%s", newId)

	response := HttpResponse{true, "success", id, regUser.Username}
	bts, err := json.Marshal(response)
	if err != nil {
		logrus.Errorf("json marshal failed.")
		io.WriteString(w, "json marshal failed.")
		return
	}
	io.WriteString(w, string(bts))
}

func broadCastOnline() {
	for client := range clients {
		contents, err := json.Marshal(onlineusers)
		if err != nil {
			logrus.Errorf("json Marshal onlineusers failed.")
			continue
		}

		var msg StringMessage
		msg.MessageType = 1
		msg.Message = contents
		err = client.WriteMessage(msg.MessageType, msg.Message)
		if err != nil {
			logrus.Errorf("Broadcast online message error: %v", err)
			client.Close()
			delete(clients, client)
		}
	}
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	// Upgrade initial GET request to a websocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	// Make sure we close the connection when the function returns
	defer ws.Close()

	// Register our new client
	clients[ws] = true
	logrus.Infof("ws:%s  Network:%s String:%s", ws.RemoteAddr(), ws.RemoteAddr().Network(), ws.RemoteAddr().String())
	//Whenever a new client was connected, send the online message to all clients
	broadCastOnline()

	for {
		//var msg Message
		// Read in a new message as JSON and map it to a Message object
		messageType, p, err := ws.ReadMessage()
		if err != nil {
			logrus.Errorf("ReadMessage error:%v", err)
			addr := ws.RemoteAddr().String()
			for index, item := range onlineusers {
				if item.Addr == addr {
					onlineusers = append(onlineusers[:index], onlineusers[index+1:]...)
				}
			}
			delete(clients, ws)
			logrus.Infof("Current online user count:%d", len(onlineusers))
			broadCastOnline()
			break
		}
		// Send the newly received message to the broadcast channel
		var msg StringMessage
		msg.MessageType = messageType
		msg.Message = p
		broadcast <- msg

		//check if there is some online infos, then parse the online info and save them
		var onlinestr string
		onlinestr = string(p[:])
		if strings.Index(onlinestr, "online") != -1 {
			var onlineuser OnlineUser
			err = json.Unmarshal([]byte(onlinestr), &onlineuser)
			if err != nil {
				logrus.Errorf("json unmarshal online info failed.")
				continue
			}
			bFind := false
			for _, item := range onlineusers {
				if item.Online.Userid == onlineuser.Online.Userid {
					bFind = true
					break
				}
			}

			if !bFind {
				addr := ws.RemoteAddr().String()
				onlineuser.Addr = addr
				onlineusers = append(onlineusers, onlineuser)
			}
		}
		logrus.Infof("Current online user count:%d", len(onlineusers))
	}
}

func handleMessages() {
	for {
		// Grab the next message from the broadcast channel
		msg := <-broadcast
		// Send it out to every client that is currently connected
		for client := range clients {
			err := client.WriteMessage(msg.MessageType, msg.Message)
			//logrus.Infof(msg.Message)
			if err != nil {
				logrus.Errorf("WriteMessage error:%v", err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}

func recordToSql(fileName string) bool {
	strSql := "insert chat_upload_files (file_name) values (?);"
	stmt, err := g_Db.Prepare(strSql)
	msg := "Prepare sql failed in recordToSql"
	if err != nil {
		logrus.Errorf(msg)
		return false
	}

	res, err := stmt.Exec(fileName)
	if err != nil {
		logrus.Errorf("insert into sql failed in recordToSql")
		return false
	}

	newId, err := res.LastInsertId()
	if err != nil {
		logrus.Errorf("Get last inserted id failed in recordToSql")
		return false
	}
	logrus.Infof("new Inserted id:%s", newId)
	return true
}

func recordToSql2(fileName string, userName string, fileSize int64) bool {
	strSql := "insert chat_upload_files (file_name, upload_user, file_size) values (?, ?, ?);"
	stmt, err := g_Db.Prepare(strSql)
	msg := "Prepare sql failed in recordToSql"
	if err != nil {
		logrus.Errorf(msg)
		return false
	}

	res, err := stmt.Exec(fileName, userName, fileSize)
	if err != nil {
		logrus.Errorf("insert into sql failed in recordToSql")
		return false
	}

	newId, err := res.LastInsertId()
	if err != nil {
		logrus.Errorf("Get last inserted id failed in recordToSql")
		return false
	}
	logrus.Infof("new Inserted id:%s", newId)
	return true
}

func recordClient(version string, versionNum int, fileName string, fileSize int64, md5 string, userName string) bool {
	strSql := "insert easy_chat_client (version, version_number, file_name, file_size, md5, upload_time, upload_by) values (?, ?, ?, ?, ?, CURRENT_TIMESTAMP, ?);"
	stmt, err := g_Db.Prepare(strSql)
	msg := "recordClient Prepare sql failed in recordToSql"
	if err != nil {
		logrus.Errorf(msg)
		return false
	}

	res, err := stmt.Exec(version, versionNum, fileName, fileSize, md5, userName)
	if err != nil {
		logrus.Errorf("recordClient insert into sql failed in recordToSql")
		return false
	}

	newId, err := res.LastInsertId()
	if err != nil {
		logrus.Errorf("recordClient Get last inserted id failed in recordToSql")
		return false
	}
	logrus.Infof("recordClient new Inserted id:%d", newId)
	return true
}

func queryUploadFilesFunction(w http.ResponseWriter, r *http.Request) {
	tokens := r.Header["Token"]
	if tokens == nil {
		res := HttpResponse{false, "need token", -1, ""}
		ret, err := json.Marshal(res)
		if err != nil {
			logrus.Errorf("json marshal failed.")
			io.WriteString(w, "json marshal failed.")
			return
		}
		io.WriteString(w, string(ret))
		return
	}
	token := tokens[0]
	logrus.Infof("token:%s", token)
	if token != "20200101" {
		res := HttpResponse{false, "Token verify failed.", -1, ""}
		_, err := json.Marshal(res)
		if err != nil {
			logrus.Errorf("json marshal failed.")
			io.WriteString(w, "json marshal failed.")
			return
		}
	}

	//check the new user exists or not
	strSql := "select file_name from chat_upload_files order by create_time desc;"
	stmt, err := g_Db.Prepare(strSql)
	msg := "Prepare sql failed 1."
	if !checkErr(err, msg, w) {
		return
	}

	rows, err := stmt.Query()
	msg = "Query sql failed."
	if !checkErr(err, msg, w) {
		return
	}
	defer rows.Close()

	var fileName string
	var files []string
	for rows.Next() {
		err := rows.Scan(&fileName)
		msg = "Failed to get sql item."
		if !checkErr(err, msg, w) {
			return
		}
		files = append(files, fileName)
	}

	response := FilesResponse{true, "success", files}
	bts, err := json.Marshal(response)
	if err != nil {
		logrus.Errorf("json marshal failed.")
		io.WriteString(w, "json marshal failed.")
		return
	}
	io.WriteString(w, string(bts))
}

func queryUploadFilesFunction2(w http.ResponseWriter, r *http.Request) {
	tokens := r.Header["Token"]
	if tokens == nil {
		res := HttpResponse{false, "need token", -1, ""}
		ret, err := json.Marshal(res)
		if err != nil {
			logrus.Errorf("json marshal failed.")
			io.WriteString(w, "json marshal failed.")
			return
		}
		io.WriteString(w, string(ret))
		return
	}
	token := tokens[0]
	logrus.Infof("token:%s", token)
	if token != "20200101" {
		res := HttpResponse{false, "Token verify failed.", -1, ""}
		_, err := json.Marshal(res)
		if err != nil {
			logrus.Errorf("json marshal failed.")
			io.WriteString(w, "json marshal failed.")
			return
		}
	}

	//check the new user exists or not
	strSql := "select file_name,file_size,upload_user from chat_upload_files order by create_time desc;"
	stmt, err := g_Db.Prepare(strSql)
	msg := "Prepare sql failed 1."
	if !checkErr(err, msg, w) {
		return
	}

	rows, err := stmt.Query()
	msg = "Query sql failed."
	if !checkErr(err, msg, w) {
		return
	}
	defer rows.Close()

	var fileName string
	var fileSize int
	var uploadUser sql.NullString
	var files []FileInfo
	for rows.Next() {
		err := rows.Scan(&fileName, &fileSize, &uploadUser)
		msg = "Failed to get sql item."
		if !checkErr(err, msg, w) {
			return
		}
		var fileInfo FileInfo
		fileInfo.FileName = fileName
		fileInfo.FileSize = fileSize
		fileInfo.UploadUser = uploadUser
		files = append(files, fileInfo)
	}

	response := FilesResponse2{true, "success", files}
	bts, err := json.Marshal(response)
	if err != nil {
		logrus.Errorf("json marshal failed.")
		io.WriteString(w, "json marshal failed.")
		return
	}
	io.WriteString(w, string(bts))
}

func deleteFile(w http.ResponseWriter, r *http.Request) {
	tokens := r.Header["Token"]
	if tokens == nil {
		res := HttpResponse{false, "need token", -1, ""}
		ret, err := json.Marshal(res)
		if err != nil {
			logrus.Errorf("json marshal failed.")
			io.WriteString(w, "json marshal failed.")
			return
		}
		io.WriteString(w, string(ret))
		return
	}
	token := tokens[0]
	if token != "20200101" {
		res := HttpResponse{false, "Token verify failed.", -1, ""}
		_, err := json.Marshal(res)
		if err != nil {
			logrus.Errorf("json marshal failed.")
			io.WriteString(w, "json marshal failed.")
			return
		}
	}

	utf8Reader := transform.NewReader(r.Body, simplifiedchinese.GBK.NewDecoder())
	body, err := ioutil.ReadAll(utf8Reader)
	msg := "delete file ReadAll body failed."
	if !checkErr(err, msg, w) {
		return
	}

	var deleteFile DeleteFile
	err = json.Unmarshal(body, &deleteFile)
	msg = "delete file json unmarshal failed."
	if !checkErr(err, msg, w) {
		return
	}

	//Check username and password from mysql
	strSql := "select id, file_name from chat_upload_files where file_name=?"
	stmt, err := g_Db.Prepare(strSql)
	msg = "Prepare sql failed."
	if !checkErr(err, msg, w) {
		return
	}

	rows, err := stmt.Query(deleteFile.FileName)
	msg = "Query sql failed."
	if !checkErr(err, msg, w) {
		return
	}
	defer rows.Close()

	var id int
	var file_name string
	for rows.Next() {
		err := rows.Scan(&id, &file_name)
		logrus.Infof("fileName:%s", file_name)
		msg = "Failed to get sql item."
		if !checkErr(err, msg, w) {
			return
		}

		delSql := "delete from chat_upload_files where file_name='" + deleteFile.FileName + "'"
		g_Db.Query(delSql)
	}

	response := HttpResponse{true, "success", id, file_name}
	bts, err := json.Marshal(response)
	if err != nil {
		logrus.Errorf("json marshal failed.")
		io.WriteString(w, "json marshal failed.")
		return
	}

	f := g_strWorkDir + "/public/uploads/" + deleteFile.FileName
	err = os.Remove(f)
	if err != nil {
		msg = "remove file from disk failed.{}"
		logrus.Errorf(msg, err)
	}
	io.WriteString(w, string(bts))
}

func deleteFile2(w http.ResponseWriter, r *http.Request) {
	tokens := r.Header["Token"]
	if tokens == nil {
		res := HttpResponse{false, "need token", -1, ""}
		ret, err := json.Marshal(res)
		if err != nil {
			logrus.Errorf("json marshal failed.")
			io.WriteString(w, "json marshal failed.")
			return
		}
		io.WriteString(w, string(ret))
		return
	}
	token := tokens[0]
	if token != "20200101" {
		res := HttpResponse{false, "Token verify failed.", -1, ""}
		_, err := json.Marshal(res)
		if err != nil {
			logrus.Errorf("json marshal failed.")
			io.WriteString(w, "json marshal failed.")
			return
		}
	}

	utf8Reader := transform.NewReader(r.Body, simplifiedchinese.GBK.NewDecoder())
	body, err := ioutil.ReadAll(utf8Reader)
	msg := "delete file ReadAll body failed."
	if !checkErr(err, msg, w) {
		return
	}

	var deleteFile DeleteFile2
	err = json.Unmarshal(body, &deleteFile)
	msg = "delete file json unmarshal failed."
	if !checkErr(err, msg, w) {
		return
	}

	logrus.Infof("delete file:%s by user:%s", deleteFile.FileName, deleteFile.UserName)

	//Check username and password from mysql
	strSql := "select id, file_name from chat_upload_files where file_name=?"
	stmt, err := g_Db.Prepare(strSql)
	msg = "Prepare sql failed."
	if !checkErr(err, msg, w) {
		return
	}

	rows, err := stmt.Query(deleteFile.FileName)
	msg = "Query sql failed."
	if !checkErr(err, msg, w) {
		return
	}
	defer rows.Close()

	var id int
	var file_name string
	for rows.Next() {
		err := rows.Scan(&id, &file_name)
		logrus.Infof("fileName:%s", file_name)
		msg = "Failed to get sql item."
		if !checkErr(err, msg, w) {
			return
		}

		delSql := "delete from chat_upload_files where file_name='" + deleteFile.FileName + "'"
		g_Db.Query(delSql)
	}

	response := HttpResponse{true, "success", id, file_name}
	bts, err := json.Marshal(response)
	if err != nil {
		logrus.Errorf("json marshal failed.")
		io.WriteString(w, "json marshal failed.")
		return
	}

	f := g_strWorkDir + "/public/uploads/" + deleteFile.FileName
	err = os.Remove(f)
	if err != nil {
		msg = "remove file from disk failed.%v"
		logrus.Errorf(msg, err)
	}
	io.WriteString(w, string(bts))
}

func uploadClient(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		logrus.Errorf("invalid request")
		io.WriteString(w, "invalid request")
		return
	}

	contentType := r.Header.Get("Content-Type")
	contentLen := r.ContentLength
	logrus.Infof("uploadClient contentType:%s contentLen:%s", contentType, contentLen)
	mediatype, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		logrus.Errorf("ParseMediaType error:%s", err)
		w.Write([]byte("ParseMediaType error"))
		return
	}
	curDir := getCurrentDirectory()
	logrus.Infof("curDir:%s", curDir)
	dir := curDir + "/public/clients"
	logrus.Infof("mediatype: %s", mediatype)
	if mediatype == "multipart/form-data" {
		logrus.Infof("in multipart parsing...")
		logrus.Infof("r.MultipartForm:%v", r.MultipartForm)
		if r.MultipartForm != nil {
			//表单名称
			for name, files := range r.MultipartForm.File {
				logrus.Errorf("req.MultipartForm.File,name=:%s files:%s", name, len(files))
				if len(files) != 1 {
					w.Write([]byte("too many files"))
					return
				}
				if name == "" {
					w.Write([]byte("is not FileData"))
					return
				}
				for _, f := range files {
					handle, err := f.Open()
					if err != nil {
						w.Write([]byte(fmt.Sprintf("unknown error,fileName=%s,fileSize=%d,err:%s", f.Filename, f.Size, err.Error())))
						return
					}

					path := dir + f.Filename
					dst, _ := os.Create(path)
					io.Copy(dst, handle)
					dst.Close()
					logrus.Infof("successful uploaded,fileName=%s,fileSize=%.2f MB,savePath=%s", f.Filename, float64(contentLen)/1024/1024, path)

					w.Write([]byte("successful,url=" + url.QueryEscape(f.Filename)))
				}
			}
		} else {
			version := ""
			versionNumber := -1
			userName := "unknown"
			md5Value := "unknown"
			reader, err := r.MultipartReader()
			if err != nil {
				panic(err)
			}
			form, err := reader.ReadForm(128)
			for key, val := range form.Value {
				logrus.Infof("form key: %s, value: %s", key, val)
				if key == "versionStr" {
					if len(val) > 0 {
						version = val[0]
					}
				}
				if key == "userName" {
					if len(val) > 0 {
						userName = val[0]
					}
				}
				if key == "versionNumber" {
					if len(val) > 0 {
						versionNumber, err = strconv.Atoi(val[0])
						if err != nil {
							logrus.Errorf("versionNumber strconv Atoi error:%v", err)
						}
					}
				}
				if key == "MD5" {
					if len(val) > 0 {
						md5Value = val[0]
					}
				}
			}
			for _, v := range form.File {
				for i := 0; i < len(v); i++ {
					fileName := v[i].Filename
					logrus.Infof("file part:%d", i)
					logrus.Infof("fileName:%s", v[i].Filename)
					logrus.Infof("part-header:%v", v[i].Header)

					_, err = os.Stat(dir)
					if err != nil && os.IsNotExist(err) {
						//目录不存在
						err = os.MkdirAll(dir, 0777)
						if err != nil {
							logrus.Infof("create dir failed:%s", err)
							continue
						}
					}
					fo, err := os.Create(dir + "/" + fileName)
					if err != nil {
						logrus.Infof("os create file err:%v", err)
						continue
					}
					defer fo.Close()
					defer recordClient(version, versionNumber, fileName, contentLen, md5Value, userName)
					f, _ := v[i].Open()
					formValue, _ := ioutil.ReadAll(f)
					fo.Write(formValue)
				}
			}
		}
	}

	logrus.Infof("***********************************")

	var bts = []byte("success")
	w.Write(bts)
}

func getVersionNumber(version string) int {
	versions := strings.Split(version, ".")
	length := len(versions)
	versionNumber := 0
	for i := 0; i < length; i++ {
		num, err := strconv.Atoi(versions[i])
		if err != nil {
			logrus.Errorf("getVersionNumber strconv Atoi error:%v", err)
			return 0
		}
		versionNumber += num * int(math.Pow10(length-1-i))
	}

	logrus.Errorf("getVersionNumber end...:%d", versionNumber)
	return versionNumber
}
