package models

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const (
	User = "kuroba239080473"
	Pass = "xzy239080473@"
)

type LineTag struct {
	RegExp string
	Type   string
}

type SearchInfo struct {
	SearchType string `json:"type"`
	SearchKey  string `json:"key"`
	Subject    string `json:"subject"`
}

type SearchResult struct {
	FileName string `json:"file_name"`
	Start    int    `json:"start"`
	End      int    `json:"end"`
	Content  string `json:"content"`
}

func (l LineTag) Parse() KeyIndex {
	raw := l.RegExp
	model := KeyIndex{
		Keyword: "",
		TagFrom: 0,
		TagTo:   0,
	}
	dataReg := regexp.MustCompile(">(.+?)<")
	titleReg := regexp.MustCompile("title='(.+?)'")
	data := dataReg.FindAllString(raw, -1)
	first := strings.ReplaceAll(data[0], "<", "")
	second := strings.ReplaceAll(first, ">", "")
	model.Keyword = second

	title := titleReg.FindAllString(raw, -1)
	if len(title) < 1 {
		fmt.Println(raw)
		return model
	}
	first = strings.ReplaceAll(title[0], "title=", "")
	first = strings.ReplaceAll(first, " ", "")
	second = strings.ReplaceAll(first, "'", "")
	fromto := strings.Split(second, ",")
	if len(fromto) < 2 {
		fmt.Println(raw)
	} else {
		from, _ := strconv.Atoi(fromto[0])
		to, _ := strconv.Atoi(fromto[1])
		model.TagFrom = from
		model.TagTo = to
	}
	return model
}

var Regs = []string{
	"<font color=#3CB371 title=(.+?)>(.+?)</font>",
	"<font color=#239080 title=(.+?)>(.+?)</font>",
	"<font color=#800000 title=(.+?)>(.+?)</font>",
	"<font color=#FFA07A title=(.+?)>(.+?)</font>",
	"<font color=#FF0000 title=(.+?)>(.+?)</font>",
	"<font color=#FFA500 title=(.+?)>(.+?)</font>",
	"<font color=#265459 title=(.+?)>(.+?)</font>",
}

const Piece = "pie"

func RegType(index int) string {
	switch index {
	case 0:
		return "def"
	case 1:
		return "key"
	case 2:
		return "code"
	case 3:
		return "func"
	case 4:
		return "imp"
	case 5:
		return "ind"
	case 6:
		return "tabl"

	}
	return ""
}

const UserID = "UserID"
const Success = "success"
const Failure = "failure"
const Retry = "retry please"
const PasswdError = "密码错误"
const NoPermission = "无权操作"

type Response struct {
	Status string      `json:"status"`
	Result interface{} `json:"result"`
}

type LoginInfo struct {
	UserID   string `json:"userid"`
	Password string `json:"password"`
}

type Cache struct {
	FileContent map[string][]string
	Files       []Files
}
