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
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"

	"time"

	_ "github.com/go-sql-driver/mysql"
	rotatelogs "github.com/lestrrat/go-file-rotatelogs"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rifflock/lfshook"
	"github.com/sirupsen/logrus"

	"flag"

	"github.com/robfig/cron"
)

var clients = make(map[*websocket.Conn]bool) // connected clients
var broadcast = make(chan StringMessage)     // broadcast channel
var onlineusers []OnlineUser
var g_Mutex sync.Mutex

// Configure the upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
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

//var g_Db *sql.DB
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

var port = flag.String("p", "5133", "服务端口")

func main() {
	flag.Parse()

	initSql()
	initLotterySqlite()

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

	TCPPort, err := strconv.Atoi(*port)
	if err != nil {
		logrus.Errorf("port to int failed.")
		*port = "5133"
		TCPPort, _ = strconv.Atoi(*port)
	}
	g_strWorkDir = getCurrentDirectory()
	//Read config
	configPath := "./config/config.json"
	g_sqlConfig = ReadConfig(configPath)
	log.Printf("sqlConfig:%v", g_sqlConfig)

	// Create a simple file server
	fs := http.FileServer(http.Dir("./public"))
	http.Handle("/", fs)

	//http request response
	http.HandleFunc("/uploads", uploadFunction)
	http.HandleFunc("/uploads2", uploadFunction2)
	http.HandleFunc("/uploads3", uploadFunction3)
	http.HandleFunc("/loginnew", loginFunction2)
	http.HandleFunc("/login", loginFunction)
	http.HandleFunc("/register", registerFunction)
	http.HandleFunc("/uploadfiles", queryUploadFilesFunction)
	http.HandleFunc("/uploadfiles2", queryUploadFilesFunction2)
	http.HandleFunc("/uploadfiles3", queryUploadFilesFunction3)
	http.HandleFunc("/delfile", deleteFile)
	http.HandleFunc("/delfile2", deleteFile2)
	http.HandleFunc("/delfile3", deleteFile3)
	http.HandleFunc("/uploadClient", uploadClient)
	http.HandleFunc("/lottery", lotteryFunc)
	http.HandleFunc("/lotteryHistory", lotteryHistoryFunc)
	http.HandleFunc("/queryKjgg", queryKjggImpl)

	// Configure websocket route
	http.HandleFunc("/ws", handleConnections)

	// Start listening for incoming chat messages
	go handleMessages()

	// start tcp server
	TCPPort = TCPPort + 1
	go handleTCPConnections(TCPPort)

	//准备启动定时器 定时查询开奖公告以及历史开奖公告
	c := cron.New()
	c.AddFunc("0 40 21 * * ?", queryKjgg) //每天21点31分
	c.Start()
	defer c.Stop()

	//启动服务时先查询一次
	go queryKjgg()

	// Start the server on localhost port 8000 and log any errors
	logrus.Infof("http server started on :%s", *port)
	err = http.ListenAndServe(":"+*port, nil)
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
	res := HttpResponse{false, "obselete interface", -1, ""}
	ret, err := json.Marshal(res)
	if err != nil {
		logrus.Errorf("json marshal failed.")
		io.WriteString(w, "json marshal failed.")
		return
	}
	io.WriteString(w, string(ret))

	// if r.Method != "POST" {
	// 	logrus.Infof("invalid request")
	// 	io.WriteString(w, "invalid request")
	// 	return
	// }

	// contentType := r.Header.Get("Content-Type")
	// contentLen := r.ContentLength
	// logrus.Infof("contentType: %s, contentLen:%s", contentType, contentLen)
	// mediatype, _, err := mime.ParseMediaType(contentType)
	// if err != nil {
	// 	logrus.Errorf("ParseMediaType error: %s", err)
	// 	w.Write([]byte("ParseMediaType error"))
	// 	return
	// }
	// curDir := getCurrentDirectory()
	// logrus.Infof("curDir:%s", curDir)
	// dir := curDir + "/public/uploads"
	// logrus.Infof("mediatype:%s", mediatype)
	// if mediatype == "multipart/form-data" {
	// 	logrus.Infof("in multipart parsing...")
	// 	logrus.Infof("r.MultipartForm:%s", r.MultipartForm)
	// 	if r.MultipartForm != nil {
	// 		for name, files := range r.MultipartForm.File {
	// 			logrus.Infof("req.MultipartForm.File,name=:%s files:%s", name, len(files))
	// 			if len(files) != 1 {
	// 				w.Write([]byte("too many files"))
	// 				return
	// 			}
	// 			if name == "" {
	// 				w.Write([]byte("is not FileData"))
	// 				return
	// 			}
	// 			for _, f := range files {
	// 				handle, err := f.Open()
	// 				if err != nil {
	// 					w.Write([]byte(fmt.Sprintf("unknown error,fileName=%s,fileSize=%d,err:%s", f.Filename, f.Size, err.Error())))
	// 					return
	// 				}

	// 				path := dir + f.Filename
	// 				dst, _ := os.Create(path)
	// 				io.Copy(dst, handle)
	// 				dst.Close()
	// 				logrus.Infof("successful uploaded,fileName=%s,fileSize=%.2f MB,savePath=%s \n", f.Filename, float64(contentLen)/1024/1024, path)

	// 				w.Write([]byte("successful,url=" + url.QueryEscape(f.Filename)))
	// 			}
	// 		}
	// 	} else {
	// 		reader, err := r.MultipartReader()
	// 		if err != nil {
	// 			panic(err)
	// 		}
	// 		for {
	// 			p, err := reader.NextPart()
	// 			if err == io.EOF {
	// 				logrus.Infof("EOF break")
	// 				break
	// 			}

	// 			if err != nil {
	// 				logrus.Infof("reader.NextPart error:%s", err)
	// 				break
	// 			}

	// 			fileName := p.FileName()
	// 			logrus.Infof("fileName:%s", fileName)
	// 			if fileName != "" {
	// 				_, err = os.Stat(dir)
	// 				if err != nil && os.IsNotExist(err) {
	// 					//目录不存在
	// 					err = os.MkdirAll(dir, 0777)
	// 					if err != nil {
	// 						logrus.Infof("create dir failed:%s", err)
	// 						continue
	// 					}
	// 				}
	// 				fo, err := os.Create(dir + "/" + fileName)
	// 				if err != nil {
	// 					logrus.Infof("os create file err:%s", err)
	// 					continue
	// 				}
	// 				defer fo.Close()
	// 				defer recordToSql(fileName)
	// 				formValue, _ := ioutil.ReadAll(p)
	// 				fo.Write(formValue)
	// 			}
	// 		}
	// 	}
	// }

	// logrus.Infof("***********************************")

	// var bts = []byte("success")
	// w.Write(bts)
}

func uploadFunction2(w http.ResponseWriter, r *http.Request) {
	res := HttpResponse{false, "obselete interface", -1, ""}
	ret, err := json.Marshal(res)
	if err != nil {
		logrus.Errorf("json marshal failed.")
		io.WriteString(w, "json marshal failed.")
		return
	}
	io.WriteString(w, string(ret))

	// if r.Method != "POST" {
	// 	logrus.Errorf("invalid request")
	// 	io.WriteString(w, "invalid request")
	// 	return
	// }

	// contentType := r.Header.Get("Content-Type")
	// contentLen := r.ContentLength
	// userName := r.Header.Get("UserName")
	// logrus.Infof("contentType:%s contentLen:%s user:%s", contentType, contentLen, userName)
	// mediatype, _, err := mime.ParseMediaType(contentType)
	// if err != nil {
	// 	logrus.Errorf("ParseMediaType error:%s", err)
	// 	w.Write([]byte("ParseMediaType error"))
	// 	return
	// }
	// curDir := getCurrentDirectory()
	// logrus.Infof("curDir:%s", curDir)
	// dir := curDir + "/public/uploads"
	// logrus.Infof("mediatype: ", mediatype)
	// if mediatype == "multipart/form-data" {
	// 	logrus.Infof("in multipart parsing...")
	// 	logrus.Infof("r.MultipartForm:%s", r.MultipartForm)
	// 	if r.MultipartForm != nil {
	// 		for name, files := range r.MultipartForm.File {
	// 			logrus.Errorf("req.MultipartForm.File,name=:%s files:%s", name, len(files))
	// 			if len(files) != 1 {
	// 				w.Write([]byte("too many files"))
	// 				return
	// 			}
	// 			if name == "" {
	// 				w.Write([]byte("is not FileData"))
	// 				return
	// 			}
	// 			for _, f := range files {
	// 				handle, err := f.Open()
	// 				if err != nil {
	// 					w.Write([]byte(fmt.Sprintf("unknown error,fileName=%s,fileSize=%d,err:%s", f.Filename, f.Size, err.Error())))
	// 					return
	// 				}

	// 				path := dir + f.Filename
	// 				dst, _ := os.Create(path)
	// 				io.Copy(dst, handle)
	// 				dst.Close()
	// 				logrus.Infof("successful uploaded,fileName=%s,fileSize=%.2f MB,savePath=%s", f.Filename, float64(contentLen)/1024/1024, path)

	// 				w.Write([]byte("successful,url=" + url.QueryEscape(f.Filename)))
	// 			}
	// 		}
	// 	} else {
	// 		reader, err := r.MultipartReader()
	// 		if err != nil {
	// 			panic(err)
	// 		}
	// 		for {
	// 			p, err := reader.NextPart()
	// 			if err == io.EOF {
	// 				logrus.Infof("EOF break")
	// 				break
	// 			}

	// 			if err != nil {
	// 				logrus.Infof("reader.NextPart error:%s", err)
	// 				break
	// 			}

	// 			fileName := p.FileName()
	// 			logrus.Infof("fileName:%s", fileName)
	// 			if fileName != "" {
	// 				_, err = os.Stat(dir)
	// 				if err != nil && os.IsNotExist(err) {
	// 					//目录不存在
	// 					err = os.MkdirAll(dir, 0777)
	// 					if err != nil {
	// 						logrus.Infof("create dir failed:%s", err)
	// 						continue
	// 					}
	// 				}
	// 				fo, err := os.Create(dir + "/" + fileName)
	// 				if err != nil {
	// 					logrus.Infof("os create file err: ", err)
	// 					continue
	// 				}
	// 				defer fo.Close()
	// 				defer recordToSql2(fileName, userName, contentLen)
	// 				formValue, _ := ioutil.ReadAll(p)
	// 				fo.Write(formValue)
	// 			}
	// 		}
	// 	}
	// }

	// logrus.Infof("***********************************")

	// var bts = []byte("success")
	// w.Write(bts)
}

func broadCastOnline() {
	for client := range clients {
		contents, err := json.Marshal(onlineusers)
		if err != nil {
			logrus.Errorf("json Marshal onlineusers failed.")
			continue
		}

		var msg StringMessage
		msg.MessageType = websocket.TextMessage
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

	go pingClient(ws)

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
		var msg StringMessage
		// Send the newly received message to the broadcast channel
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

func pingClient(ws *websocket.Conn) {
	time.Sleep(10 * time.Second)
	tryCount := 0
	for tryCount < 3 {
		err := ws.WriteMessage(websocket.PingMessage, []byte("ping"))
		if err != nil {
			logrus.Errorf("ping client error:%v, tryCount:%d", err, tryCount)
			tryCount++
			continue
		}
		time.Sleep(10 * time.Second)
	}
	ws.Close()
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

func handleTCPConnections(tcpPort int) {
	logrus.Infof("TCP ready to start on port:%v", tcpPort)
	listener, err := net.Listen("tcp", "0.0.0.0:"+strconv.Itoa(tcpPort))
	if err != nil {
		logrus.Errorf("tcp listen on port:%v failed. error message is:%v", tcpPort, err)
		return
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			logrus.Errorf("tcp accept failed:%v", err)
			continue
		}

		go handleTCPConn(conn)
	}
}

func handleTCPConn(conn net.Conn) {
	//defer conn.Close()
	buf := make([]byte, 1024*1024)
	length, err := conn.Read(buf)
	if err != nil {
		logrus.Errorf("read from tcp conn failed:%v", err)
		return
	}

	if length <= 0 {
		logrus.Warnf("read nothing at first read.")
		return
	}

	//get fileName|fileSize|uploadUser from buf
	str := string(buf[:])
	strs := strings.Split(str, "|")
	if len(strs) < 4 {
		logrus.Errorf("tcp message error")
		return
	}
	tcpType := strs[0]
	if tcpType == "upload" {
		fileName := strs[1]
		fileSize, err := strconv.ParseInt(strs[2], 10, 64)
		if err != nil {
			logrus.Errorf("parseInt failed.")
			return
		}
		userName := strings.Fields(strs[3])[0]
		logrus.Infof("fileName:%s, fileSize:%d, userName:%s", fileName, fileSize, userName)
		uploadProcess(conn, fileName, fileSize, userName)
	} else if tcpType == "download" {
		fileName := strs[1]
		userName := strs[2]
		downloadProcess(conn, fileName, userName)
	}
}

func uploadProcess(conn net.Conn, fileName string, fileSize int64, userName string) {
	defer conn.Close()
	curDir := getCurrentDirectory()
	logrus.Infof("curDir:%s", curDir)
	dir := curDir + "/public/uploads/"
	_, err := os.Stat(dir)
	if err != nil && os.IsNotExist(err) {
		//目录不存在
		err = os.MkdirAll(dir, 0777)
		if err != nil {
			logrus.Infof("create dir failed:%v", err)
			return
		}
	}
	fileFullPath := dir + fileName
	f, err := os.OpenFile(fileFullPath, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0666)
	if err != nil {
		logrus.Errorf("open file:%s failed, error message:%v", fileFullPath, err)
		return
	}
	defer f.Close()
	defer recordToSql2(fileName, userName, fileSize)
	var totalSize int64
	totalSize = 0
	buf := make([]byte, 1024*1024)
	errFlag := false
	for totalSize < fileSize {
		length, err := conn.Read(buf)
		if err != nil {
			logrus.Errorf("socket read message error:%v", err)
			errFlag = true
			break
		}
		if length <= 0 {
			break
		}

		strLen := strconv.Itoa(length)
		length64, err := strconv.ParseInt(strLen, 10, 64)
		temp := buf[0:length]

		wLen, err := f.Write(temp)
		if err != nil {
			logrus.Errorf("write file error:%v", err)
			errFlag = true
			break
		}

		tempLen := length
		for wLen < tempLen {
			buf = buf[wLen+1:]
			tempLen = tempLen - wLen
			wLen, err = f.Write(buf)
		}
		totalSize = totalSize + length64
	}
	if errFlag {
		logrus.Infof("upload file:%s by:%s failed", fileName, userName)
	} else {
		logrus.Infof("upload file:%s by:%s success", fileName, userName)
	}
}

func downloadProcess(conn net.Conn, fileName string, userName string) {
	defer conn.Close()
	curDir := getCurrentDirectory()
	logrus.Infof("curDir:%s", curDir)
	dir := curDir + "/public/uploads/"
	fileFullPath := dir + fileName
	f, err := os.OpenFile(fileFullPath, os.O_RDONLY, 0666)
	if err != nil {
		logrus.Errorf("open file:%s failed, error message:%v", fileFullPath, err)
		return
	}
	defer f.Close()
	fileInfo, err := f.Stat()
	if err != nil {
		logrus.Errorf("read file:%s info failed, error message:%v", fileFullPath, err)
		return
	}
	fileSize := fileInfo.Size()
	var readSize int64
	readSize = 0
	buf := make([]byte, 1024*1024)
	//send file name and file size to client
	sendStr := fileName + "|" + strconv.FormatInt(fileSize, 10) + "|donotremove"
	logrus.Infof("download file info:%s", sendStr)
	sendBuf := []byte(sendStr)
	conn.Write(sendBuf)
	for readSize < fileSize {
		readLen, err := f.Read(buf)
		if err != nil {
			logrus.Errorf("read file:%s failed, error message:%v", fileFullPath, err)
			return
		}

		if readLen <= 0 {
			break
		}

		tempBuf := buf[0:readLen]
		writeLen, err := conn.Write(tempBuf)
		for writeLen < readLen {
			anotherBuf := tempBuf[writeLen:]
			anotherWriteLen, err := conn.Write(anotherBuf)
			if err != nil {
				return
			}
			writeLen += anotherWriteLen
		}
		readLen64, err := strconv.ParseInt(strconv.Itoa(readLen), 10, 64)
		readSize += readLen64
	}
	logrus.Infof("user:%s download file:%s success", userName, fileName)
}

func recordToSql(fileName string) bool {
	g_Db, err := connectSql()
	if err != nil {
		logrus.Errorf("recordToSql connect sql failed:[%v]", err.Error())
		return false
	}
	defer g_Db.Close()
	g_Mutex.Lock()
	defer g_Mutex.Unlock()
	strSql := "insert into chat_upload_files (file_name) values (?);"
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
	g_Db, err := connectSql()
	if err != nil {
		logrus.Errorf("recordToSql connect sql failed:[%v]", err.Error())
		return false
	}
	defer g_Db.Close()
	g_Mutex.Lock()
	defer g_Mutex.Unlock()
	strSql := "insert into chat_upload_files (file_name, upload_user, file_size) values (?, ?, ?);"
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
	g_Db, err := connectSql()
	if err != nil {
		logrus.Errorf("recordClient connect sql failed:[%v]", err.Error())
		return false
	}
	defer g_Db.Close()
	g_Mutex.Lock()
	defer g_Mutex.Unlock()
	strSql := "insert into easy_chat_client (version, version_number, file_name, file_size, md5, upload_time, upload_by) values (?, ?, ?, ?, ?, CURRENT_TIMESTAMP, ?);"
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
	res := HttpResponse{false, "obselete interface", -1, ""}
	ret, err := json.Marshal(res)
	if err != nil {
		logrus.Errorf("json marshal failed.")
		io.WriteString(w, "json marshal failed.")
		return
	}
	io.WriteString(w, string(ret))
	//
	// tokens := r.Header["Token"]
	// if tokens == nil {
	// 	res := HttpResponse{false, "need token", -1, ""}
	// 	ret, err := json.Marshal(res)
	// 	if err != nil {
	// 		logrus.Errorf("json marshal failed.")
	// 		io.WriteString(w, "json marshal failed.")
	// 		return
	// 	}
	// 	io.WriteString(w, string(ret))
	// 	return
	// }
	// token := tokens[0]
	// logrus.Infof("queryUploadFilesFunction token:%s", token)
	// if token != "20200101" {
	// 	res := HttpResponse{false, "Token verify failed.", -1, ""}
	// 	_, err := json.Marshal(res)
	// 	if err != nil {
	// 		logrus.Errorf("queryUploadFilesFunction json marshal failed.")
	// 		io.WriteString(w, "queryUploadFilesFunction json marshal failed.")
	// 		return
	// 	}
	// }

	// //check the new user exists or not
	// g_Db, err := connectSql()
	// msg := "connect sql failed"
	// if !checkErr(err, msg, w) {
	// 	return
	// }
	// defer g_Db.Close()
	// strSql := "select file_name from chat_upload_files order by create_time desc;"
	// stmt, err := g_Db.Prepare(strSql)
	// msg = "queryUploadFilesFunction Prepare sql failed 1."
	// if !checkErr(err, msg, w) {
	// 	return
	// }

	// rows, err := stmt.Query()
	// msg = "queryUploadFilesFunction Query sql failed."
	// if !checkErr(err, msg, w) {
	// 	return
	// }
	// defer rows.Close()

	// var fileName string
	// var files []string
	// for rows.Next() {
	// 	err := rows.Scan(&fileName)
	// 	msg = "queryUploadFilesFunction Failed to get sql item."
	// 	if !checkErr(err, msg, w) {
	// 		return
	// 	}
	// 	files = append(files, fileName)
	// }

	// response := FilesResponse{true, "success", files}
	// bts, err := json.Marshal(response)
	// if err != nil {
	// 	logrus.Errorf("queryUploadFilesFunction json marshal failed.")
	// 	io.WriteString(w, "queryUploadFilesFunction json marshal failed.")
	// 	return
	// }
	// io.WriteString(w, string(bts))
}

func queryUploadFilesFunction2(w http.ResponseWriter, r *http.Request) {
	res := HttpResponse{false, "obselete interface", -1, ""}
	ret, err := json.Marshal(res)
	if err != nil {
		logrus.Errorf("json marshal failed.")
		io.WriteString(w, "json marshal failed.")
		return
	}
	io.WriteString(w, string(ret))

	// tokens := r.Header["Token"]
	// if tokens == nil {
	// 	res := HttpResponse{false, "need token", -1, ""}
	// 	ret, err := json.Marshal(res)
	// 	if err != nil {
	// 		logrus.Errorf("queryUploadFilesFunction2 json marshal failed.")
	// 		io.WriteString(w, "queryUploadFilesFunction2 json marshal failed.")
	// 		return
	// 	}
	// 	io.WriteString(w, string(ret))
	// 	return
	// }
	// token := tokens[0]
	// logrus.Infof("queryUploadFilesFunction2 token:%s", token)
	// if token != "20200101" {
	// 	res := HttpResponse{false, "Token verify failed.", -1, ""}
	// 	_, err := json.Marshal(res)
	// 	if err != nil {
	// 		logrus.Errorf("queryUploadFilesFunction2 json marshal failed.")
	// 		io.WriteString(w, "json marshal failed.")
	// 		return
	// 	}
	// }

	// //check the new user exists or not

	// g_Db, err := connectSql()
	// msg := "connect sql failed"
	// if !checkErr(err, msg, w) {
	// 	return
	// }
	// defer g_Db.Close()
	// strSql := "select file_name,file_size,upload_user,create_time from chat_upload_files order by create_time desc;"
	// stmt, err := g_Db.Prepare(strSql)
	// msg = "queryUploadFilesFunction2 Prepare sql failed 1."
	// if !checkErr(err, msg, w) {
	// 	return
	// }

	// rows, err := stmt.Query()
	// msg = "queryUploadFilesFunction2 Query sql failed."
	// if !checkErr(err, msg, w) {
	// 	return
	// }
	// defer rows.Close()

	// var fileName string
	// var fileSize int
	// var uploadUser sql.NullString
	// var createTime sql.NullTime
	// var files []FileInfo
	// for rows.Next() {
	// 	err := rows.Scan(&fileName, &fileSize, &uploadUser, &createTime)
	// 	msg = "queryUploadFilesFunction2 Failed to get sql item."
	// 	if !checkErr(err, msg, w) {
	// 		return
	// 	}
	// 	var fileInfo FileInfo
	// 	fileInfo.FileName = fileName
	// 	fileInfo.FileSize = fileSize
	// 	fileInfo.UploadUser = uploadUser
	// 	fileInfo.CreateTime = createTime
	// 	files = append(files, fileInfo)
	// }

	// response := FilesResponse2{true, "success", files}
	// bts, err := json.Marshal(response)
	// if err != nil {
	// 	logrus.Errorf("queryUploadFilesFunction2 json marshal failed.")
	// 	io.WriteString(w, "queryUploadFilesFunction2 json marshal failed.")
	// 	return
	// }
	// io.WriteString(w, string(bts))
}

func deleteFile(w http.ResponseWriter, r *http.Request) {
	res := HttpResponse{false, "obselete interface", -1, ""}
	ret, err := json.Marshal(res)
	if err != nil {
		logrus.Errorf("json marshal failed.")
		io.WriteString(w, "json marshal failed.")
		return
	}
	io.WriteString(w, string(ret))

	// tokens := r.Header["Token"]
	// if tokens == nil {
	// 	res := HttpResponse{false, "need token", -1, ""}
	// 	ret, err := json.Marshal(res)
	// 	if err != nil {
	// 		logrus.Errorf("json marshal failed.")
	// 		io.WriteString(w, "json marshal failed.")
	// 		return
	// 	}
	// 	io.WriteString(w, string(ret))
	// 	return
	// }
	// token := tokens[0]
	// if token != "20200101" {
	// 	res := HttpResponse{false, "Token verify failed.", -1, ""}
	// 	_, err := json.Marshal(res)
	// 	if err != nil {
	// 		logrus.Errorf("json marshal failed.")
	// 		io.WriteString(w, "json marshal failed.")
	// 		return
	// 	}
	// }

	// utf8Reader := transform.NewReader(r.Body, simplifiedchinese.GBK.NewDecoder())
	// body, err := ioutil.ReadAll(utf8Reader)
	// msg := "delete file ReadAll body failed."
	// if !checkErr(err, msg, w) {
	// 	return
	// }

	// var deleteFile DeleteFile
	// err = json.Unmarshal(body, &deleteFile)
	// msg = "delete file json unmarshal failed."
	// if !checkErr(err, msg, w) {
	// 	return
	// }

	// //Check username and password from mysql

	// g_Db, err := connectSql()
	// msg = "connect sql failed"
	// if !checkErr(err, msg, w) {
	// 	return
	// }
	// defer g_Db.Close()
	// strSql := "select id, file_name from chat_upload_files where file_name=?"
	// stmt, err := g_Db.Prepare(strSql)
	// msg = "Prepare sql failed."
	// if !checkErr(err, msg, w) {
	// 	return
	// }

	// rows, err := stmt.Query(deleteFile.FileName)
	// msg = "Query sql failed."
	// if !checkErr(err, msg, w) {
	// 	return
	// }
	// defer rows.Close()

	// g_Mutex.Lock()
	// defer g_Mutex.Unlock()
	// var id int
	// var file_name string
	// for rows.Next() {
	// 	err := rows.Scan(&id, &file_name)
	// 	logrus.Infof("fileName:%s", file_name)
	// 	msg = "Failed to get sql item."
	// 	if !checkErr(err, msg, w) {
	// 		return
	// 	}

	// 	delSql := "delete from chat_upload_files where file_name='" + deleteFile.FileName + "'"
	// 	g_Db.Query(delSql)
	// }

	// response := HttpResponse{true, "success", id, file_name}
	// bts, err := json.Marshal(response)
	// if err != nil {
	// 	logrus.Errorf("json marshal failed.")
	// 	io.WriteString(w, "json marshal failed.")
	// 	return
	// }

	// f := g_strWorkDir + "/public/uploads/" + deleteFile.FileName
	// err = os.Remove(f)
	// if err != nil {
	// 	msg = "remove file from disk failed.{}"
	// 	logrus.Errorf(msg, err)
	// }
	// io.WriteString(w, string(bts))
}

func deleteFile2(w http.ResponseWriter, r *http.Request) {
	res := HttpResponse{false, "obselete interface", -1, ""}
	ret, err := json.Marshal(res)
	if err != nil {
		logrus.Errorf("json marshal failed.")
		io.WriteString(w, "json marshal failed.")
		return
	}
	io.WriteString(w, string(ret))

	// tokens := r.Header["Token"]
	// if tokens == nil {
	// 	res := HttpResponse{false, "need token", -1, ""}
	// 	ret, err := json.Marshal(res)
	// 	if err != nil {
	// 		logrus.Errorf("json marshal failed.")
	// 		io.WriteString(w, "json marshal failed.")
	// 		return
	// 	}
	// 	io.WriteString(w, string(ret))
	// 	return
	// }
	// token := tokens[0]
	// if token != "20200101" {
	// 	res := HttpResponse{false, "Token verify failed.", -1, ""}
	// 	_, err := json.Marshal(res)
	// 	if err != nil {
	// 		logrus.Errorf("json marshal failed.")
	// 		io.WriteString(w, "json marshal failed.")
	// 		return
	// 	}
	// }

	// utf8Reader := transform.NewReader(r.Body, simplifiedchinese.GBK.NewDecoder())
	// body, err := ioutil.ReadAll(utf8Reader)
	// msg := "delete file ReadAll body failed."
	// if !checkErr(err, msg, w) {
	// 	return
	// }

	// var deleteFile DeleteFile2
	// err = json.Unmarshal(body, &deleteFile)
	// msg = "delete file json unmarshal failed."
	// if !checkErr(err, msg, w) {
	// 	return
	// }

	// logrus.Infof("delete file:%s by user:%s", deleteFile.FileName, deleteFile.UserName)

	// //Check username and password from mysql
	// g_Db, err := connectSql()
	// msg = "connect sql failed"
	// if !checkErr(err, msg, w) {
	// 	return
	// }
	// defer g_Db.Close()
	// strSql := "select id, file_name from chat_upload_files where file_name=?"
	// stmt, err := g_Db.Prepare(strSql)
	// msg = "Prepare sql failed."
	// if !checkErr(err, msg, w) {
	// 	return
	// }

	// rows, err := stmt.Query(deleteFile.FileName)
	// msg = "Query sql failed."
	// if !checkErr(err, msg, w) {
	// 	return
	// }
	// defer rows.Close()
	// g_Mutex.Lock()
	// defer g_Mutex.Unlock()
	// var id int64
	// delSql := "delete from chat_upload_files where file_name='" + deleteFile.FileName + "'"
	// if g_sqlConfig.SqliteFlag {
	// 	ret, err := g_Db.Exec(delSql)
	// 	if err != nil {
	// 		logrus.Errorf("sqlite db inuse:%d", g_Db.Stats().InUse)
	// 		logrus.Errorf("sqlite delete failed:[%v]", err.Error())
	// 		response := HttpResponse{false, "delete failed", -1, deleteFile.FileName}
	// 		bts, err := json.Marshal(response)
	// 		if err != nil {
	// 			logrus.Errorf("json marshal failed.")
	// 			io.WriteString(w, "json marshal failed.")
	// 			return
	// 		}
	// 		io.WriteString(w, string(bts))
	// 		return
	// 	}
	// 	id, err = ret.RowsAffected()
	// 	if !checkErr(err, msg, w) {
	// 		return
	// 	}
	// } else {
	// 	_, err = g_Db.Query(delSql)
	// 	if err != nil {
	// 		logrus.Errorf("mysql delete failed:[%v]", err.Error())
	// 	}
	// }

	// response := HttpResponse{true, "success", int(id), deleteFile.FileName}
	// bts, err := json.Marshal(response)
	// if err != nil {
	// 	logrus.Errorf("json marshal failed.")
	// 	io.WriteString(w, "json marshal failed.")
	// 	return
	// }

	// f := g_strWorkDir + "/public/uploads/" + deleteFile.FileName
	// err = os.Remove(f)
	// if err != nil {
	// 	msg = "remove file from disk failed.%v"
	// 	logrus.Errorf(msg, err)
	// }
	// io.WriteString(w, string(bts))
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
