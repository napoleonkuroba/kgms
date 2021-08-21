package models

type Files struct {
	Id		int64
	Name	string
	Subject string
}

type Subject struct {
	Id 		int64
	Name 	string
}

type KeyIndex struct {
	Id 		int64
	FileName string
	Line 	int
	Keyword	string
	KeyType	string
	TagFrom	int
	TagTo	int
	Subject string
}
