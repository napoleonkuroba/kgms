package controllers

import (
	"fmt"
	"github.com/go-xorm/xorm"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/mvc"
	"github.com/kataras/iris/v12/sessions"
	"io/ioutil"
	"knowledge-graph-management-system/models"
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
	b.Handle("POST", "/", "Login")
	b.Handle("GET", "/ParseFile/{passwd}/{name}/{subject}", "Prase")
	b.Handle("GET", "/Search/{passwd}/{subject}/{type}/{key}", "Search")
	b.Handle("GET", "/FileList/{passwd}", "FileList")
	b.Handle("GET", "/SubjectList/{passwd}", "SubjectList")
	b.Handle("GET", "/RemoveFile/{passwd}/{name}", "Remove")
}

func (c Controller) IsLogin() mvc.Result {
	login := models.Failure
	userID := c.Session.GetString(models.UserID)
	if userID != "" {
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
		c.Session.Set(models.UserID, loginInfo.UserID)
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
	passwd := c.Context.Params().GetString("passwd")
	if passwd != models.User {
		return mvc.Response{
			Object: models.Response{
				Status: models.Failure,
				Result: models.NoPermission,
			},
		}
	}

	path, _ := os.Getwd()
	fileName := c.Context.Params().GetString("name")
	subject := c.Context.Params().GetString("subject")
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
	c.Cache.FileContent[fileName] = lines
	c.Cache.Files = append(c.Cache.Files, fileInfo)
	return mvc.Response{
		Object: models.Response{
			Status: models.Success,
		},
	}
}

func (c Controller) Search() mvc.Result {
	passwd := c.Context.Params().GetString("passwd")
	if passwd != models.User {
		return mvc.Response{
			Object: models.Response{
				Status: models.Failure,
				Result: models.NoPermission,
			},
		}
	}

	key := c.Context.Params().GetString("key")
	subject := c.Context.Params().GetString("subject")
	typeStr := c.Context.Params().GetString("type")
	if key == "" || subject == "" || typeStr == "" {
		return mvc.Response{
			Object: models.Response{
				Status: models.Failure,
				Result: models.Retry,
			},
		}
	}
	datas := make([]models.KeyIndex, 0)
	err := c.MySQL.Where("Keyword like ? and KeyType=? and Subject=?", "%"+key+"%", typeStr, subject).Find(&datas)
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
		if data.TagFrom == data.TagTo {
			content = c.Cache.FileContent[data.FileName][data.Line]
		} else {
			length := 0
			for i := center; ; i-- {
				if length == 1+(0-data.TagFrom) {
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
			content += "\n"
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
				fmt.Println(line, len(lineContent))
				content += c.Cache.FileContent[data.FileName][i] + "\n"
				length++
			}
			contents = append(contents, title+content)
		}
		CreateSearchFile(contents)
		c.Context.Redirect("/resource/search.md")
	}
	return mvc.Response{
		Object: models.Response{
			Status: models.Success,
		},
	}
}

func (c Controller) FileList() mvc.Result {
	passwd := c.Context.Params().GetString("passwd")
	if passwd != models.User {
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
	passwd := c.Context.Params().GetString("passwd")
	if passwd != models.User {
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
	passwd := c.Context.Params().GetString("passwd")
	if passwd != models.User {
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
