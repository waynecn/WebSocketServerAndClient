package main

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/sirupsen/logrus"
)

func loginFunction(w http.ResponseWriter, r *http.Request) {
	logrus.Infof("interface:loginFunction")
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
	g_Db, err := connectSql()
	msg = "connect sql failed"
	if !checkErr(err, msg, w) {
		return
	}
	defer g_Db.Close()
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
	g_Db, err := connectSql()
	msg = "connect sql failed"
	if !checkErr(err, msg, w) {
		return
	}
	defer g_Db.Close()
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
		//return
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
	logrus.Infof("register request")
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
	g_Db, err := connectSql()
	msg = "connect sql failed"
	if !checkErr(err, msg, w) {
		return
	}
	defer g_Db.Close()
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
	g_Mutex.Lock()
	defer g_Mutex.Unlock()
	strSql = "insert into chat_user (user_name,mobile,password) values(?,?,?);"
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
