package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"net/url"
	"os"

	"github.com/sirupsen/logrus"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

/*
根据用户查询该用户自行
*/
func queryUploadFilesFunction3(w http.ResponseWriter, r *http.Request) {
	params := r.URL.Query()
	userId := params.Get("userId")
	if userId == "" {
		NormalResponse(w, false, "need param: userId", -1, "", runFuncName())
		return
	}

	queryUserSql := "select user_type from chat_user where id=?;"

	db, err := connectSql()
	if err != nil {
		NormalResponse(w, false, "connect sql error", -1, "", runFuncName())
		return
	}
	defer db.Close()
	stmt, err := db.Prepare(queryUserSql)
	if err != nil {
		NormalResponse(w, false, "Prepare query user sql error", -1, "", runFuncName())
		return
	}

	rows, err := stmt.Query(userId)
	if err != nil {
		NormalResponse(w, false, "Exec query user error", -1, "", runFuncName())
		return
	}

	var userType sql.NullInt16
	for rows.Next() {
		err = rows.Scan(&userType)
		if err != nil {
			NormalResponse(w, false, "scan user query result error", -1, "", runFuncName())
			return
		}
		break
	}
	rows.Close()
	if !userType.Valid {
		NormalResponse(w, false, "invalid user", -1, "", runFuncName())
		return
	}
	if userType.Int16 == 1 {
		queryFilesUrl := "select file_name,file_size,upload_user,create_time from chat_upload_files order by create_time desc;"
		stmt, err = db.Prepare(queryFilesUrl)
		if err != nil {
			NormalResponse(w, false, "Prepare query file sql error", -1, "", runFuncName())
			return
		}
		rows, err = stmt.Query()
		if err != nil {
			NormalResponse(w, false, "Query file error", -1, "", runFuncName())
			return
		}
	} else {
		queryFilesUrl := "select file_name,file_size,upload_user,create_time from chat_upload_files where user_id=? " +
			" or to_user_id=? order by create_time desc;"
		stmt, err = db.Prepare(queryFilesUrl)
		if err != nil {
			NormalResponse(w, false, "Prepare query file sql error", -1, "", runFuncName())
			return
		}
		rows, err = stmt.Query(userId, userId)
		if err != nil {
			NormalResponse(w, false, "Query file error", -1, "", runFuncName())
			return
		}
	}
	var fileName string
	var fileSize int
	var uploadUser sql.NullString
	var createTime sql.NullTime
	var files []FileInfo
	for rows.Next() {
		err = rows.Scan(&fileName, &fileSize, &uploadUser, &createTime)
		if err != nil {
			NormalResponse(w, false, "scan file query result error", -1, "", runFuncName())
			return
		}

		var fileInfo FileInfo
		fileInfo.FileName = fileName
		fileInfo.FileSize = fileSize
		fileInfo.UploadUser = uploadUser
		fileInfo.CreateTime = createTime
		files = append(files, fileInfo)
	}
	defer rows.Close()
	response := FilesResponse2{true, "success", files}
	bts, err := json.Marshal(response)
	if err != nil {
		logrus.Errorf("queryUploadFilesFunction2 json marshal failed.")
		io.WriteString(w, "queryUploadFilesFunction2 json marshal failed.")
		return
	}
	io.WriteString(w, string(bts))
}

func deleteFile3(w http.ResponseWriter, r *http.Request) {
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

	if deleteFile.UserId <= 0 {
		NormalResponse(w, false, "need userId param", -1, "", runFuncName())
		return
	}

	logrus.Infof("delete file:%s by user:%s userId:%d", deleteFile.FileName, deleteFile.UserName, deleteFile.UserId)

	//Check username and password from mysql
	g_Db, err := connectSql()
	msg = "connect sql failed"
	if !checkErr(err, msg, w) {
		return
	}
	defer g_Db.Close()

	queryUserSql := "select user_type from chat_user where id=?;"
	stmt, err := g_Db.Prepare(queryUserSql)
	if err != nil {
		NormalResponse(w, false, "prepare query user sql failed.", -1, "", runFuncName())
		return
	}
	rows, err := stmt.Query(deleteFile.UserId)
	if err != nil {
		NormalResponse(w, false, "query user sql failed.", -1, "", runFuncName())
		return
	}
	defer rows.Close()
	var userType sql.NullInt16
	count := 0
	for rows.Next() {
		err = rows.Scan(&userType)
		if err != nil {
			NormalResponse(w, false, "scan user failed.", -1, "", runFuncName())
			return
		}
		count++
	}
	if count <= 0 {
		NormalResponse(w, false, "cannot find user.", -1, "", runFuncName())
		return
	}

	strSql := "select id, file_name from chat_upload_files where file_name=?"
	stmt, err = g_Db.Prepare(strSql)
	msg = "Prepare sql failed."
	if !checkErr(err, msg, w) {
		return
	}

	rows, err = stmt.Query(deleteFile.FileName)
	msg = "Query sql failed."
	if !checkErr(err, msg, w) {
		return
	}
	defer rows.Close()
	g_Mutex.Lock()
	defer g_Mutex.Unlock()
	var id int64
	delSql := "delete from chat_upload_files where file_name='" + deleteFile.FileName + "'"
	if g_sqlConfig.SqliteFlag {
		ret, err := g_Db.Exec(delSql)
		if err != nil {
			logrus.Errorf("sqlite db inuse:%d", g_Db.Stats().InUse)
			logrus.Errorf("sqlite delete failed:[%v]", err.Error())
			response := HttpResponse{false, "delete failed", -1, deleteFile.FileName}
			bts, err := json.Marshal(response)
			if err != nil {
				logrus.Errorf("json marshal failed.")
				io.WriteString(w, "json marshal failed.")
				return
			}
			io.WriteString(w, string(bts))
			return
		}
		id, err = ret.RowsAffected()
		if !checkErr(err, msg, w) {
			return
		}
	} else {
		_, err = g_Db.Query(delSql)
		if err != nil {
			logrus.Errorf("mysql delete failed:[%v]", err.Error())
		}
	}

	response := HttpResponse{true, "success", int(id), deleteFile.FileName}
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

func uploadFunction3(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		logrus.Errorf("invalid request")
		io.WriteString(w, "invalid request")
		return
	}

	contentType := r.Header.Get("Content-Type")
	contentLen := r.ContentLength
	userName := r.Header.Get("UserName")
	userId := r.Header.Get("UserId")
	if !checkUserExist(userId) {
		NormalResponse(w, false, "user doesn't exist", -1, "", runFuncName())
		return
	}
	toUserId := r.Header.Get("ToUserId")
	logrus.Infof("contentType:%s contentLen:%s user:%s userId:%s", contentType, contentLen, userName, userId)
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
					defer recordToSql3(fileName, userName, contentLen, userId, toUserId)
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

func recordToSql3(fileName string, userName string, fileSize int64, userId string, toUserId string) bool {
	g_Db, err := connectSql()
	if err != nil {
		logrus.Errorf("recordToSql connect sql failed:[%v]", err.Error())
		return false
	}
	defer g_Db.Close()
	g_Mutex.Lock()
	defer g_Mutex.Unlock()
	strSql := "insert into chat_upload_files (file_name, upload_user, file_size, user_id, to_user_id) values (?, ?, ?, ?, ?);"
	stmt, err := g_Db.Prepare(strSql)
	msg := "Prepare sql failed in recordToSql"
	if err != nil {
		logrus.Errorf(msg)
		return false
	}

	res, err := stmt.Exec(fileName, userName, fileSize, userId, toUserId)
	if err != nil {
		logrus.Errorf("insert into sql failed in recordToSql:", err)
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
