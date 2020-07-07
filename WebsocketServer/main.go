package main

import (
	"log"
	"net/http"
	"io"
	"io/ioutil"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gorilla/websocket"

	_ "github.com/go-sql-driver/mysql"
	"database/sql"
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

var g_sqlConfig SqlConfig
var g_Db *sql.DB

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
		fmt.Printf("mysql connect failed, detail is [%v]", err.Error())
		return nil, err
    }
    return db, nil
}

func main() {
	//Read config
	configPath := "./config/config.json"
	g_sqlConfig = ReadConfig(configPath)
	fmt.Println("sqlConfig:", g_sqlConfig)
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
	http.HandleFunc("/login", loginFunction)
	http.HandleFunc("/register", registerFunction)

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

type HttpResponse struct {
	Success bool
	Msg string
	Id int
	Username string
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
