package controllers

import (
	"fmt"
	"github.com/go-xorm/xorm"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/mvc"
	"github.com/kataras/iris/v12/sessions"
	"io/ioutil"
	"knowledge-graph-management-system/models"
	"mime/multipart"
	"os"
	"regexp"
	"strconv"
	"strings"
)

type Controller struct {
	Context iris.Context
	Session *sessions.Session
	MySQL   *xorm.Engine
	Cache   *models.Cache
}

func (c Controller) BeforeActivation(b mvc.BeforeActivation) {
	b.Handle("GET", "/IsLogin", "IsLogin")
	b.Handle("POST", "/Login", "Login")
	b.Handle("POST", "/ParseFile", "Prase")
	b.Handle("POST", "/Search", "Search")
	b.Handle("GET", "/FileList", "FileList")
	b.Handle("GET", "/SubjectList", "SubjectList")
	b.Handle("GET", "/RemoveFile/{name}", "Remove")
	b.Handle("GET", "/Find/{subject}/{key}/{from}/{to}", "Find")
	b.Handle("GET", "/GetFiles/{subject}", "GetFileList")
	b.Handle("POST", "/GetDataFile", "GetFileContent")
}

func beforeSave(ctx iris.Context, file *multipart.FileHeader) {
	dataMap := ctx.FormValues()
	_, ok := dataMap["filename"]
	if !ok {
		return
	}
	fileName := file.Filename
	names := strings.Split(fileName, ".")
	file.Filename = dataMap["filename"][0] + "." + names[len(names)-1]
}

func (c Controller) Authorize() bool {
	userID := c.Session.GetString("user")
	if userID == models.User {
		return true
	}
	return false
}

func (c Controller) IsLogin() mvc.Result {
	login := models.Failure
	if c.Authorize() {
		login = models.Success
	}
	return mvc.Response{
		Object: models.Response{
			Status: login,
		},
	}
}

func (c Controller) Login() mvc.Result {
	var loginInfo models.LoginInfo
	err := c.Context.ReadJSON(&loginInfo)
	if err != nil {
		return mvc.Response{
			Object: models.Response{
				Status: models.Failure,
				Result: models.Retry,
			},
		}
	}
	if loginInfo.Password == models.Pass && loginInfo.UserID == models.User {
		c.Session.Set("user", loginInfo.UserID)
		return mvc.Response{
			Object: models.Response{
				Status: models.Success,
			},
		}
	}
	return mvc.Response{
		Object: models.Response{
			Status: models.Failure,
			Result: models.PasswdError,
		},
	}
}

func (c *Controller) Prase() mvc.Result {
	data := c.Context.FormValues()
	if !c.Authorize() {
		return mvc.Response{
			Object: models.Response{
				Status: models.Failure,
				Result: models.NoPermission,
			},
		}
	}
	path, _ := os.Getwd()
	fileName := data["filename"][0]
	subject := data["subject"][0]
	fileName = subject + "/" + fileName
	if fileName == "" || subject == "" {
		return mvc.Response{
			Object: models.Response{
				Status: models.Failure,
				Result: models.Retry,
			},
		}
	}
	fileDetail := models.Files{
		Name:    fileName,
		Subject: subject,
	}
	c.MySQL.Get(&fileDetail)
	if fileDetail.Id > 0 {
		delete(c.Cache.FileContent, fileDetail.Name)
		index := models.KeyIndex{
			FileName: fileDetail.Name,
			Subject:  subject,
		}
		c.MySQL.Delete(&index)
		c.MySQL.Delete(&fileDetail)
	}

	c.Context.UploadFormFiles(path+"/resources/"+subject+"/", beforeSave)

	subjectModel := models.Subject{Name: subject}
	go func() {
		c.MySQL.Get(&subjectModel)
		if subjectModel.Id <= 0 {
			c.MySQL.Insert(&subjectModel)
		}
	}()
	if fileName == "" {
		return mvc.Response{
			Object: models.Response{
				Status: models.Failure,
				Result: models.Retry,
			},
		}
	}
	file, err := os.Open(path + "/resources/" + fileName + ".md")
	if err != nil {
		return mvc.Response{
			Object: models.Response{
				Status: models.Failure,
				Result: models.Retry,
			},
		}
	}
	defer file.Close()
	content, err := ioutil.ReadAll(file)
	if err != nil {
		return mvc.Response{
			Object: models.Response{
				Status: models.Failure,
				Result: models.Retry,
			},
		}
	}
	var fileInfo models.Files
	fileInfo.Name = fileName
	fileInfo.Subject = subject
	_, err = c.MySQL.Get(&fileInfo)
	if err != nil {
		return mvc.Response{
			Object: models.Response{
				Status: models.Failure,
				Result: models.Retry,
			},
		}
	}
	if fileInfo.Id <= 0 {
		_, err = c.MySQL.Insert(&fileInfo)
		if err != nil {
			return mvc.Response{
				Object: models.Response{
					Status: models.Failure,
					Result: models.Retry,
				},
			}
		}
	}

	lines := strings.Split(string(content), "\n")
	//按标签归档
	for i, line := range lines {
		regParts := make([]models.LineTag, 0)
		for i, regStr := range models.Regs {
			reg := regexp.MustCompile(regStr)
			regs := reg.FindAllString(line, -1)
			for _, s := range regs {
				regParts = append(regParts, models.LineTag{
					RegExp: s,
					Type:   models.RegType(i),
				})
			}
		}
		for _, regPart := range regParts {
			data := regPart.Parse()
			data.Line = i
			data.KeyType = regPart.Type
			data.FileName = fileName
			data.Subject = subject
			_, err := c.MySQL.Insert(&data)
			if err != nil {
				return mvc.Response{
					Object: models.Response{
						Status: models.Failure,
						Result: models.Retry,
					},
				}
			}
		}
	}
	//按分块归档
	start := 0
	end := 0
	indexKey := ""
	for i, line := range lines {
		if strings.Contains(line, "/R") {
			start = i
			indexKey = line
			indexKey = strings.ReplaceAll(indexKey, "#", "")
			indexKey = strings.ReplaceAll(indexKey, " ", "")
			indexKey = strings.ReplaceAll(indexKey, "/R", "")
		}
		if strings.Contains(line, "/E") {
			end = i
			key := models.KeyIndex{
				FileName: fileName,
				Line:     start,
				Keyword:  indexKey,
				KeyType:  models.Piece,
				TagFrom:  start,
				TagTo:    end - start,
				Subject:  subject,
			}
			_, err = c.MySQL.Insert(&key)
			if err != nil {
				fmt.Println(err.Error())
			}
		}
	}
	c.Cache.FileContent[fileName] = lines
	c.Cache.Files = append(c.Cache.Files, fileInfo)
	return mvc.Response{
		Object: models.Response{
			Status: models.Success,
		},
	}
}

func (c Controller) Search() mvc.Result {
	if !c.Authorize() {
		return mvc.Response{
			Object: models.Response{
				Status: models.Failure,
				Result: models.NoPermission,
			},
		}
	}
	var searchinfo models.SearchInfo
	err := c.Context.ReadJSON(&searchinfo)
	if err != nil {
		fmt.Println(err.Error())
		return mvc.Response{
			Object: models.Response{
				Status: models.Failure,
				Result: models.Retry,
			},
		}
	}
	key := searchinfo.SearchKey
	subject := searchinfo.Subject
	typeStr := searchinfo.SearchType
	if key == "" || subject == "" || typeStr == "" {
		return mvc.Response{
			Object: models.Response{
				Status: models.Failure,
				Result: models.Retry,
			},
		}
	}
	datas := make([]models.KeyIndex, 0)
	err = c.MySQL.Where("Keyword like ? and KeyType=? and Subject=?", "%"+key+"%", typeStr, subject).Find(&datas)
	if err != nil {
		return mvc.Response{
			Object: models.Response{
				Status: models.Failure,
				Result: models.Retry,
			},
		}
	}
	contents := make([]string, 0)
	havedataReg := regexp.MustCompile("[^<>#\\n\\s]")
	for _, data := range datas {
		title := "# " + data.FileName + " " + strconv.Itoa(data.Line) + " \n "
		content := ""
		center := data.Line
		keyline := c.Cache.FileContent[data.FileName][data.Line]
		keyline = strings.ReplaceAll(keyline, "/E", "")
		keyline = strings.ReplaceAll(keyline, "/R", "")
		keyline = strings.ReplaceAll(keyline, key, " <font color=#DC143C>"+key+"</font> ")
		if data.TagFrom == data.TagTo {
			content = keyline
		} else {
			length := 0
			for i := center - 1; ; i-- {
				if length == 0-data.TagFrom {
					break
				}
				line := c.Cache.FileContent[data.FileName][i]
				lineContent := havedataReg.FindAllString(line, -1)
				if len(lineContent) <= 0 {
					continue
				}
				content = c.Cache.FileContent[data.FileName][i] + "\n" + content
				length++
			}
			content += "\n" + keyline + " \n "
			length = 0
			for i := center + 1; ; i++ {
				if length == data.TagTo {
					break
				}
				line := c.Cache.FileContent[data.FileName][i]
				lineContent := havedataReg.FindAllString(line, -1)
				if len(lineContent) <= 0 {
					continue
				}

				content += c.Cache.FileContent[data.FileName][i] + "\n"
				length++
			}
			contents = append(contents, title+content)
		}
	}
	CreateSearchFile(contents)
	return mvc.Response{
		Object: models.Response{
			Status: models.Success,
		},
	}
}

func (c Controller) FileList() mvc.Result {
	if !c.Authorize() {
		return mvc.Response{
			Object: models.Response{
				Status: models.Failure,
				Result: models.NoPermission,
			},
		}
	}
	return mvc.Response{
		Object: c.Cache.Files,
	}
}

func (c Controller) SubjectList() mvc.Result {
	if !c.Authorize() {
		return mvc.Response{
			Object: models.Response{
				Status: models.Failure,
				Result: models.NoPermission,
			},
		}
	}
	subjects := make([]models.Subject, 0)
	err := c.MySQL.Find(&subjects)
	if err != nil {
		return mvc.Response{
			Object: models.Response{
				Status: models.Failure,
				Result: models.Retry,
			},
		}
	}
	return mvc.Response{
		Object: subjects,
	}
}

func (c Controller) Remove() mvc.Result {
	if !c.Authorize() {
		return mvc.Response{
			Object: models.Response{
				Status: models.Failure,
				Result: models.NoPermission,
			},
		}
	}

	fileName := c.Context.Params().GetString("name")
	if fileName == "" {
		return mvc.Response{
			Object: models.Response{
				Status: models.Failure,
				Result: models.Retry,
			},
		}
	}
	file := models.Files{
		Name: fileName,
	}
	_, err := c.MySQL.Get(&file)
	if file.Id <= 0 {
		return mvc.Response{
			Object: models.Response{
				Status: models.Failure,
				Result: models.Retry,
			},
		}
	}
	keys := models.KeyIndex{
		FileName: fileName,
	}
	_, err = c.MySQL.Delete(&keys)
	if err != nil {
		return mvc.Response{
			Object: models.Response{
				Status: models.Failure,
				Result: models.Retry,
			},
		}
	}
	_, err = c.MySQL.Delete(&file)
	if err != nil {
		return mvc.Response{
			Object: models.Response{
				Status: models.Failure,
				Result: models.Retry,
			},
		}
	}
	c.Sync()
	return mvc.Response{
		Object: models.Response{
			Status: models.Success,
		},
	}
}

func (c Controller) Find() mvc.Result {
	if !c.Authorize() {
		return mvc.Response{
			Object: models.Response{
				Status: models.Failure,
				Result: models.NoPermission,
			},
		}
	}
	key := c.Context.Params().GetString("key")
	subject := c.Context.Params().GetString("subject")
	from, _ := c.Context.Params().GetInt("from")
	to, _ := c.Context.Params().GetInt("to")
	fileList := make([]models.Files, 0)
	for _, file := range c.Cache.Files {
		if file.Subject == subject {
			fileList = append(fileList, file)
		}
	}
	contents := make([]string, 0)
	for _, file := range fileList {
		lines := c.Cache.FileContent[file.Name]
		for i, line := range lines {
			title := "# " + file.Name + " " + strconv.Itoa(i) + " \n "
			content := ""
			content += title
			if i+to >= len(lines) {
				continue
			}
			if strings.Contains(line, key) {
				for j := from; j <= to; j++ {
					keyline := lines[j]
					keyline = strings.ReplaceAll(keyline, "/E", "")
					keyline = strings.ReplaceAll(keyline, "/R", "")
					keyline = strings.ReplaceAll(keyline, key, " <font color=#DC143C>"+key+"</font> ")
					content += keyline + " \n"
				}
				contents = append(contents, content)
			}
			i = i + to
		}
	}
	CreateSearchFile(contents)
	c.Context.Redirect("/resource/search.md")
	return mvc.Response{
		Object: models.Response{
			Status: models.Success,
		},
	}
}

func CreateSearchFile(datas []string) string {
	dataStr := ""
	for _, data := range datas {
		dataStr += data + "\n"
	}
	path, _ := os.Getwd()
	fileName := path + "/static/search.md"
	f, err := os.OpenFile(fileName, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		fmt.Println(err.Error())
		return ""
	}
	n, _ := f.Seek(0, os.SEEK_END)
	_, err = f.WriteAt([]byte(dataStr), n)
	defer f.Close()
	file, err := os.Open(fileName)
	if err != nil {
		return ""
	}
	defer file.Close()
	content, err := ioutil.ReadAll(file)
	return string(content)
}

func (c *Controller) Sync() {
	files := make([]models.Files, 0)
	cache := make(map[string][]string)
	err := c.MySQL.Find(&files)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	path, _ := os.Getwd()
	for _, fileInfo := range files {
		file, err := os.Open(path + "/resources/" + fileInfo.Name + ".md")
		if err != nil {
			return
		}
		content, err := ioutil.ReadAll(file)
		if err != nil {
			return
		}
		lines := strings.Split(string(content), "\n")
		cache[fileInfo.Name] = lines
		file.Close()
	}
	cacheData := &models.Cache{
		FileContent: cache,
		Files:       files,
	}
	c.Cache = cacheData
}

func (c Controller) GetFileList() mvc.Result {
	if !c.Authorize() {
		return mvc.Response{
			Object: models.Response{
				Status: models.Failure,
				Result: models.NoPermission,
			},
		}
	}
	subject := c.Context.Params().GetString("subject")
	files := make([]models.Files, 0)
	err := c.MySQL.Where("Subject=?", subject).Find(&files)
	if err != nil {
		fmt.Println(err.Error())
		return mvc.Response{
			Object: models.Response{
				Status: models.Failure,
				Result: models.Retry,
			},
		}
	}
	strs := make([]string, 0)
	for _, file := range files {
		data := strings.Split(file.Name, "/")
		if len(data) <= 0 {
			continue
		}
		strs = append(strs, data[len(data)-1])
	}
	return mvc.Response{
		Object: strs,
	}
}

func (c Controller) GetFileContent() mvc.Result {
	if !c.Authorize() {
		return mvc.Response{
			Object: models.Response{
				Status: models.Failure,
				Result: models.NoPermission,
			},
		}
	}
	var fileModel models.FileModel
	err := c.Context.ReadJSON(&fileModel)
	if err != nil {
		return mvc.Response{
			Object: models.Response{
				Status: models.Failure,
				Result: models.NoPermission,
			},
		}
	}
	subject := fileModel.Subject
	fileName := fileModel.FileName
	fileName = subject + "/" + fileName
	_, ok := c.Cache.FileContent[fileName]
	if !ok {
		return mvc.Response{
			Object: models.Response{
				Status: models.Failure,
				Result: models.Retry,
			},
		}
	}
	content := ""
	for _, data := range c.Cache.FileContent[fileName] {
		content += data + " \n"
	}
	return mvc.Response{
		Object: content,
	}
}
