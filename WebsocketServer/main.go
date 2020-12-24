package main

import (
	"log"
	"net/http"
	"io"
	"io/ioutil"
	"encoding/json"
	"fmt"
	"strings"
	"os"
	"mime"
	_ "mime/multipart"
	"path/filepath"
	"net/url"

	"github.com/gorilla/websocket"

	_ "github.com/go-sql-driver/mysql"
	"database/sql"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

var clients = make(map[*websocket.Conn]bool) // connected clients
var broadcast = make(chan StringMessage)           // broadcast channel
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
	Message 	[]byte
}

type LoginUser struct {
	UserName string
	Password string
}

type RegisterUser struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email string `json:"email"`
	Mobile string `json:"mobile"`
}

type SqlConfig struct {
	Host string	`json:"host"`
	Port int	`json:"port"`
	Database string	`json:"database"`
	UserName string	`json:"username"`
	Password string	`json:"password"`
	Charset string	`json:"charset"`
}

type SqlUser struct {
	Id int
	UserName string
	Mobile string
	Password string
	CreateTime string
	ModifyTime string
}

type OnlineSt struct {
	Userid string
	Username string
}

type OnlineUser struct {
	Online OnlineSt
	Addr string
}

type FileInfo struct {
	FileName string
	FileSize int
	UploadUser sql.NullString
}

type HttpResponse struct {
	Success bool
	Msg string
	Id int
	Username string
}

type FilesResponse struct {
	Success bool
	Msg string
	Files []string
}

type FilesResponse2 struct {
	Success bool
	Msg string
	Files []FileInfo
}

type DeleteFile struct {
	FileName string
}

var g_sqlConfig SqlConfig
var g_Db *sql.DB
var g_strWorkDir string

func ReadConfig(path string) (SqlConfig) {
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
	log.Println("dsn:", dsn)
    db, err := sql.Open("mysql", dsn)
    if err != nil {
		log.Printf("mysql connect failed, detail is [%v]", err.Error())
		return nil, err
    }
    return db, nil
}

func main() {
	g_strWorkDir = getCurrentDirectory()
	//Read config
	configPath := "./config/config.json"
	g_sqlConfig = ReadConfig(configPath)
	log.Println("sqlConfig:", g_sqlConfig)
	var err error
	g_Db, err = connectSql()
	if err != nil {
		log.Println("Connect sql failed.")
		return
	}
	defer g_Db.Close()

	// Create a simple file server
	fs := http.FileServer(http.Dir("./public"))
	http.Handle("/", fs)

	//http request response
	http.HandleFunc("/uploads", uploadFunction)
	http.HandleFunc("/uploads2", uploadFunction2)
	http.HandleFunc("/login", loginFunction)
	http.HandleFunc("/register", registerFunction)
	http.HandleFunc("/uploadfiles", queryUploadFilesFunction)
	http.HandleFunc("/uploadfiles2", queryUploadFilesFunction2)
	http.HandleFunc("/delfile", deleteFile)

	// Configure websocket route
	http.HandleFunc("/ws", handleConnections)

	// Start listening for incoming chat messages
	go handleMessages()

	// Start the server on localhost port 8000 and log any errors
	log.Println("http server started on :5133")
	err = http.ListenAndServe(":5133", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func getCurrentDirectory() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	return strings.Replace(dir, "\\", "/", -1)
}

func checkErr(err error, msg string, w http.ResponseWriter) (bool) {
	if err != nil {
		response := HttpResponse{false, msg + err.Error(), -1, ""}
		bts, err := json.Marshal(response)
		if err != nil {
			log.Println("json marshal failed.")
			io.WriteString(w, "json marshal failed.")
			return false
		}
		log.Println(string(bts))
		io.WriteString(w, string(bts))
		return false
	}
	return true
}

func uploadFunction(w http.ResponseWriter, r *http.Request) {
	if (r.Method != "POST") {
		log.Println("invalid request")
		io.WriteString(w, "invalid request")
		return
	}

	contentType := r.Header.Get("Content-Type")
	contentLen := r.ContentLength
	log.Println("contentType: ", contentType, " contentLen:", contentLen)
	mediatype, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		log.Println("ParseMediaType error: ", err)
		w.Write([]byte("ParseMediaType error"))
		return
	}
	curDir := getCurrentDirectory()
	log.Println("curDir:", curDir)
	dir := curDir + "/public/uploads"
	log.Println("mediatype: ", mediatype)
	if mediatype == "multipart/form-data" {
		log.Println("in multipart parsing...")
		log.Println("r.MultipartForm:", r.MultipartForm)
		if (r.MultipartForm != nil) {
			for name, files := range r.MultipartForm.File {
				log.Printf("req.MultipartForm.File,name=:", name, " files:", len(files))
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
					fmt.Printf("successful uploaded,fileName=%s,fileSize=%.2f MB,savePath=%s \n", f.Filename, float64(contentLen)/1024/1024, path)
		
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
					log.Println("EOF break")
					break;
				}

				if err != nil {
					log.Println("reader.NextPart error!", err)
					break;
				}

				fileName := p.FileName()
				log.Println("fileName:", fileName)
				if fileName != "" {
					_, err = os.Stat(dir)
					if err != nil && os.IsNotExist(err) {
						//目录不存在
						err = os.MkdirAll(dir, 0777)
						if err != nil {
							log.Println("create dir failed:", err)
							continue
						}
					}
					fo, err := os.Create(dir + "/" + fileName)
					if err != nil {
						log.Println("os create file err: ", err)
						continue
					}
					defer fo.Close()
					defer recordToSql(fileName)
					formValue, _ := ioutil.ReadAll(p)
					fo.Write(formValue);
				}
			}
		}
	}

	log.Println("***********************************")

	var bts = []byte("success");
	w.Write(bts)
}

func uploadFunction2(w http.ResponseWriter, r *http.Request) {
	if (r.Method != "POST") {
		log.Println("invalid request")
		io.WriteString(w, "invalid request")
		return
	}

	contentType := r.Header.Get("Content-Type")
	contentLen := r.ContentLength
	userName := r.Header.Get("UserName")
	log.Println("contentType: ", contentType, " contentLen:", contentLen, " user:", userName)
	mediatype, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		log.Println("ParseMediaType error: ", err)
		w.Write([]byte("ParseMediaType error"))
		return
	}
	curDir := getCurrentDirectory()
	log.Println("curDir:", curDir)
	dir := curDir + "/public/uploads"
	log.Println("mediatype: ", mediatype)
	if mediatype == "multipart/form-data" {
		log.Println("in multipart parsing...")
		log.Println("r.MultipartForm:", r.MultipartForm)
		if (r.MultipartForm != nil) {
			for name, files := range r.MultipartForm.File {
				log.Printf("req.MultipartForm.File,name=:", name, " files:", len(files))
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
					fmt.Printf("successful uploaded,fileName=%s,fileSize=%.2f MB,savePath=%s \n", f.Filename, float64(contentLen)/1024/1024, path)
		
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
					log.Println("EOF break")
					break;
				}

				if err != nil {
					log.Println("reader.NextPart error!", err)
					break;
				}

				fileName := p.FileName()
				log.Println("fileName:", fileName)
				if fileName != "" {
					_, err = os.Stat(dir)
					if err != nil && os.IsNotExist(err) {
						//目录不存在
						err = os.MkdirAll(dir, 0777)
						if err != nil {
							log.Println("create dir failed:", err)
							continue
						}
					}
					fo, err := os.Create(dir + "/" + fileName)
					if err != nil {
						log.Println("os create file err: ", err)
						continue
					}
					defer fo.Close()
					defer recordToSql2(fileName, userName, contentLen)
					formValue, _ := ioutil.ReadAll(p)
					fo.Write(formValue);
				}
			}
		}
	}

	log.Println("***********************************")

	var bts = []byte("success");
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
				log.Println("json marshal failed.")
				io.WriteString(w, "json marshal failed.")
				return
			}
			log.Println(string(bts))
			io.WriteString(w, string(bts))
			return
		}
		count = count + 1
	}
	if count < 1 {
		response := HttpResponse{false, "User does not exist.", -1, ""}
		bts, err := json.Marshal(response)
		if err != nil {
			log.Println("json marshal failed.")
			io.WriteString(w, "json marshal failed.")
			return
		}
		log.Println(string(bts))
		io.WriteString(w, string(bts))
		return
	}
	response := HttpResponse{true, "success", id, user_name}
	bts, err := json.Marshal(response)
	if err != nil {
		log.Println("json marshal failed.")
		io.WriteString(w, "json marshal failed.")
		return
	}
	io.WriteString(w, string(bts))
}

func registerFunction(w http.ResponseWriter, r *http.Request) {
	token := r.Header["Token"][0]
	log.Println("token:", token)
	if token != "20200101" {
		res := HttpResponse{false, "Token verify failed.", -1, ""}
		_, err := json.Marshal(res)
		if err != nil {
			log.Println("json marshal failed.")
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
				log.Println("json marshal failed.")
				io.WriteString(w, "json marshal failed.")
				return
			}
			log.Println(string(bts))
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
	log.Println("new Insert id:", newId)

	response := HttpResponse{true, "success", id, regUser.Username}
	bts, err := json.Marshal(response)
	if err != nil {
		log.Println("json marshal failed.")
		io.WriteString(w, "json marshal failed.")
		return
	}
	io.WriteString(w, string(bts))
}

func broadCastOnline() {
	for client := range clients {
		contents, err := json.Marshal(onlineusers)
		if err != nil {
			log.Println("json Marshal onlineusers failed.")
			continue
		}

		var msg StringMessage
		msg.MessageType = 1
		msg.Message = contents
		err = client.WriteMessage(msg.MessageType, msg.Message)
		if err != nil {
			log.Printf("Broadcast online message error: %v", err)
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
	log.Println("ws:", ws.RemoteAddr(), " Network:", ws.RemoteAddr().Network(), " String:", ws.RemoteAddr().String())
	//Whenever a new client was connected, send the online message to all clients
	broadCastOnline()

	for {
		//var msg Message
		// Read in a new message as JSON and map it to a Message object
		messageType, p, err := ws.ReadMessage()
		if err != nil {
			log.Printf("ReadMessage error: %v", err)
			addr := ws.RemoteAddr().String()
			for index, item := range onlineusers {
				if item.Addr == addr {
					onlineusers = append(onlineusers[:index], onlineusers[index + 1:]...)
				}
			}
			delete(clients, ws)
			log.Println("Current online user count:", len(onlineusers))
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
				log.Println("json unmarshal online info failed.")
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
		log.Println("Current online user count:", len(onlineusers))
	}
}

func handleMessages() {
	for {
		// Grab the next message from the broadcast channel
		msg := <-broadcast
		// Send it out to every client that is currently connected
		for client := range clients {
			err := client.WriteMessage(msg.MessageType, msg.Message)
			//log.Println(msg.Message)
			if err != nil {
				log.Printf("WriteMessage error: %v", err)
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
		log.Println(msg)
		return false
	}
	
	res, err := stmt.Exec(fileName)
	if err != nil {
		log.Println("insert into sql failed in recordToSql")
		return false
	}

	newId, err := res.LastInsertId()
	if err != nil {
		log.Println("Get last inserted id failed in recordToSql")
		return false
	}
	log.Println("new Inserted id:", newId)
	return true
}

func recordToSql2(fileName string, userName string, fileSize int64) bool {
	strSql := "insert chat_upload_files (file_name, upload_user, file_size) values (?, ?, ?);"
	stmt, err := g_Db.Prepare(strSql)
	msg := "Prepare sql failed in recordToSql"
	if err != nil {
		log.Println(msg)
		return false
	}
	
	res, err := stmt.Exec(fileName, userName, fileSize)
	if err != nil {
		log.Println("insert into sql failed in recordToSql")
		return false
	}

	newId, err := res.LastInsertId()
	if err != nil {
		log.Println("Get last inserted id failed in recordToSql")
		return false
	}
	log.Println("new Inserted id:", newId)
	return true
}

func queryUploadFilesFunction(w http.ResponseWriter, r *http.Request) {
	tokens := r.Header["Token"]
	if tokens == nil {
		res := HttpResponse{false, "need token", -1, ""}
		ret, err := json.Marshal(res)
		if err != nil {
			log.Println("json marshal failed.")
			io.WriteString(w, "json marshal failed.")
			return
		}
		io.WriteString(w, string(ret))
		return
	}
	token := tokens[0]
	log.Println("token:", token)
	if token != "20200101" {
		res := HttpResponse{false, "Token verify failed.", -1, ""}
		_, err := json.Marshal(res)
		if err != nil {
			log.Println("json marshal failed.")
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
		log.Println("json marshal failed.")
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
			log.Println("json marshal failed.")
			io.WriteString(w, "json marshal failed.")
			return
		}
		io.WriteString(w, string(ret))
		return
	}
	token := tokens[0]
	log.Println("token:", token)
	if token != "20200101" {
		res := HttpResponse{false, "Token verify failed.", -1, ""}
		_, err := json.Marshal(res)
		if err != nil {
			log.Println("json marshal failed.")
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
		log.Println("json marshal failed.")
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
			log.Println("json marshal failed.")
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
			log.Println("json marshal failed.")
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
		log.Println("fileName:", file_name)
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
		log.Println("json marshal failed.")
		io.WriteString(w, "json marshal failed.")
		return
	}

	f := g_strWorkDir + "/public/uploads/" + deleteFile.FileName
	err = os.Remove(f)
	if err != nil {
		msg = "remove file from disk failed.{}"
		log.Println(msg, err)
	}
	io.WriteString(w, string(bts))
}